package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/innovatex-tech/commercex-provisioner/internal/cli"
	"github.com/innovatex-tech/commercex-provisioner/internal/core"
	"github.com/innovatex-tech/commercex-provisioner/internal/db"
	"github.com/innovatex-tech/commercex-provisioner/internal/deploy"
	"github.com/innovatex-tech/commercex-provisioner/internal/registry"
	"github.com/innovatex-tech/commercex-provisioner/internal/tui"
	"github.com/spf13/cobra"
)

const Version = "1.0.0"

// ─── Config ───────────────────────────────────────────────────────────────────

func getWorkDir() string {
	if dir := os.Getenv("INNOVATEX_WORK_DIR"); dir != "" {
		return dir
	}
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".innovatex", "clients")
}

func getTemplateDir() string {
	if dir := os.Getenv("INNOVATEX_TEMPLATE_DIR"); dir != "" {
		return dir
	}
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".innovatex", "templates")
}

func getRegistryPath() string {
	if path := os.Getenv("INNOVATEX_REGISTRY"); path != "" {
		return path
	}
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".innovatex", "registry.json")
}

func getConfig() *core.Config {
	cfg := &core.Config{
		WorkDir:        getWorkDir(),
		TemplateDir:    getTemplateDir(),
		StorefrontRepo: envOrDefault("INNOVATEX_STOREFRONT_REPO", "https://github.com/The-Coding-Kiddo/clothing-storefront.git"),
		DBHost:         envOrDefault("INNOVATEX_DB_HOST", "localhost"),
		DBPort:         6543,
		DBUser:         envOrDefault("INNOVATEX_DB_USER", "vendure"),
		DBPassword:     os.Getenv("INNOVATEX_DB_PASSWORD"),
		AdminDB:        "vendure",
		BasePort:       8000,
	}
	if p := os.Getenv("INNOVATEX_DB_PORT"); p != "" {
		fmt.Sscanf(p, "%d", &cfg.DBPort)
	}
	if p := os.Getenv("INNOVATEX_BASE_PORT"); p != "" {
		fmt.Sscanf(p, "%d", &cfg.BasePort)
	}
	return cfg
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// ─── Template Bootstrap ──────────────────────────────────────────────────────

func ensureTemplates() error {
	templateDir := getTemplateDir()
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		return err
	}

	templates := map[string]string{
		".env.tmpl": `# CommerceX Environment Configuration
# Client: {{.ClientID}}
# Generated: {{.GeneratedAt}}

APP_ENV=production
PORT=3000
COOKIE_SECRET={{.CookieSecret}}
SUPERADMIN_USERNAME={{.AdminUsername}}
SUPERADMIN_PASSWORD={{.AdminPassword}}
DB_HOST=postgres_db
DB_PORT=5432
DB_NAME={{.DBName}}
DB_SCHEMA=public
DB_USERNAME={{.DBUsername}}
DB_PASSWORD={{.DBPassword}}
POSTGRES_PORT={{.PostgresPort}}
ENABLE_SSL=false
VITE_API_HOST=http://localhost
VITE_API_PORT={{.AppPort}}
`,
		"docker-compose.yml.tmpl": `# CommerceX Provisioner - Client: {{.ClientID}}
services:
    nginx:
        image: nginx:alpine
        container_name: nginx_{{.ClientID}}
        ports:
            - "{{.AppPort}}:80"
        volumes:
            - ./nginx.conf:/etc/nginx/conf.d/default.conf:ro
        depends_on:
            - commercex-server
        restart: unless-stopped
        networks:
            - {{.ClientID}}_network

    commercex-server:
        image: ${REGISTRY:-abduazizali}/commercex:${TAG:-latest}
        container_name: commercex_server_{{.ClientID}}
        env_file:
            - .env
        volumes:
            - commercex_static_{{.ClientID}}:/app/static
        depends_on:
            postgres_db:
                condition: service_healthy
        command: ["node", "dist/index.js"]
        restart: unless-stopped
        networks:
            - {{.ClientID}}_network

    commercex-worker:
        image: ${REGISTRY:-abduazizali}/commercex:${TAG:-latest}
        container_name: commercex_worker_{{.ClientID}}
        env_file:
            - .env
        volumes:
            - commercex_static_{{.ClientID}}:/app/static
        depends_on:
            postgres_db:
                condition: service_healthy
        command: ["node", "dist/index-worker.js"]
        restart: unless-stopped
        networks:
            - {{.ClientID}}_network

    postgres_db:
        image: postgres:16-alpine
        container_name: postgres_{{.ClientID}}
        volumes:
            - postgres_data_{{.ClientID}}:/var/lib/postgresql/data
        ports:
            - "{{.PostgresPort}}:5432"
        environment:
            POSTGRES_DB: {{.DBName}}
            POSTGRES_USER: {{.DBUsername}}
            POSTGRES_PASSWORD: {{.DBPassword}}
        healthcheck:
            test: ["CMD-SHELL", "pg_isready -U {{.DBUsername}}"]
            interval: 5s
            timeout: 5s
            retries: 5
        networks:
            - {{.ClientID}}_network

    storefront:
        build:
            context: ./storefront
        container_name: storefront_{{.ClientID}}
        ports:
            - "{{.StorefrontPort}}:80"
        depends_on:
            - commercex-server
        networks:
            - {{.ClientID}}_network

volumes:
    postgres_data_{{.ClientID}}:
        driver: local
    commercex_static_{{.ClientID}}:
        driver: local

networks:
    {{.ClientID}}_network:
        driver: bridge
`,
		"nginx.conf.tmpl": `# Nginx for {{.ClientID}} — {{.Domain}}
server {
    listen 80;
    server_name {{.Domain}} www.{{.Domain}} localhost;
    client_max_body_size 100M;
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    gzip on;
    gzip_vary on;
    gzip_types text/plain text/css application/json application/javascript text/xml application/xml text/javascript;
    location / {
        proxy_pass http://commercex-server:3000;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
    }
}
`,
	}

	for name, content := range templates {
		path := filepath.Join(templateDir, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := os.WriteFile(path, []byte(content), 0644); err != nil {
				return fmt.Errorf("writing template %s: %w", name, err)
			}
		}
	}
	return nil
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

var (
	reClientID = regexp.MustCompile(`^[a-z0-9-]{3,}$`)
	reDomain   = regexp.MustCompile(`^[a-zA-Z0-9.-]{3,}$`)
	reDBName   = regexp.MustCompile(`^[a-zA-Z0-9_]{3,}$`)
	reUsername = regexp.MustCompile(`^[a-zA-Z0-9_]{2,}$`)
)

func errf(code, msg string, retryable ...bool) *cli.CLIError {
	r := false
	if len(retryable) > 0 {
		r = retryable[0]
	}
	return &cli.CLIError{Code: code, Message: msg, Retryable: r}
}

func validateClientID(s string) error {
	if !reClientID.MatchString(s) {
		return fmt.Errorf("client ID must be 3+ lowercase alphanumeric/dash characters")
	}
	return nil
}

func validateDomain(s string) error {
	if !reDomain.MatchString(s) {
		return fmt.Errorf("invalid domain format")
	}
	return nil
}

func validateDBName(s string) error {
	if !reDBName.MatchString(s) {
		return fmt.Errorf("database name must be 3+ alphanumeric/underscore characters")
	}
	return nil
}

func validateUsername(s string) error {
	if !reUsername.MatchString(s) {
		return fmt.Errorf("username must be 2+ alphanumeric/underscore characters")
	}
	return nil
}

func validatePassword(s string) error {
	if len(s) < 6 {
		return fmt.Errorf("password must be at least 6 characters")
	}
	return nil
}

// ─── Interactive Prompt ───────────────────────────────────────────────────────

func promptInput(scanner *bufio.Scanner, prompt string, validator func(string) error) string {
	for {
		fmt.Fprintf(os.Stderr, "  > %s: ", prompt)
		scanner.Scan()
		input := strings.TrimSpace(scanner.Text())
		if err := validator(input); err != nil {
			fmt.Fprintf(os.Stderr, "    error: %s\n", err.Error())
			continue
		}
		return input
	}
}

func promptInputWithDefault(scanner *bufio.Scanner, prompt, defaultValue string, validator func(string) error) string {
	fmt.Fprintf(os.Stderr, "  > %s [%s]: ", prompt, defaultValue)
	scanner.Scan()
	input := strings.TrimSpace(scanner.Text())
	if input == "" {
		return defaultValue
	}
	if err := validator(input); err != nil {
		fmt.Fprintf(os.Stderr, "    error: %s, using default: %s\n", err.Error(), defaultValue)
		return defaultValue
	}
	return input
}

// ─── Root Command ─────────────────────────────────────────────────────────────

func main() {
	if err := ensureTemplates(); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing templates: %v\n", err)
		os.Exit(cli.ExitRuntime)
	}

	var formatFlag string

	rootCmd := &cobra.Command{
		Use:   "innovatex",
		Short: "Multi-tenant e-commerce provisioner",
		Long:  "Provision, manage, and monitor isolated CommerceX environments.",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Disable pagers when not a TTY
			if !cli.IsTTY(os.Stdout) {
				os.Setenv("Pager", "")
			}
		},
	}

	rootCmd.PersistentFlags().StringVar(&formatFlag, "format", "", "Output format: json, table (default: auto-detect TTY)")

	rootCmd.AddCommand(createCmd(&formatFlag))
	rootCmd.AddCommand(listCmd(&formatFlag))
	rootCmd.AddCommand(deleteCmd(&formatFlag))
	rootCmd.AddCommand(statusCmd(&formatFlag))
	rootCmd.AddCommand(startCmd(&formatFlag))
	rootCmd.AddCommand(stopCmd(&formatFlag))
	rootCmd.AddCommand(logsCmd(&formatFlag))
	rootCmd.AddCommand(dashboardCmd())

	if err := rootCmd.Execute(); err != nil {
		if cliErr, ok := err.(*cli.CLIError); ok {
			w := cli.NewWriter(cli.ParseFormat(formatFlag))
			w.Error(cliErr.Code, cliErr.Message, cliErr.Retryable)
			os.Exit(cliErr.ExitCode())
		}
		w := cli.NewWriter(cli.DetectFormat())
		code := w.Error("runtime_error", err.Error(), false)
		os.Exit(code.ExitCode())
	}
}

// ─── Commands ─────────────────────────────────────────────────────────────────

func createCmd(formatFlag *string) *cobra.Command {
	var clientID, domain, brandName string
	var dbName, dbUsername, dbPassword string
	var adminUsername, adminPassword string
	var yes bool

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new commerce client",
		Long:  "Provision a new isolated e-commerce environment. In non-interactive mode (piped stdin), all flags are required.",
		RunE: func(cmd *cobra.Command, args []string) error {
			format := cli.ParseFormat(*formatFlag)
			w := cli.NewWriter(format)
			cfg := getConfig()

			interactive := cli.IsTTY(os.Stdin)

			// Interactive prompts when TTY and flags missing
			if interactive && !yes {
				scanner := bufio.NewScanner(os.Stdin)
				fmt.Fprintln(os.Stderr, "\nCommerceX Provisioner — Create Client")

				if clientID == "" {
					clientID = promptInput(scanner, "Client ID (lowercase, alphanumeric, dashes)", validateClientID)
				}
				if domain == "" {
					domain = promptInput(scanner, "Domain or IP (e.g. localhost, mystore.com)", validateDomain)
				}
				if brandName == "" {
					brandName = promptInput(scanner, "Brand Name", func(s string) error {
						if len(s) < 2 {
							return fmt.Errorf("brand name must be at least 2 characters")
						}
						return nil
					})
				}
				if dbName == "" {
					dbName = promptInput(scanner, "Database Name", validateDBName)
				}
				if dbUsername == "" {
					dbUsername = promptInputWithDefault(scanner, "Database Username", "vendure", validateUsername)
				}
				if dbPassword == "" {
					dbPassword = promptInput(scanner, "Database Password (min 6 chars)", validatePassword)
				}
				if adminUsername == "" {
					adminUsername = promptInput(scanner, "Admin Username", validateUsername)
				}
				if adminPassword == "" {
					adminPassword = promptInput(scanner, "Admin Password (min 6 chars)", validatePassword)
				}
			}

			// Validate all required flags are present
			missing := []string{}
			if clientID == "" {
				missing = append(missing, "--id")
			}
			if domain == "" {
				missing = append(missing, "--domain")
			}
			if brandName == "" {
				missing = append(missing, "--brand")
			}
			if dbName == "" {
				missing = append(missing, "--db-name")
			}
			if dbPassword == "" {
				missing = append(missing, "--db-password")
			}
			if adminUsername == "" {
				missing = append(missing, "--admin-user")
			}
			if adminPassword == "" {
				missing = append(missing, "--admin-password")
			}

			if len(missing) > 0 {
				if interactive && !yes {
					return fmt.Errorf("all fields are required in non-interactive mode")
				}
				return &cli.CLIError{Code: "validation_error", Message: fmt.Sprintf("missing required flags: %s", strings.Join(missing, ", "))}
			}

			// Validate inputs
			for _, v := range []struct {
				name string
				fn   func(string) error
				val  string
			}{
				{"--id", validateClientID, clientID},
				{"--domain", validateDomain, domain},
				{"--db-name", validateDBName, dbName},
				{"--admin-user", validateUsername, adminUsername},
				{"--db-password", validatePassword, dbPassword},
				{"--admin-password", validatePassword, adminPassword},
			} {
				if err := v.fn(v.val); err != nil {
					return &cli.CLIError{Code: "validation_error", Message: fmt.Sprintf("%s: %v", v.name, err)}
				}
			}

			if dbUsername == "" {
				dbUsername = "vendure"
			}

			reg := registry.NewStore(getRegistryPath())
			dbProv := db.NewProvisioner(cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.AdminDB)
			prov := core.NewProvisioner(cfg, reg, dbProv)

			client, err := prov.Create(&core.CreateRequest{
				ClientID:      clientID,
				Domain:        domain,
				BrandName:     brandName,
				DBName:        dbName,
				DBUsername:    dbUsername,
				DBPassword:    dbPassword,
				AdminUsername: adminUsername,
				AdminPassword: adminPassword,
			}, nil)
			if err != nil {
				return errf("runtime_error", err.Error(), true)
			}

			w.Success(map[string]interface{}{
				"id":             client.ID,
				"brand":          client.BrandName,
				"domain":         client.Domain,
				"db_name":        client.DBName,
				"app_port":       client.AppPort,
				"storefront_port": client.StorefrontPort,
				"postgres_port":  client.PostgresPort,
				"admin_user":     client.AdminUsername,
				"admin_pass":     client.AdminPassword,
			})
			return nil
		},
	}

	cmd.Flags().StringVarP(&clientID, "id", "i", "", "Client ID (required in non-interactive mode)")
	cmd.Flags().StringVarP(&domain, "domain", "d", "", "Domain or IP (required in non-interactive mode)")
	cmd.Flags().StringVarP(&brandName, "brand", "b", "", "Brand name (required in non-interactive mode)")
	cmd.Flags().StringVar(&dbName, "db-name", "", "Database name")
	cmd.Flags().StringVar(&dbUsername, "db-user", "", "Database username (default: vendure)")
	cmd.Flags().StringVar(&dbPassword, "db-password", "", "Database password")
	cmd.Flags().StringVar(&adminUsername, "admin-user", "", "Admin username (required in non-interactive mode)")
	cmd.Flags().StringVar(&adminPassword, "admin-password", "", "Admin password (required in non-interactive mode)")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip interactive prompts (non-interactive mode)")

	return cmd
}

func listCmd(formatFlag *string) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all provisioned clients",
		RunE: func(cmd *cobra.Command, args []string) error {
			format := cli.ParseFormat(*formatFlag)
			w := cli.NewWriter(format)

			reg := registry.NewStore(getRegistryPath())
			clients, err := reg.List()
			if err != nil {
				return errf("runtime_error", err.Error(), true)
			}

			type clientRow struct {
				ID             string `json:"id"`
				Brand          string `json:"brand"`
				Status         string `json:"status"`
				AppPort        int    `json:"app_port"`
				StorefrontPort int    `json:"storefront_port"`
				PostgresPort   int    `json:"postgres_port"`
			}

			rows := make([]clientRow, len(clients))
			for i, c := range clients {
				rows[i] = clientRow{
					ID: c.ID, Brand: c.BrandName, Status: c.Status,
					AppPort: c.AppPort, StorefrontPort: c.StorefrontPort, PostgresPort: c.PostgresPort,
				}
			}

			w.Success(map[string]interface{}{
				"clients": rows,
				"count":   len(rows),
			})
			return nil
		},
	}
}

func deleteCmd(formatFlag *string) *cobra.Command {
	var clientID string
	var yes bool

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a client and its containers",
		Long:  "Permanently remove a client's containers, work directory, and registry entry. Requires --yes or interactive confirmation.",
		RunE: func(cmd *cobra.Command, args []string) error {
			format := cli.ParseFormat(*formatFlag)
			w := cli.NewWriter(format)

			if len(args) > 0 {
				clientID = args[0]
			}

			if clientID == "" {
				return errf("validation_error", "client ID is required (pass --id or as positional argument)", false)
			}

			// Confirmation gate
			interactive := cli.IsTTY(os.Stdin)
			if !yes {
				if interactive {
					fmt.Fprintf(os.Stderr, "Delete client %s? This is irreversible. [y/N]: ", clientID)
					scanner := bufio.NewScanner(os.Stdin)
					scanner.Scan()
					answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
					if answer != "y" && answer != "yes" {
						w.Diag("Aborted.")
						return nil
					}
				} else {
					return errf("confirmation_required", "pass --yes to confirm deletion", false)
				}
			}

			reg := registry.NewStore(getRegistryPath())
			deployer := deploy.NewDockerDeployer(getWorkDir())

			if err := deployer.Stop(clientID); err != nil {
				return errf("runtime_error", err.Error(), true)
			}

			if err := reg.Delete(clientID); err != nil {
				return errf("runtime_error", err.Error(), true)
			}

			w.Success(map[string]interface{}{
				"id":     clientID,
				"status": "deleted",
			})
			return nil
		},
	}

	cmd.Flags().StringVarP(&clientID, "id", "i", "", "Client ID")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation")

	return cmd
}

func statusCmd(formatFlag *string) *cobra.Command {
	var clientID string

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show client details and connection info",
		RunE: func(cmd *cobra.Command, args []string) error {
			format := cli.ParseFormat(*formatFlag)
			w := cli.NewWriter(format)

			if len(args) > 0 {
				clientID = args[0]
			}
			if clientID == "" {
				return errf("validation_error", "client ID is required (pass --id or as positional argument)", false)
			}

			reg := registry.NewStore(getRegistryPath())
			client, err := reg.Get(clientID)
			if err != nil {
				return errf("not_found", fmt.Sprintf("client %q not found", clientID), false)
			}

			w.Success(map[string]interface{}{
				"id":              client.ID,
				"brand":           client.BrandName,
				"status":          client.Status,
				"domain":          client.Domain,
				"db_name":         client.DBName,
				"app_port":        client.AppPort,
				"storefront_port": client.StorefrontPort,
				"postgres_port":   client.PostgresPort,
				"admin_user":      client.AdminUsername,
				"admin_pass":      client.AdminPassword,
				"created_at":      client.CreatedAt,
			})
			return nil
		},
	}

	cmd.Flags().StringVarP(&clientID, "id", "i", "", "Client ID")
	return cmd
}

func startCmd(formatFlag *string) *cobra.Command {
	var clientID string

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start a stopped client's containers",
		RunE: func(cmd *cobra.Command, args []string) error {
			format := cli.ParseFormat(*formatFlag)
			w := cli.NewWriter(format)

			if len(args) > 0 {
				clientID = args[0]
			}
			if clientID == "" {
				return errf("validation_error", "client ID is required", false)
			}

			reg := registry.NewStore(getRegistryPath())
			if _, err := reg.Get(clientID); err != nil {
				return errf("not_found", fmt.Sprintf("client %q not found", clientID), false)
			}

			deployer := deploy.NewDockerDeployer(getWorkDir())
			w.Diag("Starting %s...", clientID)
			if err := deployer.Start(clientID); err != nil {
				return errf("runtime_error", err.Error(), true)
			}

			w.Success(map[string]interface{}{
				"id":     clientID,
				"status": "started",
			})
			return nil
		},
	}

	cmd.Flags().StringVarP(&clientID, "id", "i", "", "Client ID")
	return cmd
}

func stopCmd(formatFlag *string) *cobra.Command {
	var clientID string

	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop a running client's containers",
		RunE: func(cmd *cobra.Command, args []string) error {
			format := cli.ParseFormat(*formatFlag)
			w := cli.NewWriter(format)

			if len(args) > 0 {
				clientID = args[0]
			}
			if clientID == "" {
				return errf("validation_error", "client ID is required", false)
			}

			reg := registry.NewStore(getRegistryPath())
			if _, err := reg.Get(clientID); err != nil {
				return errf("not_found", fmt.Sprintf("client %q not found", clientID), false)
			}

			deployer := deploy.NewDockerDeployer(getWorkDir())
			w.Diag("Stopping %s...", clientID)
			if err := deployer.Stop(clientID); err != nil {
				return errf("runtime_error", err.Error(), true)
			}

			w.Success(map[string]interface{}{
				"id":     clientID,
				"status": "stopped",
			})
			return nil
		},
	}

	cmd.Flags().StringVarP(&clientID, "id", "i", "", "Client ID")
	return cmd
}

func logsCmd(formatFlag *string) *cobra.Command {
	var clientID, service string
	var tail int
	var follow bool

	cmd := &cobra.Command{
		Use:   "logs",
		Short: "View container logs for a client",
		RunE: func(cmd *cobra.Command, args []string) error {
			format := cli.ParseFormat(*formatFlag)

			if len(args) > 0 {
				clientID = args[0]
			}
			if clientID == "" {
				return errf("validation_error", "client ID is required", false)
			}

			reg := registry.NewStore(getRegistryPath())
			if _, err := reg.Get(clientID); err != nil {
				return errf("not_found", fmt.Sprintf("client %q not found", clientID), false)
			}

			deployer := deploy.NewDockerDeployer(getWorkDir())

			// In JSON mode, capture logs and return as data
			if format == cli.FormatJSON {
				// For JSON mode, we still pass through to docker (logs are inherently streaming)
				return deployer.Logs(clientID, service, tail, follow)
			}

			return deployer.Logs(clientID, service, tail, follow)
		},
	}

	cmd.Flags().StringVarP(&clientID, "id", "i", "", "Client ID")
	cmd.Flags().StringVarP(&service, "service", "s", "", "Filter by service name")
	cmd.Flags().IntVar(&tail, "tail", 100, "Number of lines from the end")
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log output")
	return cmd
}

func dashboardCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "dashboard",
		Short: "Open interactive TUI dashboard",
		Long:  "Launch a live terminal dashboard for managing all clients.",
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := registry.NewStore(getRegistryPath())
			dashboard := tui.NewDashboard(getWorkDir(), reg)

			p := tea.NewProgram(
				dashboard,
				tea.WithAltScreen(),
				tea.WithMouseCellMotion(),
			)

			if _, err := p.Run(); err != nil {
				return fmt.Errorf("dashboard error: %v", err)
			}
			return nil
		},
	}
}
