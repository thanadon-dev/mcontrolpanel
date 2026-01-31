# mControlPanel

A lightweight, high-performance web hosting control panel written in Go. Single binary deployment with minimal resource usage.

## ✨ Features

- **Single Binary** - No runtime dependencies, just one executable
- **Low Memory** - Uses ~10-20MB RAM (vs 200MB+ for Python/Node alternatives)
- **Fast** - Instant startup, sub-millisecond response times
- **Cross-Platform** - Works on Linux and Windows
- **Unlimited Domains** - Create and manage virtual hosts
- **MySQL Management** - Full database CRUD operations
- **WordPress Installer** - One-click WordPress deployment
- **PHP Multi-Version** - Support for PHP 7.4, 8.0, 8.1, 8.2, 8.3
- **Backup System** - Full/files/database backups
- **Service Control** - Start/stop/restart Nginx, MySQL, PHP-FPM
- **Modern UI** - Clean, responsive dark-themed interface

## 📊 Resource Comparison

| Panel | Language | Binary Size | RAM Usage | Startup |
|-------|----------|-------------|-----------|---------|
| **mControlPanel** | Go | ~15MB | ~15MB | <1s |
| cPanel | Perl/C | N/A | 500MB+ | 30s+ |
| Plesk | PHP | N/A | 400MB+ | 20s+ |
| Similar Python Panel | Python | ~50MB+ deps | 200MB+ | 5s+ |

## 🚀 Quick Install

### Linux (Ubuntu/Debian)

```bash
# Install dependencies
sudo apt update && sudo apt install -y nginx mysql-server php-fpm curl

# Download mControlPanel
curl -Lo mcontrolpanel https://github.com/thanadon-dev/mcontrolpanel/releases/latest/download/mcontrolpanel-linux-amd64
chmod +x mcontrolpanel

# Run setup
sudo ./mcontrolpanel --setup

# Start the panel
sudo ./mcontrolpanel
```

### Linux (CentOS/RHEL/Fedora)

```bash
# Install dependencies
sudo dnf install -y nginx mysql-server php-fpm curl

# Start services
sudo systemctl enable --now nginx mysqld php-fpm

# Download mControlPanel
curl -Lo mcontrolpanel https://github.com/thanadon-dev/mcontrolpanel/releases/latest/download/mcontrolpanel-linux-amd64
chmod +x mcontrolpanel

# Run setup
sudo ./mcontrolpanel --setup

# Start the panel
sudo ./mcontrolpanel
```

### Windows

```powershell
# Install Chocolatey (if not installed)
Set-ExecutionPolicy Bypass -Scope Process -Force
[System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072
iex ((New-Object System.Net.WebClient).DownloadString('https://community.chocolatey.org/install.ps1'))

# Install dependencies
choco install nginx mysql php -y

# Download mControlPanel
Invoke-WebRequest -Uri "https://github.com/thanadon-dev/mcontrolpanel/releases/latest/download/mcontrolpanel-windows-amd64.exe" -OutFile "mcontrolpanel.exe"

# Run setup
.\mcontrolpanel.exe --setup

# Start the panel
.\mcontrolpanel.exe
```

### Build from Source

```bash
# Clone repository
git clone https://github.com/thanadon-dev/mcontrolpanel.git
cd mcontrolpanel

# Build
go build -ldflags="-s -w" -o mcontrolpanel .

# Run
./mcontrolpanel --setup
```

## 📁 Project Structure

```
mcontrolpanel/
├── main.go                    # Entry point
├── go.mod                     # Go modules
├── internal/
│   ├── config/config.go       # Configuration management
│   ├── database/database.go   # SQLite database & models
│   ├── server/server.go       # HTTP server setup
│   ├── handlers/handlers.go   # Request handlers
│   └── middleware/middleware.go # Auth middleware
└── web/
    ├── templates/             # HTML templates
    └── static/                # CSS/JS assets
```

## ⚙️ Configuration

Create `config.yaml` in the same directory:

```yaml
server:
  host: 127.0.0.1
  port: 8080
  secret_key: your-secret-key-here

database:
  path: data/panel.db
  mysql_host: localhost
  mysql_port: 3306
  mysql_user: root
  mysql_pass: ""

paths:
  www_root: /var/www
  backup_dir: /var/backups/mcontrolpanel
  nginx_conf: /etc/nginx/sites-enabled

php:
  default_version: "8.2"
  versions: ["7.4", "8.0", "8.1", "8.2", "8.3"]
```

## 🔧 Command Line Options

```
Usage: mcontrolpanel [options]

Options:
  --config string   Path to config file (default "config.yaml")
  --host string     Override server host
  --port int        Override server port
  --setup           Run initial setup wizard
  --version         Show version information
```

## 🖥️ Usage

1. Open your browser to `http://127.0.0.1:8080`
2. Login with your admin credentials
3. Start managing your domains, databases, and WordPress sites!

## 🔒 Security Notes

- Change the default `secret_key` in production
- Run behind a reverse proxy (Nginx) with HTTPS in production
- Restrict panel access to trusted IPs
- Use strong admin passwords

## 🛠️ Development

```bash
# Run in development mode
go run . --config config.yaml

# Build for production
go build -ldflags="-s -w" -o mcontrolpanel .

# Cross-compile
GOOS=linux GOARCH=amd64 go build -o mcontrolpanel-linux-amd64 .
GOOS=windows GOARCH=amd64 go build -o mcontrolpanel-windows-amd64.exe .
```

## 📝 License

MIT License - feel free to use for personal or commercial projects.

## 🤝 Contributing

Contributions welcome! Please open an issue or pull request.

---

**mControlPanel** - Lightweight hosting made simple.
