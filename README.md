# commercex-provisioner

Enterprise-grade multi-tenant e-commerce provisioning platform. Instantly deploy isolated CommerceX commerce stacks with production-ready React/Vite storefronts, dynamic port management, and automated Docker orchestration.

**One command. Unlimited commerce environments. Zero manual configuration.**

---

## 🚀 Key Features

### **Multi-Tenant Infrastructure Isolation**
- **Complete Isolation**: Each client gets dedicated PostgreSQL database, Docker network, and container stack
- **Zero Cross-Contamination**: Clients cannot interfere with each other's data, performance, or availability
- **Dynamic Port Allocation**: Intelligent port management system prevents conflicts across unlimited deployments

### **Production-Ready Deployment**
- **One-Command Provisioning**: Single CLI command provisions complete commerce infrastructure
- **Multi-Stage Docker Builds**: Optimized production builds with nginx serving for maximum performance
- **Health Orchestration**: PostgreSQL health checks ensure proper service startup sequencing
- **Cryptographic Security**: Automatically generates secure admin passwords and session secrets per client

### **Modern Commerce Stack**
- **CommerceX Commerce Engine**: Node.js-based commerce platform with full GraphQL API
- **React/Vite Storefront**: Lightning-fast frontend with component library and production serving
- **Service Discovery**: Internal Docker networks enable secure service-to-service communication
- **Persistent Storage**: Named volumes ensure database persistence across container lifecycles

### **Developer Experience**
- **Cobra CLI Interface**: Professional command-line interface with intuitive subcommands
- **Registry Management**: JSON-based client state tracking and conflict prevention
- **Real-Time Feedback**: Progress indicators and error reporting during deployments
- **Flexible Configuration**: Command-line flags for all client customization parameters

---

## 📋 Architecture

### **System Components**

```
commercex-provisioner/
├── cmd/innovatex/main.go          # CLI entry point and Cobra commands
├── internal/
│   ├── core/provisioner.go        # Orchestration logic and client lifecycle
│   ├── registry/store.go          # Client state persistence
│   ├── deploy/docker.go           # Container lifecycle management
│   ├── db/provisioner.go          # Database provisioning
│   ├── secrets/generator.go       # Cryptographic secret generation
│   └── templates/renderer.go      # Configuration rendering
├── templates/
│   ├── docker-compose.yml.tmpl    # Container orchestration definitions
│   ├── .env.tmpl                  # Commerce engine configuration
│   └── nginx.conf.tmpl            # Nginx reverse proxy configuration
└── data/
    ├── registry.json              # Client metadata store
    └── clients/                   # Per-client isolated environments
```

### **Deployment Architecture**

Each client deployment consists of four orchestrated services:

```yaml
Client Environment:
├── PostgreSQL Database (postgres_db)
│   ├── Isolated database per client
│   ├── Named volume for persistence
│   └── Health checks for startup sequencing
│
├── CommerceX Server (commercex-server)
│   ├── Main application server
│   ├── GraphQL API on port 3000
│   └── Depends on database health
│
├── CommerceX Worker (commercex-worker)
│   ├── Background job processing
│   └── Depends on database health
│
└── React/Vite Storefront
    ├── Production nginx serving
    ├── Multi-stage optimized build
    └── Internal API communication
```

### **Client Isolation Strategy**

- **Network Isolation**: Separate Docker network per client (`{clientID}_network`)
- **Database Separation**: Unique PostgreSQL database per client
- **Container Isolation**: Dedicated containers (`postgres_{clientID}`, `commercex_server_{clientID}`, `commercex_worker_{clientID}`, `storefront_{clientID}`)
- **Port Independence**: Dynamic allocation (`BasePort + 3*clientIndex`) - 3 ports per client

---

## 📦 Prerequisites

- **Go** 1.21+ ([install](https://golang.org/dl/))
- **Docker** 20.10+ and **Docker Compose** 2.0+ ([install](https://docs.docker.com/get-docker/))
- **Port Availability**: Default allocation starts at port 8000 (configurable)
- **Disk Space**: ~2GB per deployment (database + container images)

---

## 🔧 Installation

### 🚀 **Recommended: Global Install**

The simplest way—one command, no cloning needed:

```bash
go install github.com/innovatex-tech/commercex-provisioner/cmd/innovatex@latest
```

This puts `innovatex` in your Go bin directory (usually `~/go/bin`).

Verify installation:
```bash
innovatex --version    # Should output: innovatex version 1.0.0
innovatex --help       # Display available commands
```

**If you see 'command not found', add Go bin to your PATH:**
```bash
export PATH=$PATH:$(go env GOPATH)/bin
```

### Option 2: Clone and Build (for development)

For development or customization:

```bash
git clone https://github.com/innovatex-tech/commercex-provisioner.git
cd commercex-provisioner
go build -o bin/innovatex cmd/innovatex/main.go
./bin/innovatex --version
```

---

## 🎯 Quick Start

### **Deploy Your First Commerce Environment**

```bash
# Interactive mode (prompts for all values)
innovatex create

# Or with flags (also supports interactive prompts for missing values)
innovatex create \
  --id=mystore \
  --domain=mystore.local \
  --brand="My Store" \
  --db-name=mystore_db \
  --db-user=mystore_user \
  --db-password=secure_pass \
  --admin-user=admin \
  --admin-password=admin_pass
```

**What happens automatically:**
1. ✓ Generates cryptographically secure cookie secrets
2. ✓ Allocates unique ports (App: 8000, Postgres: 8001, Storefront: 8002)
3. ✓ Creates isolated PostgreSQL database
4. ✓ Clones and builds React/Vite storefront with nginx
5. ✓ Starts all 4 services with health check orchestration
6. ✓ Registers client in registry for ongoing management

### **Access Your Services**

```bash
# Storefront (React/Vite with nginx)
http://localhost:8002

# CommerceX Application
http://localhost:8000

# PostgreSQL (external access)
localhost:8001
```

---

## 📖 CLI Reference

### **Create a New Commerce Client**

```bash
# All flags are optional - CLI will prompt for missing values
innovatex create \
  --id=store-name \
  --domain=store-name.local \
  --brand="Store Display Name" \
  --db-name=store_db \
  --db-user=store_user \
  --db-password=db_pass123 \
  --admin-user=admin \
  --admin-password=admin_pass123
```

**Parameters:**
- `--id`: Unique client identifier (lowercase, alphanumeric, dashes)
- `--domain`: Client domain/hostname
- `--brand`: Display name for the storefront
- `--db-name`: PostgreSQL database name
- `--db-user`: Database username
- `--db-password`: Database password
- `--admin-user`: Admin panel username
- `--admin-password`: Admin panel password

**Output:**
```
Creating client: store-name
✓ Validation passed
✓ Secrets generated
✓ Assigned ports: App=8000, Postgres=8001, Storefront=8002
✓ Work directory created
✓ Storefront cloned
✓ Dockerfile created
✓ Templates rendered (env, nginx, docker-compose)
✓ Containers deployed
✓ Client registered

Application: http://localhost:8000
Storefront: http://localhost:8002
```

### **List All Clients**

```bash
innovatex list
```

Displays all provisioned clients with status, ports, and domains.

### **Check Client Status**

```bash
innovatex status --id=store-name
```

Shows container health, service status, and deployment information.

### **Delete a Client**

```bash
innovatex delete --id=store-name
```

Removes all containers, networks, and databases. Database data is preserved for recovery if needed.

---

## 🔐 Security & Secrets

### **Automatic Secret Generation**

For each client deployment, the provisioner automatically generates:

- **Admin Password**: 16-character cryptographically random string
- **Session Secret**: 32-byte random token for session management
- **Isolated Credentials**: Database credentials unique per client

**Access credentials via registry:**

```bash
grep -A 2 "store-name" data/registry.json | grep -E "AdminPassword|CookieSecret"
```

### **Best Practices**

1. ✓ Store admin passwords securely (password manager, secrets vault)
2. ✓ Change passwords after initial login via Vendure admin
3. ✓ Rotate session secrets periodically for production
4. ✓ Use environment variables for sensitive data in CI/CD

---

## 🛠️ Configuration

### **Environment Variables**

Customize provisioner behavior via environment variables:

```bash
# Client deployment directory (default: ~/.innovatex/clients/)
export INNOVATEX_WORK_DIR=/var/lib/innovatex/clients

# Template files location (default: ~/.innovatex/templates/)
export INNOVATEX_TEMPLATE_DIR=/etc/innovatex/templates

# Client registry path (default: ~/.innovatex/registry.json)
export INNOVATEX_REGISTRY=/var/lib/innovatex/registry.json
```

### **Template Customization**

Templates in `templates/` use Go's `text/template` syntax:

- `docker-compose.yml.tmpl`: Container orchestration definitions
- `.env.tmpl`: Commerce engine configuration
- `nginx.conf.tmpl`: Reverse proxy configuration

**Template variables available:**
- `{{.ClientID}}`: Unique client identifier
- `{{.DBName}}`: Database name
- `{{.DBUsername}}`: Database username
- `{{.DBPassword}}`: Database password
- `{{.AdminUsername}}`: Admin username
- `{{.AdminPassword}}`: Admin password
- `{{.CookieSecret}}`: Session secret
- `{{.AppPort}}`: CommerceX application port
- `{{.PostgresPort}}`: PostgreSQL external port
- `{{.StorefrontPort}}`: Storefront port
- `{{.BrandName}}`: Client brand name
- `{{.Domain}}`: Client domain

---

## 🚀 Production Deployment

### **Pre-Deployment Checklist**

- [ ] Docker and Docker Compose installed and running
- [ ] Sufficient disk space (2GB+ per client)
- [ ] Port ranges available and not conflicting with existing services
- [ ] PostgreSQL container images cached locally
- [ ] Firewall rules configured for public access (if needed)

### **Deployment Verification**

```bash
# Verify client provisioning
innovatex list

# Check container health
docker ps | grep {clientID}
# Expected: commercex_server_{clientID}, commercex_worker_{clientID}, postgres_{clientID}, storefront_{clientID}

# Monitor logs
docker logs commercex_server_{clientID} -f
docker logs storefront_{clientID} --tail=50

# Test storefront accessibility
curl -I http://localhost:{StorefrontPort}

# Check database connectivity
docker exec postgres_{clientID} pg_isready -U {db_username}
```

### **Multi-Client Scaling**

Deploy multiple isolated stores in sequence:

```bash
for i in {1..5}; do
  innovatex create \
    --id=store-$i \
    --domain=store-$i.local \
    --brand="Store $i" \
    --db-name=store_${i}_db \
    --db-user=store${i}_user \
    --db-password=pass${i} \
    --admin-user=admin \
    --admin-password=admin${i}
done

# Verify all deployments
innovatex list
```

Each deployment automatically allocates unique ports (3 per client) and isolated databases.

---

## 🐛 Troubleshooting

### **Common Issues**

#### **"Port already in use"**
Solution: The port allocation algorithm increments by 3 for each client (app, postgres, storefront). Verify no services occupy your port range:
```bash
netstat -tulpn | grep LISTEN
# Default starts at 8000: Client 1 uses 8000-8002, Client 2 uses 8003-8005, etc.
```

#### **"Docker build timeout"**
Solution: Multi-stage builds may take time on slower systems. Check storefront build logs:
```bash
docker logs storefront_{clientID} --follow
```

#### **"Database connection refused"**
Solution: PostgreSQL health checks may not have completed. Wait 10-15 seconds and verify:
```bash
docker logs postgres_{clientID}
# Check health: docker inspect postgres_{clientID} --format='{{.State.Health.Status}}'
```

#### **"Storefront shows blank page"**
Solution: Verify CommerceX API connectivity. Check server logs:
```bash
docker logs commercex_server_{clientID}
# Ensure CommerceX completed initialization
```

### **Debug Mode**

Enable detailed logging:

```bash
# View provisioner operations
set -x
./bin/innovatex create --id=debug-store --domain=debug.local --brand="Debug" --email=admin@debug.com
set +x

# View docker-compose logs
docker-compose -f data/clients/debug-store/docker-compose.yml logs -f
```

### **Recovery Procedures**

**Reset a client deployment:**
```bash
# Stop containers
docker-compose -f data/clients/{clientID}/docker-compose.yml down

# Remove containers and networks
docker system prune -f

# Re-provision
innovatex create --id={clientID} ...
```

**Backup database before deletion:**
```bash
docker exec postgres_{clientID} pg_dump -U {db_username} {db_name} > backup_{clientID}.sql
```

---

## 📚 Documentation

- **[Copilot Instructions](.github/copilot-instructions.md)**: Comprehensive AI assistant instructions for development
- **[Architecture Guide](docs/architecture.md)**: Detailed system design and component interactions
- **[API Reference](docs/api-reference.md)**: Vendure Shop API integration guide

---

## 🤝 Contributing

Contributions are welcome! Please follow these guidelines:

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/your-feature`
3. Commit your changes: `git commit -m "Add feature description"`
4. Push to the branch: `git push origin feature/your-feature`
5. Submit a pull request

### **Development Workflow**

```bash
# Build development binary
go build -o bin/innovatex cmd/innovatex/main.go

# Test provisioning workflow
./bin/innovatex create --id=test --domain=test.local --brand="Test" --email=test@test.com

# View logs and verify deployment
docker-compose -f data/clients/test/docker-compose.yml logs -f

# Clean up test deployment
./bin/innovatex delete --id=test
```

---

## 📊 Project Status

### **✅ Production Ready Features**
- Multi-tenant provisioning with complete isolation (4-container stack per client)
- Docker Compose orchestration with health checks
- Dynamic port allocation (3 ports per client)
- PostgreSQL database per client with external access
- CommerceX commerce engine integration (server + worker)
- React/Vite storefront deployment with nginx
- CLI interface with interactive prompts and flag support
- JSON registry for client state management
- Cryptographic secret generation (32-byte cookie secrets)
- Template-driven configuration system

### **⚠️ Known Limitations**
- Database provisioning logic exists but currently unused
- Manual database backup required before deletion

### **🔮 Roadmap**
- [ ] Kubernetes deployment support
- [ ] Multi-region scaling
- [ ] SSL/TLS termination integration
- [ ] Automated backup and recovery
- [ ] Metrics and monitoring dashboard
- [ ] Payment gateway pre-configuration
- [ ] Theme customization templates

---

## 📄 License

MIT License - See [LICENSE](LICENSE) file for details

---

## 🙋 Support

- **Issues**: [GitHub Issues](https://github.com/innovatex-tech/commercex-provisioner/issues)
- **Discussions**: [GitHub Discussions](https://github.com/innovatex-tech/commercex-provisioner/discussions)
- **Email**: support@innovatex-tech.com

---

## 🌟 Acknowledgments

Built with modern cloud-native technologies:
- [CommerceX](https://github.com/abduazizali/commercex) - Node.js commerce platform
- [Docker](https://www.docker.com/) - Container orchestration
- [React](https://react.dev/) - UI framework
- [Vite](https://vitejs.dev/) - Lightning-fast build tool
- [Cobra](https://cobra.dev/) - CLI framework
- [PostgreSQL](https://www.postgresql.org/) - Database engine

---

**Deploy unlimited commerce environments. One command at a time.**
