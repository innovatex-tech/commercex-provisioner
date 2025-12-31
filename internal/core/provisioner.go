package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/innovatex/provisioner/internal/db"
	"github.com/innovatex/provisioner/internal/deploy"
	"github.com/innovatex/provisioner/internal/registry"
	"github.com/innovatex/provisioner/internal/secrets"
	"github.com/innovatex/provisioner/internal/templates"
)

type CreateRequest struct {
	ClientID   string
	Domain     string
	BrandName  string
	AdminEmail string
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

	// 1. Validate
	if err := p.validate(req); err != nil {
		return nil, err
	}
	fmt.Println("✓ Validation passed")

	// 2. Generate secrets
	adminPassword := secrets.GeneratePassword(16)
	cookieSecret := secrets.GenerateSecret()
	revalidationSecret := secrets.GenerateSecret()
	fmt.Println("✓ Secrets generated")

	// 3. Get ports
	vendurePort := p.getNextPort()
	storefrontPort := vendurePort + 1
	fmt.Printf("✓ Assigned ports: Vendure=%d, Storefront=%d\n", vendurePort, storefrontPort)

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

	// In the Create function, after line "✓ Storefront cloned"
	storefrontDir := filepath.Join(clientDir, "storefront")
	if err := p.createStorefrontDockerfile(storefrontDir); err != nil {
		return nil, fmt.Errorf("failed to create Dockerfile: %v", err)
	}
	fmt.Println("✓ Dockerfile created")

	// 6. Database name
	dbName := fmt.Sprintf("vendure_%s", req.ClientID)

	// 7. Render templates
	templateData := map[string]interface{}{
		"ClientID":           req.ClientID,
		"Domain":             req.Domain,
		"BrandName":          req.BrandName,
		"DBName":             dbName,
		"AdminEmail":         req.AdminEmail,
		"AdminPassword":      adminPassword,
		"CookieSecret":       cookieSecret,
		"RevalidationSecret": revalidationSecret,
		"VendurePort":        vendurePort,
		"StorefrontPort":     storefrontPort,
	}

	if err := p.renderer.Render("vendure.env.tmpl", templateData, filepath.Join(clientDir, "vendure.env")); err != nil {
		return nil, fmt.Errorf("failed to render vendure.env: %v", err)
	}

	if err := p.renderer.Render("docker-compose.yml.tmpl", templateData, filepath.Join(clientDir, "docker-compose.yml")); err != nil {
		return nil, fmt.Errorf("failed to render docker-compose.yml: %v", err)
	}
	fmt.Println("✓ Templates rendered")

	// 8. Deploy
	fmt.Println("Deploying containers...")
	if err := p.deployer.Deploy(req.ClientID); err != nil {
		return nil, fmt.Errorf("deployment failed: %v", err)
	}
	fmt.Println("✓ Containers deployed")

	// 9. Create client record
	client := &registry.Client{
		ID:             req.ClientID,
		Domain:         req.Domain,
		BrandName:      req.BrandName,
		Status:         "active",
		DBName:         dbName,
		AdminEmail:     req.AdminEmail,
		AdminPassword:  adminPassword,
		CookieSecret:   cookieSecret,
		VendurePort:    vendurePort,
		StorefrontPort: storefrontPort,
		CreatedAt:      time.Now(),
	}

	if err := p.registry.Save(client); err != nil {
		return nil, fmt.Errorf("failed to save client: %v", err)
	}
	fmt.Println("✓ Client registered")

	return client, nil
}

func (p *Provisioner) Delete(clientID string) error {
	fmt.Printf("Deleting client: %s\n", clientID)

	// 1. Get client
	client, err := p.registry.Get(clientID)
	if err != nil {
		return err
	}

	// 2. Stop and remove containers
	fmt.Println("Removing containers...")
	if err := p.deployer.Remove(clientID); err != nil {
		fmt.Printf("Warning: Failed to remove containers: %v\n", err)
	}

	// 3. Remove work directory
	clientDir := filepath.Join(p.config.WorkDir, clientID)
	if err := os.RemoveAll(clientDir); err != nil {
		fmt.Printf("Warning: Failed to remove directory: %v\n", err)
	}

	// 4. Remove from registry
	if err := p.registry.Delete(clientID); err != nil {
		return err
	}

	fmt.Printf("✓ Client %s deleted (DB: %s preserved)\n", clientID, client.DBName)
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
	return p.config.BasePort + (len(clients) * 2)
}

// Add this function to provisioner.go
func (p *Provisioner) createStorefrontDockerfile(storefrontDir string) error {
	dockerfile := `FROM node:20 AS builder

WORKDIR /app

COPY package.json ./
RUN rm -f package-lock.json && npm install

COPY . .
RUN npm run build

# Production stage - serve with nginx
FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
`

	return os.WriteFile(filepath.Join(storefrontDir, "Dockerfile"), []byte(dockerfile), 0644)
}
