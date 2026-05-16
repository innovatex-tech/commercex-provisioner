package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/innovatex-tech/commercex-provisioner/internal/db"
	"github.com/innovatex-tech/commercex-provisioner/internal/deploy"
	"github.com/innovatex-tech/commercex-provisioner/internal/registry"
	"github.com/innovatex-tech/commercex-provisioner/internal/secrets"
	"github.com/innovatex-tech/commercex-provisioner/internal/templates"
)

type CreateRequest struct {
	ClientID  string
	Domain    string
	BrandName string

	// Database (user provides)
	DBName     string
	DBUsername string
	DBPassword string

	// Admin (user provides)
	AdminUsername string
	AdminPassword string

	// Remote Server (optional)
	ServerHost  string
	ServerUser  string
	SSHPassword string
	SSHKeyPath  string
}

type Config struct {
	WorkDir        string
	TemplateDir    string
	StorefrontRepo string
	DBHost         string
	DBPort         int
	DBUser         string
	DBPassword     string
	AdminDB        string
	BasePort       int
}

type Provisioner struct {
	registry *registry.Store
	db       *db.Provisioner
	renderer *templates.Renderer
	deployer *deploy.DockerDeployer
	config   *Config
}

func NewProvisioner(config *Config, reg *registry.Store, dbProv *db.Provisioner) *Provisioner {
	return &Provisioner{
		registry: reg,
		db:       dbProv,
		renderer: templates.NewRenderer(config.TemplateDir),
		deployer: deploy.NewDockerDeployer(config.WorkDir),
		config:   config,
	}
}

type ProgressStep struct {
	Step    string
	Percent float64
}

type ProgressFunc func(ProgressStep)

func (p *Provisioner) Create(req *CreateRequest, onProgress ProgressFunc) (*registry.Client, error) {
	if onProgress == nil {
		onProgress = func(ProgressStep) {}
	}

	onProgress(ProgressStep{"Validating request...", 0.05})
	isRemote := req.ServerHost != ""

	// 1. Validate
	if err := p.validate(req); err != nil {
		return nil, err
	}
	onProgress(ProgressStep{"Assigning ports...", 0.10})

	// 2. Generate secrets (auto-generated, user doesn't need to input)
	cookieSecret := secrets.GenerateSecret()
	onProgress(ProgressStep{"Generating secrets...", 0.15})

	// 3. Get ports (3 ports per client: app, postgres, storefront)
	appPort := p.getNextPort()
	postgresPort := appPort + 1
	storefrontPort := appPort + 2
	onProgress(ProgressStep{"Creating work directory...", 0.20})

	// 4. Create work directory
	clientDir := filepath.Join(p.config.WorkDir, req.ClientID)
	if err := os.MkdirAll(clientDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %v", err)
	}
	onProgress(ProgressStep{"Cloning storefront repo...", 0.25})

	// 5. Clone storefront
	if err := p.cloneStorefront(clientDir); err != nil {
		return nil, fmt.Errorf("failed to clone storefront: %v", err)
	}
	onProgress(ProgressStep{"Configuring storefront...", 0.35})

	// 6. Create storefront Dockerfile
	storefrontDir := filepath.Join(clientDir, "storefront")
	if err := p.createStorefrontDockerfile(storefrontDir); err != nil {
		return nil, fmt.Errorf("failed to create Dockerfile: %v", err)
	}
	onProgress(ProgressStep{"Generating environment files...", 0.45})

	// 6.5. Create storefront .env file
	if err := p.createStorefrontEnv(storefrontDir, req.BrandName, req.Domain, appPort, req.ServerHost); err != nil {
		return nil, fmt.Errorf("failed to create storefront .env: %v", err)
	}
	onProgress(ProgressStep{"Rendering templates...", 0.50})

	// 7. Prepare template data
	targetHost := req.Domain
	if isRemote && req.ServerHost != "" {
		targetHost = req.ServerHost
	}

	publicStorefrontURL := fmt.Sprintf("http://%s:%d", targetHost, storefrontPort)

	templateData := map[string]interface{}{
		"ClientID":            req.ClientID,
		"Domain":              req.Domain,
		"BrandName":           req.BrandName,
		"DBName":              req.DBName,
		"DBUsername":          req.DBUsername,
		"DBPassword":          req.DBPassword,
		"AdminUsername":       req.AdminUsername,
		"AdminPassword":       req.AdminPassword,
		"CookieSecret":        cookieSecret,
		"AppPort":             appPort,
		"PostgresPort":        postgresPort,
		"StorefrontPort":      storefrontPort,
		"PublicStorefrontURL": publicStorefrontURL,
		"GeneratedAt":         time.Now().Format(time.RFC3339),
	}

	// 8. Render .env file
	if err := p.renderer.Render(".env.tmpl", templateData, filepath.Join(clientDir, ".env")); err != nil {
		return nil, fmt.Errorf("failed to render .env: %v", err)
	}

	// 9. Render nginx.conf
	if err := p.renderer.Render("nginx.conf.tmpl", templateData, filepath.Join(clientDir, "nginx.conf")); err != nil {
		return nil, fmt.Errorf("failed to render nginx.conf: %v", err)
	}

	// 10. Render docker-compose.yml
	if err := p.renderer.Render("docker-compose.yml.tmpl", templateData, filepath.Join(clientDir, "docker-compose.yml")); err != nil {
		return nil, fmt.Errorf("failed to render docker-compose.yml: %v", err)
	}
	onProgress(ProgressStep{"Preparing deployment...", 0.60})

	// 10. Deploy
	isRemote = req.ServerHost != ""
	if isRemote {
		onProgress(ProgressStep{fmt.Sprintf("Connecting to %s...", req.ServerHost), 0.65})
		ssh, err := deploy.NewSSHOrchestrator(req.ServerHost, req.ServerUser, req.SSHPassword, req.SSHKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize SSH: %v", err)
		}

		if err := ssh.CheckPrerequisites(); err != nil {
			return nil, err
		}

		remotePath := fmt.Sprintf("/opt/innovatex/clients/%s", req.ClientID)
		onProgress(ProgressStep{"Transferring files...", 0.70})
		if err := ssh.PushFiles(clientDir, remotePath); err != nil {
			return nil, err
		}

		onProgress(ProgressStep{"Booting containers on remote...", 0.85})
		if err := ssh.DeployRemote(remotePath); err != nil {
			return nil, err
		}
		onProgress(ProgressStep{"Finalizing deployment...", 0.95})
	} else {
		onProgress(ProgressStep{"Finalizing local deployment...", 0.95})
	}

	// 11. Create client record
	client := &registry.Client{
		ID:             req.ClientID,
		Domain:         req.Domain,
		BrandName:      req.BrandName,
		Status:         "active",
		DBName:         req.DBName,
		DBUsername:     req.DBUsername,
		DBPassword:     req.DBPassword,
		AdminUsername:  req.AdminUsername,
		AdminPassword:  req.AdminPassword,
		CookieSecret:   cookieSecret,
		AppPort:        appPort,
		PostgresPort:   postgresPort,
		StorefrontPort: storefrontPort,
		IsRemote:       isRemote,
		ServerHost:     req.ServerHost,
		ServerUser:     req.ServerUser,
		SSHPassword:    req.SSHPassword,
		SSHKeyPath:     req.SSHKeyPath,
		CreatedAt:      time.Now(),
	}

	if err := p.registry.Save(client); err != nil {
		return nil, fmt.Errorf("failed to save client: %v", err)
	}
	onProgress(ProgressStep{"Complete!", 1.0})

	return client, nil
}

func (p *Provisioner) Delete(clientID string, purge bool) error {
	fmt.Printf("Deleting client: %s\n", clientID)

	// 1. Get client
	client, err := p.registry.Get(clientID)
	if err != nil {
		return err
	}

	// 2. Stop and remove containers + data
	if client.IsRemote {
		fmt.Printf("Cleaning up remote server %s...\n", client.ServerHost)
		ssh, err := deploy.NewSSHOrchestrator(client.ServerHost, client.ServerUser, client.SSHPassword, client.SSHKeyPath)
		if err != nil {
			return fmt.Errorf("failed to connect to remote for cleanup: %v", err)
		}
		defer ssh.Close()

		remotePath := fmt.Sprintf("/opt/innovatex/clients/%s", clientID)
		if purge {
			fmt.Println("Purging remote containers and volumes (database)...")
			ssh.RunCommand(fmt.Sprintf("cd %s && docker compose down -v", remotePath))
			fmt.Println("Removing remote directory...")
			ssh.RunCommand(fmt.Sprintf("rm -rf %s", remotePath))
		} else {
			fmt.Println("Stopping remote containers...")
			ssh.RunCommand(fmt.Sprintf("cd %s && docker compose stop", remotePath))
		}
	} else {
		fmt.Println("Removing local containers...")
		if err := p.deployer.Remove(clientID); err != nil {
			fmt.Printf("Warning: Failed to remove containers: %v\n", err)
		}
		
		clientDir := filepath.Join(p.config.WorkDir, clientID)
		if purge {
			fmt.Println("Removing local directory and data...")
			os.RemoveAll(clientDir)
		}
	}

	// 4. Remove from registry
	if err := p.registry.Delete(clientID); err != nil {
		return err
	}

	if purge {
		fmt.Printf("✓ Client %s and all data purged\n", clientID)
	} else {
		fmt.Printf("✓ Client %s removed from registry (server files preserved)\n", clientID)
	}
	return nil
}

func (p *Provisioner) validate(req *CreateRequest) error {
	// Check for spaces in ID
	if strings.Contains(req.ClientID, " ") {
		return fmt.Errorf("client ID cannot contain spaces (use-dashes-instead)")
	}

	clients, err := p.registry.List()
	if err != nil {
		return fmt.Errorf("failed to read registry: %v", err)
	}

	for _, c := range clients {
		// Check for ID collision
		if c.ID == req.ClientID {
			return fmt.Errorf("client with ID '%s' already exists", req.ClientID)
		}
		// Check for Domain collision
		if c.Domain == req.Domain && req.Domain != "localhost" {
			return fmt.Errorf("domain '%s' is already assigned to client '%s'", req.Domain, c.ID)
		}
	}

	return nil
}

func (p *Provisioner) cloneStorefront(targetDir string) error {
	storefrontDir := filepath.Join(targetDir, "storefront")
	cmd := exec.Command("git", "clone", p.config.StorefrontRepo, storefrontDir)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone failed: %s", string(output))
	}

	return nil
}

func (p *Provisioner) getNextPort() int {
	clients, _ := p.registry.List()
	return p.config.BasePort + (len(clients) * 3) // 3 ports per client: app, postgres, storefront
}

func (p *Provisioner) createStorefrontDockerfile(storefrontDir string) error {
	dockerfile := `FROM node:20 AS builder
WORKDIR /app
COPY package.json ./
RUN rm -f package-lock.json && npm install
COPY . .
RUN npm run build

FROM nginx:alpine
# SPA Routing: redirect all 404s to index.html
RUN echo 'server { \
    listen 80; \
    location / { \
        root /usr/share/nginx/html; \
        index index.html index.htm; \
        try_files $uri $uri/ /index.html; \
    } \
}' > /etc/nginx/conf.d/default.conf

COPY --from=builder /app/dist /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
`
	return os.WriteFile(filepath.Join(storefrontDir, "Dockerfile"), []byte(dockerfile), 0644)
}

func (p *Provisioner) createStorefrontEnv(storefrontDir, brandName, domain string, appPort int, serverHost string) error {
	var apiURL string

	// If it's a remote deployment and we have a serverHost, use the Host (IP) for now
	// This ensures the site works even if the domain DNS hasn't propagated yet.
	target := domain
	if serverHost != "" {
		target = serverHost
	}

	if domain == "localhost" || strings.HasSuffix(domain, ".local") {
		apiURL = fmt.Sprintf("http://localhost:%d/shop-api", appPort)
	} else {
		apiURL = fmt.Sprintf("http://%s:%d/shop-api", target, appPort)
	}

	envContent := fmt.Sprintf(`# API Configuration
VITE_API_URL=%s
VITE_SITE_NAME=%s
`, apiURL, brandName)

	return os.WriteFile(filepath.Join(storefrontDir, ".env"), []byte(envContent), 0644)
}
