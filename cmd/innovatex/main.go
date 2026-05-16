package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/innovatex-tech/commercex-provisioner/internal/core"
	"github.com/innovatex-tech/commercex-provisioner/internal/db"
	"github.com/innovatex-tech/commercex-provisioner/internal/deploy"
	"github.com/innovatex-tech/commercex-provisioner/internal/registry"
	"github.com/innovatex-tech/commercex-provisioner/internal/tui"
	"github.com/spf13/cobra"
)

const Version = "1.0.0"

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

func ensureTemplates() error {
	templateDir := getTemplateDir()

	// Create template directory if it doesn't exist
	if err := os.MkdirAll(templateDir, 0755); err != nil {
		return err
	}

	// Check if templates exist, if not create them
	envTemplate := filepath.Join(templateDir, ".env.tmpl")
	if _, err := os.Stat(envTemplate); os.IsNotExist(err) {
		// Create .env.tmpl
		envContent := `# CommerceX Environment Configuration
# Client: {{.ClientID}}
# Generated: {{.GeneratedAt}}

# Application Configuration
APP_ENV=production
PORT=3000

# Authentication
COOKIE_SECRET={{.CookieSecret}}
SUPERADMIN_USERNAME={{.AdminUsername}}
SUPERADMIN_PASSWORD={{.AdminPassword}}

# Database Configuration
DB_HOST=postgres_db
DB_PORT=5432
DB_NAME={{.DBName}}
DB_SCHEMA=public
DB_USERNAME={{.DBUsername}}
DB_PASSWORD={{.DBPassword}}

# PostgreSQL External Port (host machine)
POSTGRES_PORT={{.PostgresPort}}

# Optional: Enable SSL for database connection
ENABLE_SSL=false

# Vite Build Configuration (for dashboard schema introspection)
VITE_API_HOST=http://localhost
VITE_API_PORT={{.AppPort}}
`
		if err := os.WriteFile(envTemplate, []byte(envContent), 0644); err != nil {
			return err
		}
	}

	// Check and create docker-compose.yml.tmpl
	composeTemplate := filepath.Join(templateDir, "docker-compose.yml.tmpl")
	if _, err := os.Stat(composeTemplate); os.IsNotExist(err) {
		composeContent := `# Production Docker Compose - CommerceX Provisioner
# Client: {{.ClientID}}

services:
    # Nginx Reverse Proxy
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

    # CommerceX Application Server
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

    # CommerceX Background Worker
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

    # Storefront
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
`
		if err := os.WriteFile(composeTemplate, []byte(composeContent), 0644); err != nil {
			return err
		}
	}

	// Check and create nginx.conf.tmpl
	nginxTemplate := filepath.Join(templateDir, "nginx.conf.tmpl")
	if _, err := os.Stat(nginxTemplate); os.IsNotExist(err) {
		nginxContent := `# Nginx Configuration for {{.ClientID}}
# Domain: {{.Domain}}

server {
    listen 80;
    server_name {{.Domain}} www.{{.Domain}} localhost;
    
    # Increase upload size for product images
    client_max_body_size 100M;
    
    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    
    # Compression
    gzip on;
    gzip_vary on;
    gzip_types text/plain text/css application/json application/javascript text/xml application/xml text/javascript;
    
    # Proxy to CommerceX backend (internal port 3000)
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
`
		if err := os.WriteFile(nginxTemplate, []byte(nginxContent), 0644); err != nil {
			return err
		}
	}

	return nil
}

var config = &core.Config{
	WorkDir:        getWorkDir(),
	TemplateDir:    getTemplateDir(),
	StorefrontRepo: "https://github.com/The-Coding-Kiddo/clothing-storefront.git",
	DBHost:         "localhost",
	DBPort:         6543,
	DBUser:         "vendure",
	DBPassword:     "XTE9YTewFVAY2hvXK9-MUg",
	AdminDB:        "vendure",
	BasePort:       8000,
}

func main() {
	// Ensure templates exist on first run
	if err := ensureTemplates(); err != nil {
		fmt.Printf("Error initializing templates: %v\n", err)
		os.Exit(1)
	}

	rootCmd := &cobra.Command{
		Use:     "innovatex",
		Short:   "InnovateX multi-client e-commerce provisioner",
		Version: Version,
	}

	rootCmd.AddCommand(createCmd())
	rootCmd.AddCommand(listCmd())
	rootCmd.AddCommand(deleteCmd())
	rootCmd.AddCommand(statusCmd())
	rootCmd.AddCommand(startCmd())
	rootCmd.AddCommand(stopCmd())
	rootCmd.AddCommand(logsCmd())
	rootCmd.AddCommand(dashboardCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func createCmd() *cobra.Command {
	var clientID, domain, brandName string
	var dbName, dbUsername, dbPassword string
	var adminUsername, adminPassword string
	var server, sshPass, sshKey string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new commerce client",
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := registry.NewStore(getRegistryPath())
			dbProv := db.NewProvisioner(config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.AdminDB)
			prov := core.NewProvisioner(config, reg, dbProv)

			var req core.CreateRequest

			// If flags are provided, use them. Otherwise launch Wizard.
			if clientID != "" && domain != "" && brandName != "" {
				var serverUser, serverHost string
				if server != "" {
					parts := strings.Split(server, "@")
					if len(parts) == 2 {
						serverUser, serverHost = parts[0], parts[1]
					} else {
						serverUser, serverHost = "root", server
					}
				}

				req = core.CreateRequest{
					ClientID:      clientID,
					Domain:        domain,
					BrandName:     brandName,
					DBName:        dbName,
					DBUsername:    dbUsername,
					DBPassword:    dbPassword,
					AdminUsername: adminUsername,
					AdminPassword: adminPassword,
					ServerHost:    serverHost,
					ServerUser:    serverUser,
					SSHPassword:   sshPass,
					SSHKeyPath:    sshKey,
				}
			} else {
				m := tui.NewWizard()
				p := tea.NewProgram(m, tea.WithAltScreen())
				finalModel, err := p.Run()
				if err != nil {
					return err
				}
				wizard := finalModel.(tui.WizardModel)
				var confirmed bool
				req, confirmed = wizard.GetRequest()
				if !confirmed {
					fmt.Println("\n❌ Creation cancelled.")
					return nil
				}
			}

			fmt.Println("\n🚀  PROVISIONING STARTING...")
			client, err := prov.Create(&req)
			if err != nil {
				return err
			}

			fmt.Println("\n╔══════════════════════════════════════════════════════════╗")
			fmt.Println("║                                                          ║")
			fmt.Println("║           ✅  DEPLOYMENT SUCCESSFUL!  ✅                ║")
			fmt.Println("║                                                          ║")
			fmt.Println("╚══════════════════════════════════════════════════════════╝")
			fmt.Println()

			// Client info box
			fmt.Println("┌─ 📦 Client Information ─────────────────────────────────┐")
			fmt.Printf("│  Client ID:    %-42s │\n", client.ID)
			fmt.Printf("│  Brand:        %-42s │\n", client.BrandName)
			fmt.Printf("│  Database:     %-42s │\n", client.DBName)
			fmt.Println("└──────────────────────────────────────────────────────────┘")

			// Access URLs
			urlTarget := client.Domain
			if client.IsRemote {
				urlTarget = client.ServerHost
			}

			fmt.Println("\n┌─ 🌐 Access URLs ────────────────────────────────────────┐")
			fmt.Printf("│  🛍️  Storefront:   http://%-29s │\n", fmt.Sprintf("%s:%d", urlTarget, client.StorefrontPort))
			fmt.Printf("│  🔧 CommerceX:     http://%-29s │\n", fmt.Sprintf("%s:%d", urlTarget, client.AppPort))
			fmt.Printf("│  🗄️  PostgreSQL:   %-38s │\n", fmt.Sprintf("%s:%d", urlTarget, client.PostgresPort))
			fmt.Println("└──────────────────────────────────────────────────────────┘")

			// Admin credentials box
			fmt.Println("\n┌─ 🔐 Admin Credentials ──────────────────────────────────┐")
			fmt.Printf("│  Username:     %-42s │\n", client.AdminUsername)
			fmt.Printf("│  Password:     %-42s │\n", client.AdminPassword)
			fmt.Println("└──────────────────────────────────────────────────────────┘")

			fmt.Println("\n💡 Next Step: Run 'innovatex dashboard' to manage your new client.")
			fmt.Println()

			return nil
		},
	}

	// Optional flags
	cmd.Flags().StringVarP(&clientID, "id", "i", "", "Client ID")
	cmd.Flags().StringVarP(&domain, "domain", "d", "", "Domain")
	cmd.Flags().StringVarP(&brandName, "brand", "b", "", "Brand name")
	cmd.Flags().StringVar(&dbName, "db-name", "", "Database name")
	cmd.Flags().StringVar(&dbUsername, "db-user", "", "Database username")
	cmd.Flags().StringVar(&dbPassword, "db-password", "", "Database password")
	cmd.Flags().StringVar(&adminUsername, "admin-user", "", "Admin username")
	cmd.Flags().StringVar(&adminPassword, "admin-password", "", "Admin password")
	cmd.Flags().StringVar(&server, "server", "", "Remote server (user@host)")
	cmd.Flags().StringVar(&sshPass, "ssh-pass", "", "SSH password")
	cmd.Flags().StringVar(&sshKey, "ssh-key", "", "SSH private key path")

	return cmd
}

// Validation functions
func validateClientID(input string) error {
	if len(input) < 3 {
		return fmt.Errorf("client ID must be at least 3 characters")
	}
	matched, _ := regexp.MatchString("^[a-z0-9-]+$", input)
	if !matched {
		return fmt.Errorf("client ID must contain only lowercase letters, numbers, and dashes")
	}
	return nil
}

func validateDomain(input string) error {
	if len(input) < 3 {
		return fmt.Errorf("domain must be at least 3 characters")
	}
	matched, _ := regexp.MatchString("^[a-zA-Z0-9.-]+$", input)
	if !matched {
		return fmt.Errorf("invalid domain format")
	}
	return nil
}

func validateBrandName(input string) error {
	if len(input) < 2 {
		return fmt.Errorf("brand name must be at least 2 characters")
	}
	return nil
}

func validateDBName(input string) error {
	if len(input) < 3 {
		return fmt.Errorf("database name must be at least 3 characters")
	}
	matched, _ := regexp.MatchString("^[a-zA-Z0-9_]+$", input)
	if !matched {
		return fmt.Errorf("database name must contain only letters, numbers, and underscores")
	}
	return nil
}

func validateDBUsername(input string) error {
	if len(input) < 2 {
		return fmt.Errorf("username must be at least 2 characters")
	}
	matched, _ := regexp.MatchString("^[a-zA-Z0-9_]+$", input)
	if !matched {
		return fmt.Errorf("username must contain only letters, numbers, and underscores")
	}
	return nil
}

func validateUsername(input string) error {
	if len(input) < 3 {
		return fmt.Errorf("username must be at least 3 characters")
	}
	matched, _ := regexp.MatchString("^[a-zA-Z0-9_]+$", input)
	if !matched {
		return fmt.Errorf("username must contain only letters, numbers, and underscores")
	}
	return nil
}

func validatePassword(input string) error {
	if len(input) < 6 {
		return fmt.Errorf("password must be at least 6 characters")
	}
	return nil
}

// Helper function to prompt for input with validation
func promptInput(scanner *bufio.Scanner, prompt string, validator func(string) error) string {
	for {
		fmt.Printf("  ▸ %s: ", prompt)
		scanner.Scan()
		input := strings.TrimSpace(scanner.Text())

		if err := validator(input); err != nil {
			fmt.Printf("    ❌ %s\n", err.Error())
			continue
		}

		return input
	}
}

// Helper function to prompt for input with default value
func promptInputWithDefault(scanner *bufio.Scanner, prompt, defaultValue string, validator func(string) error) string {
	fmt.Printf("  ▸ %s [%s]: ", prompt, defaultValue)
	scanner.Scan()
	input := strings.TrimSpace(scanner.Text())

	if input == "" {
		return defaultValue
	}

	if err := validator(input); err != nil {
		fmt.Printf("  ❌ %s, using default: %s\n", err.Error(), defaultValue)
		return defaultValue
	}

	return input
}

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all clients",
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := registry.NewStore(getRegistryPath())
			clients, err := reg.List()
			if err != nil {
				return err
			}

			if len(clients) == 0 {
				fmt.Println("No clients found")
				return nil
			}

			fmt.Printf("\nTotal clients: %d\n\n", len(clients))
			fmt.Printf("%-15s %-20s %-10s %-20s\n", "ID", "BRAND", "STATUS", "PORTS (API/SF/PG)")
			fmt.Printf("%-15s %-20s %-10s %-20s\n", "───", "─────", "──────", "─────────────────")

			for _, c := range clients {
				ports := fmt.Sprintf("%d/%d/%d", c.AppPort, c.StorefrontPort, c.PostgresPort)
				fmt.Printf("%-15s %-20s %-10s %-20s\n", c.ID, c.BrandName, c.Status, ports)
			}
			fmt.Println()

			return nil
		},
	}
}

func deleteCmd() *cobra.Command {
	var clientID string
	var purge bool

	cmd := &cobra.Command{
		Use:   "delete [id]",
		Short: "Delete a client",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				clientID = args[0]
			}

			if clientID == "" {
				return fmt.Errorf("client ID is required as an argument or via --id flag")
			}

			reg := registry.NewStore(getRegistryPath())
			dbProv := db.NewProvisioner(config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.AdminDB)
			prov := core.NewProvisioner(config, reg, dbProv)

			return prov.Delete(clientID, purge)
		},
	}

	cmd.Flags().StringVarP(&clientID, "id", "i", "", "Client ID")
	cmd.Flags().BoolVar(&purge, "purge", false, "Remove all containers, files, and data (database)")

	return cmd
}

func statusCmd() *cobra.Command {
	var clientID string

	cmd := &cobra.Command{
		Use:   "status [id]",
		Short: "Show client status",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				clientID = args[0]
			}

			if clientID == "" {
				return fmt.Errorf("client ID is required as an argument or via --id flag")
			}

			reg := registry.NewStore(getRegistryPath())
			client, err := reg.Get(clientID)
			if err != nil {
				return err
			}

			fmt.Printf("\n")
			fmt.Printf("Client: %s\n", client.ID)
			fmt.Printf("Brand:  %s\n", client.BrandName)
			fmt.Printf("Status: %s\n", client.Status)
			fmt.Printf("DB:     %s\n\n", client.DBName)
			fmt.Printf("CommerceX API: http://localhost:%d\n", client.AppPort)
			fmt.Printf("Storefront:    http://localhost:%d\n", client.StorefrontPort)
			fmt.Printf("PostgreSQL:    localhost:%d\n\n", client.PostgresPort)
			fmt.Printf("Admin: %s / %s\n\n", client.AdminUsername, client.AdminPassword)

			return nil
		},
	}

	cmd.Flags().StringVarP(&clientID, "id", "i", "", "Client ID")

	return cmd
}

func startCmd() *cobra.Command {
	var clientID string

	cmd := &cobra.Command{
		Use:   "start [id]",
		Short: "Start a stopped client's containers",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				clientID = args[0]
			}
			if clientID == "" {
				return fmt.Errorf("client ID is required as an argument or via --id flag")
			}

			// Verify client exists
			reg := registry.NewStore(getRegistryPath())
			if _, err := reg.Get(clientID); err != nil {
				return err
			}

			deployer := deploy.NewDockerDeployer(getWorkDir())
			fmt.Printf("Starting %s...\n", clientID)
			if err := deployer.Start(clientID); err != nil {
				return err
			}
			fmt.Printf("✓ %s started\n", clientID)
			return nil
		},
	}

	cmd.Flags().StringVarP(&clientID, "id", "i", "", "Client ID")
	return cmd
}

func stopCmd() *cobra.Command {
	var clientID string

	cmd := &cobra.Command{
		Use:   "stop [id]",
		Short: "Stop a running client's containers",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				clientID = args[0]
			}
			if clientID == "" {
				return fmt.Errorf("client ID is required as an argument or via --id flag")
			}

			reg := registry.NewStore(getRegistryPath())
			if _, err := reg.Get(clientID); err != nil {
				return err
			}

			deployer := deploy.NewDockerDeployer(getWorkDir())
			fmt.Printf("Stopping %s...\n", clientID)
			if err := deployer.Stop(clientID); err != nil {
				return err
			}
			fmt.Printf("✓ %s stopped\n", clientID)
			return nil
		},
	}

	cmd.Flags().StringVarP(&clientID, "id", "i", "", "Client ID")
	return cmd
}

func logsCmd() *cobra.Command {
	var clientID, service string
	var tail int
	var follow bool

	cmd := &cobra.Command{
		Use:   "logs [id]",
		Short: "View logs for a client's containers",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				clientID = args[0]
			}
			if clientID == "" {
				return fmt.Errorf("client ID is required as an argument or via --id flag")
			}

			reg := registry.NewStore(getRegistryPath())
			if _, err := reg.Get(clientID); err != nil {
				return err
			}

			deployer := deploy.NewDockerDeployer(getWorkDir())
			return deployer.Logs(clientID, service, tail, follow)
		},
	}

	cmd.Flags().StringVarP(&clientID, "id", "i", "", "Client ID")
	cmd.Flags().StringVarP(&service, "service", "s", "", "Specific service (commercex-server, commercex-worker, postgres_db, storefront)")
	cmd.Flags().IntVar(&tail, "tail", 100, "Number of log lines to show from the end")
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log output (like tail -f)")
	return cmd
}

func dashboardCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "dashboard",
		Short: "Open the interactive TUI dashboard",
		Long:  "Launch a live terminal dashboard showing all clients, container health, and management actions.",
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
