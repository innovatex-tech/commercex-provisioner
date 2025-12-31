# commercex-provisioner

Enterprise-grade multi-tenant e-commerce provisioning platform. Instantly deploy isolated Vendure commerce stacks with production-ready React/Vite storefronts, dynamic port management, and automated Docker orchestration.

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
- **Vendure Commerce Engine**: Industry-leading headless commerce platform with full GraphQL API
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
│   └── vendure.env.tmpl           # Commerce engine configuration
└── data/
    ├── registry.json              # Client metadata store
    └── clients/                   # Per-client isolated environments
```

### **Deployment Architecture**

Each client deployment consists of three orchestrated services:

```yaml
Client Environment:
├── PostgreSQL Database
│   ├── Isolated schema per client
│   ├── Named volume for persistence
│   └── Health checks for startup sequencing
│
├── Vendure Commerce Engine
│   ├── GraphQL Admin & Shop APIs
│   ├── Service discovery via internal network
│   └── Depends on database health
│
└── React/Vite Storefront
    ├── Production nginx serving
    ├── Multi-stage optimized build
    └── Internal API communication
```

### **Client Isolation Strategy**

- **Network Isolation**: Separate Docker network per client (`{clientID}_network`)
- **Database Separation**: Unique PostgreSQL database (`vendure_{clientID}`)
- **Container Isolation**: Dedicated containers (`postgres_{clientID}`, `vendure_{clientID}`, `storefront_{clientID}`)
- **Port Independence**: Dynamic allocation (`BasePort + 2*clientIndex`)

---

## 📦 Prerequisites

- **Go** 1.21+ ([install](https://golang.org/dl/))
- **Docker** 20.10+ and **Docker Compose** 2.0+ ([install](https://docs.docker.com/get-docker/))
- **Port Availability**: Default allocation starts at port 8000 (configurable)
- **Disk Space**: ~2GB per deployment (database + container images)

---

## 🔧 Installation

### **1. Clone the Repository**

```bash
git clone https://github.com/innovatex-tech/commercex-provisioner.git
cd commercex-provisioner
```

### **2. Build the Binary**

```bash
go build -o bin/innovatex cmd/innovatex/main.go
```

### **3. Verify Installation**

```bash
./bin/innovatex --version
./bin/innovatex --help
```

---

## 🎯 Quick Start

### **Deploy Your First Commerce Environment**

```bash
./bin/innovatex create \
  --id=mystore \
  --domain=mystore.local \
  --brand="My Store" \
  --email=admin@mystore.com
```

**What happens automatically:**
1. ✓ Generates cryptographically secure admin password and session secrets
2. ✓ Allocates unique ports (Vendure on 8000, Storefront on 8001)
3. ✓ Creates isolated PostgreSQL database
4. ✓ Builds and deploys React/Vite storefront with nginx
5. ✓ Starts all services with health check orchestration
6. ✓ Registers client in registry for ongoing management

### **Access Your Storefront**

```bash
# Storefront (React/Vite with nginx)
http://localhost:8001

# Vendure Admin Panel
http://localhost:8000

# GraphQL Shop API (internal)
http://vendure:3000/shop-api
```

---

## 📖 CLI Reference

### **Create a New Commerce Client**

```bash
./bin/innovatex create \
  --id=store-name \
  --domain=store-name.local \
  --brand="Store Display Name" \
  --email=admin@store-name.com
```

**Parameters:**
- `--id` (required): Unique client identifier
- `--domain` (required): Client domain/hostname
- `--brand` (required): Display name for the storefront
- `--email` (required): Admin email address

**Output:**
```
Creating commerce environment for client: store-name
✓ Generated secure secrets
✓ Allocated ports: 8000 (Vendure), 8001 (Storefront)
✓ Created database: vendure_store_name
✓ Built storefront container
✓ Started services with health checks
✓ Client registered in registry

Admin Panel: http://localhost:8000
Storefront: http://localhost:8001
```

### **List All Clients**

```bash
./bin/innovatex list
```

Displays all provisioned clients with status, ports, and domains.

### **Check Client Status**

```bash
./bin/innovatex status --id=store-name
```

Shows container health, service status, and deployment information.

### **Delete a Client**

```bash
./bin/innovatex delete --id=store-name
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
# Custom base port (default: 8000)
export BASE_PORT=9000

# Custom storage path (default: ./data)
export DATA_PATH=/var/lib/commercex

# Database credentials
export DB_USER=vendure_admin
export DB_PASSWORD=secure_password
```

### **Template Customization**

Templates in `templates/` use Go's `text/template` syntax:

- `docker-compose.yml.tmpl`: Container orchestration definitions
- `vendure.env.tmpl`: Commerce engine configuration

**Template variables available:**
- `{{.ClientID}}`: Unique client identifier
- `{{.DBName}}`: Database name
- `{{.AdminPassword}}`: Generated admin password
- `{{.CookieSecret}}`: Session secret
- `{{.VendurePort}}`: Allocated Vendure port
- `{{.StorefrontPort}}`: Allocated storefront port
- `{{.BrandName}}`: Client brand name

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
./bin/innovatex list

# Check container health
docker ps | grep {clientID}

# Monitor logs
docker logs vendure_{clientID}
docker logs storefront_{clientID}

# Test storefront accessibility
curl -I http://localhost:{StorefrontPort}

# Verify Vendure API
curl http://localhost:{VendurePort}/health
```

### **Multi-Client Scaling**

Deploy multiple isolated stores in sequence:

```bash
for i in {1..5}; do
  ./bin/innovatex create \
    --id=store-$i \
    --domain=store-$i.local \
    --brand="Store $i" \
    --email=admin@store-$i.com
done

# Verify all deployments
./bin/innovatex list
```

Each deployment automatically allocates unique ports and databases.

---

## 🐛 Troubleshooting

### **Common Issues**

#### **"Port already in use"**
Solution: The port allocation algorithm increments by 2 for each client. Verify no services occupy your port range:
```bash
netstat -tulpn | grep LISTEN
# Adjust BASE_PORT environment variable if needed
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
```

#### **"Storefront shows blank page"**
Solution: Verify GraphQL API connectivity. Check Vendure logs:
```bash
docker logs vendure_{clientID}
# Ensure Vendure completed initialization
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
./bin/innovatex create --id={clientID} ...
```

**Backup database before deletion:**
```bash
docker exec postgres_{clientID} pg_dump -U vendure_admin vendure_{clientID} > backup_{clientID}.sql
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
- Multi-tenant provisioning with complete isolation
- Docker Compose orchestration with health checks
- Dynamic port allocation and management
- PostgreSQL database per client
- Vendure commerce engine integration
- React/Vite storefront deployment
- CLI interface with all CRUD operations
- JSON registry for client state management
- Cryptographic secret generation

### **⚠️ Known Limitations**
- Vendure schema auto-sync may require manual intervention in some containerized environments
- Database migrations should be validated before production deployments

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
- [Vendure](https://www.vendure.io/) - Headless commerce platform
- [Docker](https://www.docker.com/) - Container orchestration
- [React](https://react.dev/) - UI framework
- [Vite](https://vitejs.dev/) - Lightning-fast build tool
- [Cobra](https://cobra.dev/) - CLI framework

---

**Deploy unlimited commerce environments. One command at a time.**
