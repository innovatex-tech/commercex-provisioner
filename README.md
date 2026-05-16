# InnovateX CommerceX Provisioner 🚀

Enterprise-grade multi-tenant e-commerce provisioning platform. Instantly deploy isolated CommerceX stacks with production-ready React/Vite storefronts, dynamic port management, and automated Docker orchestration across local and remote servers.

**One command. Unlimited commerce environments. Zero manual configuration.**

---

## 🌟 Modern Experience

Unlike traditional CLI tools, InnovateX provides a **Premium TUI (Text User Interface)** experience:

*   **Interactive Wizard**: A beautiful multi-step setup with secure password masking and real-time validation.
*   **Live Dashboard**: A mission-control center to monitor container health, stream logs, and manage client lifecycle.
*   **Animated Provisioning**: High-fidelity animated progress bars with smooth gradients and step-by-step reporting.
*   **Minimalist Aesthetic**: Modern "Slate & Cyan" design inspired by high-end developer tools.

---

## 🚀 Key Features

### **Hybrid Deployment Orchestration**
- **Remote Cloud Support**: Deploy to any VPS over SSH with a single flag. Handles file syncing (tar-pipes) and remote Docker orchestration automatically.
- **Local Sandbox**: Provision full-stack environments on your local machine for rapid development and testing.
- **Atomic Cleanup**: The `--purge` flag ensures that when a client is deleted, all remote data, volumes, and directories are scrubbed clean.

### **Multi-Tenant Infrastructure Isolation**
- **Complete Isolation**: Each client gets a dedicated PostgreSQL database, private Docker network, and isolated container stack.
- **Dynamic Port Management**: Intelligent allocation (3-port blocks) prevents conflicts across unlimited concurrent deployments.
- **Domain Safety**: Built-in collision detection prevents ID or Domain conflicts before deployment begins.

### **Production-Ready Stack**
- **CommerceX Engine**: Node.js-based commerce platform with a full GraphQL API.
- **Dynamic Storefront**: React/Vite frontend with custom Nginx routing and deep-linking support (SPA fixes built-in).
- **Auto-Security**: Generates cryptographically secure admin credentials and session secrets automatically.

---

## 📋 Architecture

```
commercex-provisioner/
├── bin/innovatex                  # The compiled binary
├── cmd/innovatex/main.go          # CLI entry point
├── internal/
│   ├── core/provisioner.go        # Orchestration & Progress logic
│   ├── tui/                       # UI Components (Wizard, Dashboard, Progress)
│   ├── deploy/                    # Local & Remote (SSH) Orchestrators
│   ├── registry/                  # Client state & port management
│   └── templates/                 # Renderers for .env, nginx, and docker-compose
├── templates/                     # Source templates for deployments
└── data/                          # Local registry and client workspaces
```

---

## 📦 Installation

### Quick Install
One command installs everything (Go, Docker, and InnovateX):
```bash
curl -fsSL https://raw.githubusercontent.com/innovatex-tech/commercex-provisioner/main/install.sh | bash
```

### Manual Build
```bash
git clone https://github.com/innovatex-tech/commercex-provisioner.git
cd commercex-provisioner
go build -o bin/innovatex cmd/innovatex/main.go
```

---

## 🎯 Quick Start

### **1. Launch the Wizard**
Run the interactive setup to build your first store:
```bash
innovatex create
```

### **2. Deploy to Remote Server**
Bypass the UI and deploy directly to a production VPS:
```bash
innovatex create \
  --id=my-store \
  --domain=mystore.com \
  --server=root@45.147.x.x
```

### **3. Manage via Dashboard**
Open the mission control center to see all your stores:
```bash
innovatex dashboard
```
*   `↑/↓` to navigate clients
*   `Enter` to see details (Ports, URLs, Credentials)
*   `L` to stream real-time container logs
*   `T` to start/stop the stack
*   `D` to delete a client

---

## 📖 CLI Reference

| Command | Description |
| :--- | :--- |
| `innovatex create` | Starts the interactive provisioning wizard |
| `innovatex dashboard` | Opens the interactive management center |
| `innovatex list` | Quick list of all provisioned clients |
| `innovatex delete --id=<id>` | Removes containers and registry entry |
| `innovatex delete --id=<id> --purge` | **Deep Wipe**: Removes all remote files and database volumes |

---

## 🐛 Troubleshooting

*   **Port Conflicts**: Ensure ports 8000+ are available on the target machine.
*   **SSH Failures**: Ensure the remote server has `docker` and `docker compose` installed.
*   **Database Sync**: If the backend can't see the DB, check the `POSTGRES_DB` name in the Dashboard credentials.

---

## 📄 License
MIT License - See [LICENSE](LICENSE) file for details.

---

**Deploy unlimited commerce environments. One command at a time.**
