# InnovateX Provisioner - AI Coding Agent Instructions

## Project Overview

**InnovateX Provisioner** is a multi-client e-commerce provisioning platform that automatically creates fully isolated, production-ready commerce environments. This system provisions complete CommerceX instances with React/Vite storefronts, delivering true multi-tenancy through containerized infrastructure isolation.

**Tech Stack**: Go 1.25+, Docker Compose, PostgreSQL 16, CommerceX (Node.js commerce engine), React/Vite storefronts, Nginx reverse proxy

## Core Architecture

### **Multi-Tenant Isolation Strategy**
- **Complete Infrastructure Isolation**: Each client receives dedicated PostgreSQL database, Docker network, and 4-container stack
- **Dynamic Port Allocation**: 3 sequential ports per client (`BasePort + clientIndex*3`): app, postgres, storefront
- **Service Stack**: `commercex-server` (main app), `commercex-worker` (background jobs), `postgres_db`, `storefront` (React/Vite)
- **Container Naming**: `{service}_{clientID}` pattern (e.g., `commercex_server_demo-store`)
- **Network Isolation**: Each client has dedicated bridge network (`{clientID}_network`)

### **Deployment Pipeline**
- **Template Rendering**: Go `text/template` generates `.env`, `docker-compose.yml`, `nginx.conf` per client
- **Health Orchestration**: PostgreSQL health checks ensure database readiness before app startup
- **Storefront Provisioning**: Git clone from `StorefrontRepo` + generated multi-stage Dockerfile (node build → nginx serve)
- **Security**: Cryptographically secure secrets via `crypto/rand` (32-byte cookie secrets, configurable admin passwords)

### **Environment Configuration**
The system uses configurable paths (defaults to `~/.innovatex/`, override with env vars):
- `INNOVATEX_WORK_DIR`: Client deployment directories (default: `~/.innovatex/clients/`)
- `INNOVATEX_TEMPLATE_DIR`: Template files location (default: `~/.innovatex/templates/`)
- `INNOVATEX_REGISTRY`: Client registry JSON path (default: `~/.innovatex/registry.json`)

**Template Bootstrapping**: On first run, `ensureTemplates()` in [cmd/innovatex/main.go](cmd/innovatex/main.go) writes embedded templates to template dir if missing.

### **CLI Interface (Cobra-based)**
- **Interactive Prompts**: All flags optional - CLI prompts for missing values with validation
- **Commands**: `create`, `list`, `delete`, `status` (see [cmd/innovatex/main.go](cmd/innovatex/main.go))
- **Progress Feedback**: ✓ checkmarks for each provisioning step
- **Input Validation**: Regex validation for clientID (`^[a-z0-9-]+$`), domain, brand name

### **System Components**
```
internal/
├── core/provisioner.go       # Orchestrates 11-step Create() workflow, Delete() cleanup
├── registry/store.go         # JSON persistence (Save, List, Get, Delete)
├── deploy/docker.go          # Wraps docker-compose commands (Deploy, Stop, Remove)
├── templates/renderer.go     # text/template.Execute() wrapper
├── secrets/generator.go      # crypto/rand-based secret generation
└── db/provisioner.go         # Database provisioning logic (currently unused)
```

### **Client Creation Flow** (11 Steps in `Provisioner.Create()`)
1. **Validation**: Check clientID uniqueness in registry
2. **Secret Generation**: `secrets.GenerateSecret()` → 32-byte base64 cookie secret
3. **Port Allocation**: `getNextPort()` → `BasePort + (len(clients) * 3)` (3 ports per client)
4. **Directory Setup**: Create `{WorkDir}/{clientID}/`
5. **Storefront Clone**: Git clone from `config.StorefrontRepo`
6. **Dockerfile Generation**: Write multi-stage Dockerfile to `storefront/Dockerfile`
7. **Template Data Prep**: Build `templateData` map with all `{{.Variables}}`
8. **Render `.env`**: Template → `{clientDir}/.env`
9. **Render `nginx.conf`**: Template → `{clientDir}/nginx.conf`
10. **Render `docker-compose.yml`**: Template → `{clientDir}/docker-compose.yml`
11. **Deploy**: `docker-compose up -d --build` via `deployer.Deploy()`
12. **Registry Save**: Persist `Client` struct to JSON registry

## Developer Workflows

### **Build & Run**
```bash
# Build binary
go build -o bin/innovatex cmd/innovatex/main.go

# Create client (flags optional - will prompt if missing)
./bin/innovatex create --id=demo --domain=demo.local --brand="Demo Store" \
  --db-name=demo_db --db-username=demouser --db-password=pass123 \
  --admin-username=admin --admin-password=admin123

# Or use interactive mode (all prompts)
./bin/innovatex create

# List all clients
./bin/innovatex list

# Check client status
./bin/innovatex status --id=demo

# Delete client (keeps DB, removes containers/files)
./bin/innovatex delete --id=demo
```

### **Configuration Values** ([cmd/innovatex/main.go](cmd/innovatex/main.go#L237-L245))
```go
config := &core.Config{
  WorkDir:        "~/.innovatex/clients/",  // or $INNOVATEX_WORK_DIR
  TemplateDir:    "~/.innovatex/templates/", // or $INNOVATEX_TEMPLATE_DIR
  StorefrontRepo: "https://github.com/The-Coding-Kiddo/clothing-storefront.git",
  BasePort:       8000,  // Clients get 8000-8002, 8003-8005, etc.
}
```

## Project-Specific Patterns

### **Template Rendering System**
All templates use Go `text/template` syntax ([templates/](templates/)):
```go
// In provisioner.Create(), build templateData map:
templateData := map[string]interface{}{
  "ClientID":       req.ClientID,
  "AppPort":        appPort,        // e.g., 8000
  "PostgresPort":   postgresPort,   // e.g., 8001
  "StorefrontPort": storefrontPort, // e.g., 8002
  "DBName":         req.DBName,
  "DBUsername":     req.DBUsername,
  "DBPassword":     req.DBPassword,
  "AdminUsername":  req.AdminUsername,
  "AdminPassword":  req.AdminPassword,
  "CookieSecret":   cookieSecret,
  "GeneratedAt":    time.Now().Format(time.RFC3339),
}

// Render each template to client directory
renderer.Render(".env.tmpl", templateData, "{clientDir}/.env")
renderer.Render("nginx.conf.tmpl", templateData, "{clientDir}/nginx.conf")
renderer.Render("docker-compose.yml.tmpl", templateData, "{clientDir}/docker-compose.yml")
```

**Template Variable Usage**: `{{.ClientID}}`, `{{.AppPort}}`, `{{.DBName}}`, etc. are injected during rendering.

### **Docker Compose Stack** ([templates/docker-compose.yml.tmpl](templates/docker-compose.yml.tmpl))
4 services per client:
1. **`commercex-server`**: Main app (`abduazizali/commercex:latest`), port `{{.AppPort}}:3000`
2. **`commercex-worker`**: Background worker (same image, different entrypoint: `dist/index-worker.js`)
3. **`postgres_db`**: PostgreSQL 16-alpine, port `{{.PostgresPort}}:5432`, health checks
4. **`storefront`**: React/Vite build → nginx, port `{{.StorefrontPort}}:80`

**Volumes**: `postgres_data_{{.ClientID}}`, `commercex_static_{{.ClientID}}`  
**Network**: `{{.ClientID}}_network` (bridge driver)

### **Registry Pattern** ([internal/registry/store.go](internal/registry/store.go))
JSON file-based persistence:
```go
type Client struct {
  ID, Domain, BrandName, Status, DBName, DBUsername, DBPassword string
  AdminUsername, AdminPassword, CookieSecret                    string
  AppPort, PostgresPort, StorefrontPort                         int
  CreatedAt                                                     time.Time
}
```

**Operations**: `Save()` (upsert), `List()`, `Get(id)`, `Delete(id)`  
**Usage**: Prevents duplicate clientIDs, tracks port allocations, enables `list` command

## Storefront Integration

### **React/Vite Stack**
- **Cloning**: `git clone {StorefrontRepo}` into `{clientDir}/storefront/`
- **Dockerfile Generation**: Provisioner writes multi-stage Dockerfile ([provisioner.go#L228-L242](internal/core/provisioner.go)):
  ```dockerfile
  FROM node:20 AS builder
  WORKDIR /app
  COPY package.json ./
  RUN rm -f package-lock.json && npm install
  COPY . .
  RUN npm run build
  
  FROM nginx:alpine
  COPY --from=builder /app/dist /usr/share/nginx/html
  EXPOSE 80
  CMD ["nginx", "-g", "daemon off;"]
  ```
- **UI Stack**: Shadcn/ui (Radix), Tailwind CSS, React Hook Form, Apollo Client for GraphQL

### **CommerceX Commerce Engine**
- **Container Image**: `abduazizali/commercex:latest` (custom Node.js commerce platform)
- **Dual Containers**: `commercex-server` (web), `commercex-worker` (background jobs)
- **Internal API**: Server runs on port 3000 inside container, proxied by nginx
- **Environment Config**: `.env` file with DB credentials, ports, cookie secrets
- **Health Dependency**: Services wait for `postgres_db` health check before starting

### **Service Discovery**
- Storefront → CommerceX: Internal hostname `commercex-server:3000` (via Docker network)
- Nginx reverse proxy: Exposed on `{{.AppPort}}` for external access
- Database: Hostname `postgres_db` within network, external `localhost:{{.PostgresPort}}`

## Code Patterns

### **Provisioner Orchestration** ([internal/core/provisioner.go](internal/core/provisioner.go))
The `Create()` method is the heart of client provisioning:
- Returns `(*registry.Client, error)` - either full client record or error
- Uses `fmt.Println("✓ Step description")` for progress feedback
- Calls helper methods: `validate()`, `cloneStorefront()`, `createStorefrontDockerfile()`, `getNextPort()`
- All template rendering happens via `renderer.Render(templateName, data, outputPath)`
- Final step: `deployer.Deploy(clientID)` wraps `docker-compose up -d --build`

`Delete()` method cleanup:
- Stops containers: `deployer.Remove(clientID)` → `docker-compose down -v`
- Removes directory: `os.RemoveAll(clientDir)`
- Preserves database (warning printed to user)
- Updates registry: `registry.Delete(clientID)`

### **CLI Structure** ([cmd/innovatex/main.go](cmd/innovatex/main.go))
- **Main function**: Calls `ensureTemplates()`, builds Cobra rootCmd, adds subcommands
- **Config initialization** (line 237): `config := &core.Config{...}` with hardcoded defaults
- **Interactive prompts**: `promptInput(scanner, label, validationFunc)` pattern throughout
- **Validation helpers**: `validateClientID`, `validateDomain`, `validateBrandName` use regex
- **Template bootstrapping**: `ensureTemplates()` writes `.env.tmpl`, `docker-compose.yml.tmpl`, `nginx.conf.tmpl` if missing

## Testing & Debugging

### **Verify Provisioning**
```bash
# Check registry entry
cat ~/.innovatex/registry.json | jq '.[] | select(.ID=="demo")'  # or use grep

# List running containers
docker ps | grep demo-store
# Expected: commercex_server_demo, commercex_worker_demo, postgres_demo, storefront_demo

# Check logs
docker logs commercex_server_demo-store -f
docker logs storefront_demo-store --tail=50

# Test connectivity
curl http://localhost:8002  # storefront (assuming port 8002)
curl http://localhost:8000  # commercex app (assuming port 8000)

# Inspect network
docker network ls | grep demo-store
docker network inspect demo-store_network
```

### **Common Issues**
- **Port conflicts**: Check `getNextPort()` in [provisioner.go](internal/core/provisioner.go#L214) - formula is `BasePort + (len(clients) * 3)`
- **Template errors**: Verify template files exist in `~/.innovatex/templates/` (or `$INNOVATEX_TEMPLATE_DIR`)
- **Git clone fails**: Check network access to `StorefrontRepo` URL in config
- **DB connection**: Ensure postgres container is healthy before commercex starts (health check in docker-compose)
- **Docker build**: Multi-stage Dockerfile requires `npm run build` to succeed in storefront

## Modifying the Codebase

### **Adding New Template Variables**
1. Add field to `templateData` map in [provisioner.go](internal/core/provisioner.go#L110-L123)
2. Update template file (e.g., `templates/.env.tmpl`) with `{{.NewVariable}}`
3. Optionally add to `registry.Client` struct if persistence needed

### **Changing Port Allocation**
Modify `getNextPort()` in [provisioner.go](internal/core/provisioner.go#L214):
```go
// Current: 3 ports per client (app, postgres, storefront)
return p.config.BasePort + (len(clients) * 3)

// Example: 4 ports per client (add nginx proxy)
return p.config.BasePort + (len(clients) * 4)
```

### **Adding New CLI Commands**
1. Create command function in [cmd/innovatex/main.go](cmd/innovatex/main.go): `func statusCmd() *cobra.Command { ... }`
2. Add to rootCmd: `rootCmd.AddCommand(statusCmd())`
3. Call provisioner methods within `RunE` handler
4. Use `promptInput()` for interactive values

### **Extending Docker Services**
1. Edit [templates/docker-compose.yml.tmpl](templates/docker-compose.yml.tmpl)
2. Add service definition with `{{.ClientID}}` in names/networks
3. Update `templateData` if new env vars needed
4. Adjust port allocation in `getNextPort()` if new ports required

### **Key Files for Common Changes**
- **Port/network changes**: [internal/core/provisioner.go](internal/core/provisioner.go)
- **CLI flags/prompts**: [cmd/innovatex/main.go](cmd/innovatex/main.go)
- **Container stack**: [templates/docker-compose.yml.tmpl](templates/docker-compose.yml.tmpl)
- **Environment variables**: `templates/.env.tmpl` (embedded in [main.go](cmd/innovatex/main.go))
- **Registry schema**: [internal/registry/models.go](internal/registry/models.go)
