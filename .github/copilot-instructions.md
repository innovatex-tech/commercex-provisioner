# InnovateX Provisioner - AI Coding Agent Instructions

## Project Overview

**InnovateX Provisioner** is a revolutionary multi-client e-commerce provisioning platform that automatically creates fully isolated, production-ready commerce environments. This enterprise-grade system provisions complete Vendure commerce instances with React/Vite storefronts, delivering true multi-tenancy through containerized infrastructure isolation.

## Platform Innovations & Accomplishments

### 🚀 **InnovateX Shop Platform - Key Innovations**

#### **Multi-Tenant Commerce Architecture**
- **Complete Infrastructure Isolation**: Each client receives dedicated PostgreSQL databases, Docker networks, and container stacks
- **Dynamic Port Allocation**: Intelligent port management system prevents conflicts across unlimited client deployments
- **Zero-Dependency Client Separation**: Clients cannot interfere with each other's data, performance, or availability

#### **Advanced Production Deployment Pipeline**
- **Multi-Stage Docker Builds**: Optimized production builds with nginx serving for maximum performance
- **Container Health Orchestration**: PostgreSQL health checks ensure proper service startup sequencing
- **Template-Driven Configuration**: Dynamic environment and infrastructure generation per client
- **Automated Secret Management**: Cryptographically secure password and session secret generation

#### **Enterprise-Grade Storefront Technology**
- **Modern React/Vite Stack**: High-performance frontend with hot module replacement and optimized builds
- **Component Library Integration**: Shadcn/ui (Radix UI) for consistent, accessible design systems
- **GraphQL-First API Integration**: Seamless Vendure Shop API integration with type safety
- **Production-Ready Serving**: Nginx-based static asset serving with proper caching headers

### 🏗️ **Technical Architecture Breakthroughs**

#### **Intelligent Provisioning Engine**
- **One-Command Deployment**: Single CLI command provisions complete commerce infrastructure
- **Template Rendering System**: Go text/template system dynamically generates Docker Compose and environment configurations
- **Registry-Based State Management**: JSON-based client registry for deployment tracking and management
- **Error-Resilient Workflows**: Comprehensive validation and rollback capabilities

#### **Container Orchestration Mastery**
- **Service Discovery Networks**: Internal Docker networks enable secure service-to-service communication
- **Persistent Data Management**: Named volume strategies for database persistence across container lifecycles
- **Build Cache Optimization**: Layered Docker builds with intelligent caching for rapid deployments
- **Health Check Integration**: Container dependency management with health-based startup sequencing

### 💡 **Development Experience Innovations**

#### **Developer-Friendly CLI Interface**
- **Cobra-Powered Commands**: Professional CLI with create, list, status, and delete operations
- **Progress Feedback**: Real-time deployment progress with visual checkmarks and error reporting
- **Flexible Configuration**: Command-line flag support for all client parameters
- **Registry Integration**: Persistent client state management and conflict prevention

#### **Debugging & Maintenance Tools**
- **Container Log Aggregation**: Direct access to individual service logs per client
- **Health Status Monitoring**: Real-time container and service status checking
- **Port Conflict Resolution**: Automatic port allocation with collision avoidance
- **Template Validation**: Built-in template rendering verification

## Core Architecture Pattern

### **System Components**
- **Provisioner** ([internal/core/provisioner.go](internal/core/provisioner.go)) - Orchestrates complete client lifecycle workflows
- **Registry** ([internal/registry/](internal/registry/)) - Maintains persistent client metadata and state
- **Database Layer** ([internal/db/](internal/db/)) - Manages per-client PostgreSQL provisioning
- **Docker Deployer** ([internal/deploy/docker.go](internal/deploy/docker.go)) - Handles container orchestration lifecycle
- **Template Engine** ([internal/templates/](internal/templates/)) - Dynamic configuration generation system
- **Secrets Generator** ([internal/secrets/](internal/secrets/)) - Cryptographic password and token generation

### **Revolutionary Client Creation Flow**
1. **Input Validation**: CLI ([cmd/innovatex/main.go](cmd/innovatex/main.go)) accepts and validates: `clientID`, `domain`, `brandName`, `adminEmail`
2. **Security Generation**: Provisioner generates cryptographically secure secrets (16-char admin passwords, 32-byte random cookie secrets)
3. **Infrastructure Allocation**: Assigns sequential ports using algorithm: `BasePort + 2*clientIndex` for Vendure, `+1` for Storefront
4. **Storefront Provisioning**: Clones production-ready React/Vite template from `StorefrontRepo` into isolated client directory
5. **Configuration Generation**: Renders dynamic `docker-compose.yml` & `vendure.env` from Go templates with client-specific data
6. **Container Deployment**: Executes `docker-compose up -d --build` with health check orchestration
7. **Registry Registration**: Persists client metadata in `data/registry.json` for ongoing management

## Key Developer Workflows

### **Building & Deployment**
```bash
# Build the provisioner binary
go build -o bin/innovatex cmd/innovatex/main.go

# Deploy a new client (complete commerce infrastructure in one command)
./bin/innovatex create --id=client1 --domain=client1.local --brand="Client Store" --email=admin@client1.com
```

### **Management Operations**
- **Create**: `./bin/innovatex create --id=store1 --domain=store1.local --brand="Store One" --email=admin@store1.com`
- **List All Clients**: `./bin/innovatex list` (displays all provisioned clients from registry)
- **Client Status**: `./bin/innovatex status --id=store1` (health and container status)
- **Cleanup**: `./bin/innovatex delete --id=store1` (complete infrastructure teardown with data preservation option)

### **Configuration & Security**
- **Database Credentials**: Centralized PostgreSQL credentials in config with per-client database isolation
- **Per-Client Secrets**: Unique admin passwords (16-character complexity), cryptographic cookie secrets (32-byte entropy)
- **Port Management**: Intelligent allocation system in `getNextPort()` with collision prevention

## Project-Specific Patterns

### **Multi-Client Isolation Strategy**
Revolutionary approach to true multi-tenancy:
- **Network Isolation**: Separate Docker network per client: `{clientID}_network`
- **Database Separation**: Unique PostgreSQL database: `vendure_{clientID}`
- **Container Isolation**: Dedicated container names: `postgres_{clientID}`, `vendure_{clientID}`, `storefront_{clientID}`
- **Port Independence**: Dynamic port allocation eliminates conflicts across unlimited clients

### **Template-Driven Infrastructure**
Advanced configuration management:
- **Go Template Integration**: Files in `templates/` use `text/template` syntax for dynamic generation
- **Variable Injection**: `{{.ClientID}}`, `{{.DBName}}`, `{{.VendurePort}}` injected during provisioning
- **Environment Generation**: Client-specific environment variables and Docker Compose services
- **Render Pipeline**: Templates rendered to `data/clients/{clientID}/` during deployment workflow

### **Container Orchestration Excellence**
Production-ready Docker architecture:
- **Three-Service Stack**: PostgreSQL (data), Vendure (commerce engine), Storefront (React/Vite frontend)
- **Persistent Storage**: Named volumes for database persistence across container lifecycles
- **Service Discovery**: Internal Docker networks enable secure `vendure` hostname resolution
- **Multi-Stage Builds**: Optimized production builds with nginx serving layer

### **Registry Storage Innovation**
File-based state management system:
```go
type Client struct {
  ID, Domain, BrandName, Status, DBName, AdminEmail, AdminPassword, CookieSecret
  VendurePort, StorefrontPort int
  CreatedAt time.Time
}
```
Enables status monitoring, conflict prevention, and deployment history tracking.

## Storefront Technology Stack

### **Modern Frontend Architecture**
- **Framework**: React 18+ with Vite 5.x for lightning-fast development and builds
- **UI Library**: Shadcn/ui (Radix UI components) for professional, accessible design systems
- **Styling**: Tailwind CSS with PostCSS for utility-first responsive design
- **Forms**: React Hook Form + Zod validation for type-safe form handling
- **GraphQL Integration**: Vendure Shop API with generated TypeScript types
- **Production Serving**: Nginx static asset serving with optimized caching

### **Advanced Environment Configuration**
Dynamically set during provisioning in `docker-compose.yml`:
- `VENDURE_SHOP_API_URL=http://vendure:3000/shop-api` (internal service discovery)
- `VENDURE_CHANNEL_TOKEN=__default_channel__` (API authentication)
- `NEXT_PUBLIC_SITE_URL=http://localhost:{StorefrontPort}` (dynamic URL generation)
- `REVALIDATION_SECRET={cookie_secret}` (ISR cache invalidation security)
- `NEXT_PUBLIC_SITE_NAME={BrandName}` (dynamic brand customization)

### **Extensibility & Customization**
- **Commerce Components**: Modular component library in `src/components/commerce/` for Vendure-specific business logic
- **Authentication System**: Comprehensive auth context in `src/contexts/auth-context.tsx` for customer session management  
- **Search Integration**: Advanced search capabilities via Vendure Search API with filtering and faceting
- **Checkout Flow**: Complete order lifecycle integration with payment processing and fulfillment

## Critical Integration Points

### **Vendure Commerce Engine**
Enterprise-grade commerce backend:
- **Container Image**: `abduazizali/commercex:latest` (production-optimized Vendure distribution)
- **Admin Interface**: Full-featured admin panel accessible at `localhost:{VendurePort}`
- **Shop API**: GraphQL Shop API for storefront integration at internal `http://vendure:3000/shop-api`
- **Database Integration**: Auto-configured PostgreSQL connection via templated `vendure.env`
- **Health Orchestration**: PostgreSQL health checks ensure database readiness before Vendure startup

### **Production Infrastructure**
- **Multi-Stage Docker Builds**: Optimized build pipeline with node build stage and nginx serving stage
- **Container Health Management**: Sophisticated health check system with retry logic and timeout handling
- **Port Template Variables**: Dynamic port allocation using `{{.VendurePort}}` template substitution
- **Network Security**: Isolated Docker networks prevent cross-client communication

## Code Architecture Excellence

### **Go Module Organization**
- **Internal Packages**: Business logic isolated in `internal/` directory (registry, db, deploy, core, secrets, templates)
- **Thin CLI Layer**: `cmd/main.go` delegates to provisioner with minimal logic
- **Cobra Command Pattern**: Each operation (create, list, delete, status) as separate, focused command functions
- **Error Handling**: Descriptive error propagation with user-friendly progress indicators

### **Development Best Practices**
- **Progress Feedback**: Real-time deployment progress with ✓ checkmarks and status updates
- **Validation Pipeline**: Input validation, resource availability checks, and conflict prevention
- **Template Safety**: Safe template rendering with error handling and validation
- **Resource Cleanup**: Proper container and volume cleanup on deletion with data preservation options

## Testing & Production Verification

### **Deployment Verification Workflow**
1. **Registry Verification**: `grep clientID data/registry.json` - confirm client registration
2. **Container Status**: `docker ps | grep {clientID}` - verify all services running
3. **Service Logs**: `docker logs vendure_{clientID}` and `docker logs storefront_{clientID}` - check startup health
4. **Connectivity Test**: `curl http://localhost:{StorefrontPort}` - verify storefront accessibility
5. **API Health**: `curl http://localhost:{VendurePort}/health` - confirm Vendure API status

### **Common Issues & Solutions**
- **Port Conflicts**: Advanced `getNextPort()` algorithm prevents collisions across unlimited clients
- **Build Optimization**: Multi-stage Docker builds with nginx serving eliminate Node.js runtime issues
- **Database Connectivity**: Health check orchestration ensures PostgreSQL readiness before dependent services
- **Template Validation**: Comprehensive template data validation prevents deployment failures

## Platform Evolution & Extensibility

### **Future Expansion Capabilities**
1. **Multi-Region Deployment**: Template system ready for cloud provider integration
2. **SSL/TLS Integration**: Nginx configuration extensible for production SSL termination
3. **Monitoring Integration**: Registry system ready for metrics collection and alerting
4. **Auto-Scaling**: Container orchestration prepared for Kubernetes deployment
5. **Backup Automation**: Database volume management ready for automated backup strategies

### **Modification Guidelines**
1. **Client Lifecycle Enhancement**: Extend `Provisioner.Create()` workflow in [internal/core/provisioner.go](internal/core/provisioner.go)
2. **Environment Variables**: Add to `templateData` map in provisioner and update template files
3. **Port Management**: Modify allocation algorithm in `getNextPort()` for custom port strategies
4. **Command Addition**: Extend Cobra CLI in [cmd/main.go](cmd/main.go) with new provisioner methods
5. **Infrastructure Changes**: Update templates in [templates/docker-compose.yml.tmpl](templates/docker-compose.yml.tmpl) for new services

## Reference Architecture

### **Key Implementation Files**
- [internal/core/provisioner.go](internal/core/provisioner.go) - Core orchestration logic and client lifecycle management
- [cmd/innovatex/main.go](cmd/innovatex/main.go) - CLI interface and command definitions  
- [templates/docker-compose.yml.tmpl](templates/docker-compose.yml.tmpl) - Container orchestration definitions
- [internal/registry/store.go](internal/registry/store.go) - Client state persistence and management
- [internal/deploy/docker.go](internal/deploy/docker.go) - Docker container lifecycle operations

### **Achievement Summary**
The **InnovateX Provisioner** represents a breakthrough in e-commerce infrastructure automation, delivering enterprise-grade multi-tenant commerce platform provisioning through a single CLI command. This system successfully combines modern containerization, template-driven configuration, and production-ready frontend architecture to enable unlimited isolated commerce environments with zero manual configuration.

### Template Rendering
Files in `templates/` use Go's `text/template` syntax:
- `{{.ClientID}}`, `{{.DBName}}`, `{{.VendurePort}}` are injected from provisioner
- Rendered to `data/clients/{clientID}/` during Create workflow
- Use case: Environment variables and Docker Compose services require per-client customization

### Docker Compose Orchestration
- **Three-service stack**: PostgreSQL, Vendure (commercex), Storefront (Next.js)
- **Mount strategy**: Named volume for DB persistence (`postgres_data`)
- **Service discovery**: Internal network allows `vendure` hostname in Storefront API calls
- **Build args**: Storefront service builds from local Dockerfile (created during provision)

### Registry Storage Pattern
File-based JSON store (`data/registry.json`) tracks:
```go
type Client struct {
  ID, Domain, BrandName, Status, DBName, AdminEmail, AdminPassword, CookieSecret
  VendurePort, StorefrontPort int
  CreatedAt time.Time
}
```
Used for status checks, list operations, and preventing duplicate client IDs.

## Storefront Technology Stack

### **Modern Frontend Architecture**
- **Framework**: React 18+ with Vite 5.x for lightning-fast development and builds
- **UI Library**: Shadcn/ui (Radix UI components) for professional, accessible design systems
- **Styling**: Tailwind CSS with PostCSS for utility-first responsive design
- **Forms**: React Hook Form + Zod validation for type-safe form handling
- **GraphQL Integration**: Vendure Shop API with generated TypeScript types
- **Production Serving**: Nginx static asset serving with optimized caching

### **Advanced Environment Configuration**
Dynamically set during provisioning in `docker-compose.yml`:
- `VENDURE_SHOP_API_URL=http://vendure:3000/shop-api` (internal service discovery)
- `VENDURE_CHANNEL_TOKEN=__default_channel__` (API authentication)
- `NEXT_PUBLIC_SITE_URL=http://localhost:{StorefrontPort}` (dynamic URL generation)
- `REVALIDATION_SECRET={cookie_secret}` (ISR cache invalidation security)
- `NEXT_PUBLIC_SITE_NAME={BrandName}` (dynamic brand customization)

### **Extensibility & Customization**
- **Commerce Components**: Modular component library in `src/components/commerce/` for Vendure-specific business logic
- **Authentication System**: Comprehensive auth context in `src/contexts/auth-context.tsx` for customer session management  
- **Search Integration**: Advanced search capabilities via Vendure Search API with filtering and faceting
- **Checkout Flow**: Complete order lifecycle integration with payment processing and fulfillment

## Critical Integration Points

### Vendure Commerce Engine
- **Container**: `abduazizali/commercex:latest` (pre-built Vendure instance)
- **Admin API**: Accessible from host at `localhost:{VendurePort}`
- **Shop API**: Internal endpoint for Storefront (`http://vendure:3000/shop-api`)
- **Database**: PostgreSQL connection string auto-injected via `vendure.env` template

### Code Structure Rules
- **Go modules**: Place business logic in `internal/` packages (registry, db, deploy, core, secrets, templates)
- **CLI Layer**: Keep `cmd/main.go` thin—delegate to provisioner
- **Cobra Commands**: Each operation (create, list, delete, status) is a separate command function
- **Error Propagation**: Return descriptive errors early; provisioner logs progress with ✓ checkmarks

## Testing & Debugging

### Verify Provisioning Success
1. Check registry: `grep clientID data/registry.json`
2. List containers: `docker ps | grep {clientID}`
3. Check logs: `docker logs vendure_{clientID}` or `docker logs storefront_{clientID}`
4. Access storefront: `curl http://localhost:{StorefrontPort}`

### Common Issues
- **Port conflicts**: Review `getNextPort()` logic if ports collide
- **Docker build failures**: Check `storefront/Dockerfile` generation in `createStorefrontDockerfile()`
- **DB connection**: Verify PostgreSQL service is running and credentials match config
- **Template rendering**: Ensure template file path is correct and data keys match `{{.FieldName}}`

## When Modifying This Codebase

1. **Client lifecycle changes**: Update `Provisioner.Create()` flow in `internal/core/provisioner.go`
2. **New environment variables**: Add to `templateData` map in provisioner and update `.env.tmpl`
3. **Port allocation logic**: Modify `getNextPort()` in provisioner.go
4. **Adding commands**: Extend Cobra CLI in `cmd/main.go` and call provisioner methods
5. **Infrastructure changes**: Update Docker Compose template in `templates/docker-compose.yml.tmpl`

Reference key files:
- [internal/core/provisioner.go](internal/core/provisioner.go) - Orchestration logic
- [cmd/innovatex/main.go](cmd/innovatex/main.go) - CLI entry points
- [templates/docker-compose.yml.tmpl](templates/docker-compose.yml.tmpl) - Container definitions
