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

func (p *Provisioner) Create(req *CreateRequest) (*registry.Client, error) {
	fmt.Printf("Creating client: %s\n", req.ClientID)
	isRemote := req.ServerHost != ""

	// 1. Validate
	if err := p.validate(req); err != nil {
		return nil, err
	}
	fmt.Println("✓ Validation passed")

	// 2. Generate secrets (auto-generated, user doesn't need to input)
	cookieSecret := secrets.GenerateSecret()
	fmt.Println("✓ Secrets generated")

	// 3. Get ports (3 ports per client: app, postgres, storefront)
	appPort := p.getNextPort()
	postgresPort := appPort + 1
	storefrontPort := appPort + 2
	fmt.Printf("✓ Assigned ports: App=%d, Postgres=%d, Storefront=%d\n", appPort, postgresPort, storefrontPort)

	// 4. Create work directory
	clientDir := filepath.Join(p.config.WorkDir, req.ClientID)
	if err := os.MkdirAll(clientDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %v", err)
	}
	fmt.Println("✓ Work directory created")

	// 5. Clone storefront
	if err := p.cloneStorefront(clientDir); err != nil {
		return nil, fmt.Errorf("failed to clone storefront: %v", err)
	}
	fmt.Println("✓ Storefront cloned")

	// 6. Create storefront Dockerfile
	storefrontDir := filepath.Join(clientDir, "storefront")
	if err := p.createStorefrontDockerfile(storefrontDir); err != nil {
		return nil, fmt.Errorf("failed to create Dockerfile: %v", err)
	}
	fmt.Println("✓ Dockerfile created")

	// 6.5. Create storefront .env file
	if err := p.createStorefrontEnv(storefrontDir, req.BrandName, req.Domain, appPort, req.ServerHost); err != nil {
		return nil, fmt.Errorf("failed to create storefront .env: %v", err)
	}
	fmt.Println("✓ Storefront .env created")

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
	fmt.Println("✓ Templates rendered (env, nginx, docker-compose)")

	// 10. Deploy
	isRemote = req.ServerHost != ""
	if isRemote {
		fmt.Printf("Deploying to remote server %s...\n", req.ServerHost)
		ssh, err := deploy.NewSSHOrchestrator(req.ServerHost, req.ServerUser, req.SSHPassword, req.SSHKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize SSH: %v", err)
		}

		if err := ssh.CheckPrerequisites(); err != nil {
			return nil, err
		}

		remotePath := fmt.Sprintf("/opt/innovatex/clients/%s", req.ClientID)
		fmt.Println("Transferring files...")
		if err := ssh.PushFiles(clientDir, remotePath); err != nil {
			return nil, err
		}

		fmt.Println("Booting containers on remote...")
		if err := ssh.DeployRemote(remotePath); err != nil {
			return nil, err
		}
		fmt.Println("✓ Remote deployment successful")
	} else {
		fmt.Println("Deploying containers locally...")
		if err := p.deployer.Deploy(req.ClientID); err != nil {
			return nil, fmt.Errorf("deployment failed: %v", err)
		}
		fmt.Println("✓ Containers deployed")
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
	fmt.Println("✓ Client registered")

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
	existing, _ := p.registry.Get(req.ClientID)
	if existing != nil {
		return fmt.Errorf("client %s already exists", req.ClientID)
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
