#!/bin/bash
#
# mControlPanel Installation Script
# ติดตั้ง mControlPanel อัตโนมัติสำหรับ Linux
#

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Variables
VERSION="1.0.0"
INSTALL_DIR="/opt/mcontrolpanel"
BIN_PATH="/usr/local/bin/mcontrolpanel"
SERVICE_FILE="/etc/systemd/system/mcontrolpanel.service"
CONFIG_FILE="/etc/mcontrolpanel/config.yaml"
DATA_DIR="/var/lib/mcontrolpanel"

# Detect OS
detect_os() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS=$ID
        VERSION_ID=$VERSION_ID
    else
        echo -e "${RED}ไม่สามารถระบุ OS ได้${NC}"
        exit 1
    fi
}

# Print banner
print_banner() {
    echo -e "${BLUE}"
    echo "╔══════════════════════════════════════════════════════╗"
    echo "║          mControlPanel Installer v${VERSION}            ║"
    echo "║       Lightweight Web Hosting Control Panel          ║"
    echo "╚══════════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

# Check root
check_root() {
    if [ "$EUID" -ne 0 ]; then
        echo -e "${RED}กรุณารันด้วย root หรือ sudo${NC}"
        exit 1
    fi
}

# Install dependencies based on OS
install_dependencies() {
    echo -e "${YELLOW}[1/6] ติดตั้ง dependencies...${NC}"
    
    case $OS in
        ubuntu|debian)
            apt-get update -qq
            apt-get install -y -qq nginx mysql-server php-fpm php-mysql php-curl php-gd php-mbstring php-xml php-zip curl tar wget
            ;;
        centos|rhel|fedora|rocky|almalinux)
            if command -v dnf &> /dev/null; then
                dnf install -y nginx mysql-server php-fpm php-mysqlnd php-curl php-gd php-mbstring php-xml php-zip curl tar wget
            else
                yum install -y nginx mysql-server php-fpm php-mysqlnd php-curl php-gd php-mbstring php-xml php-zip curl tar wget
            fi
            ;;
        arch|manjaro)
            pacman -Sy --noconfirm nginx mysql php php-fpm curl tar wget
            ;;
        *)
            echo -e "${RED}OS '$OS' ไม่รองรับ${NC}"
            exit 1
            ;;
    esac
    
    echo -e "${GREEN}✓ ติดตั้ง dependencies สำเร็จ${NC}"
}

# Start services
start_services() {
    echo -e "${YELLOW}[2/6] เริ่มต้น services...${NC}"
    
    systemctl enable nginx mysql 2>/dev/null || true
    systemctl start nginx mysql 2>/dev/null || true
    
    # Start PHP-FPM (different names on different distros)
    for PHP_FPM in php-fpm php8.3-fpm php8.2-fpm php8.1-fpm php8.0-fpm php7.4-fpm; do
        if systemctl list-unit-files | grep -q "^$PHP_FPM"; then
            systemctl enable $PHP_FPM 2>/dev/null || true
            systemctl start $PHP_FPM 2>/dev/null || true
            break
        fi
    done
    
    echo -e "${GREEN}✓ เริ่มต้น services สำเร็จ${NC}"
}

# Download and install mControlPanel
install_mcontrolpanel() {
    echo -e "${YELLOW}[3/6] ดาวน์โหลด mControlPanel...${NC}"
    
    # Create directories
    mkdir -p $INSTALL_DIR
    mkdir -p /etc/mcontrolpanel
    mkdir -p $DATA_DIR
    mkdir -p /var/www
    mkdir -p /var/backups/mcontrolpanel
    mkdir -p /var/log/mcontrolpanel
    
    # Detect architecture
    ARCH=$(uname -m)
    case $ARCH in
        x86_64)
            BINARY_NAME="mcontrolpanel-linux-amd64"
            ;;
        aarch64|arm64)
            BINARY_NAME="mcontrolpanel-linux-arm64"
            ;;
        *)
            echo -e "${RED}สถาปัตยกรรม '$ARCH' ไม่รองรับ${NC}"
            exit 1
            ;;
    esac
    
    # Download binary
    DOWNLOAD_URL="https://github.com/thanadon-dev/mcontrolpanel/releases/download/v${VERSION}/${BINARY_NAME}"
    
    echo "กำลังดาวน์โหลดจาก: $DOWNLOAD_URL"
    
    if curl -fsSL "$DOWNLOAD_URL" -o "$BIN_PATH"; then
        chmod +x "$BIN_PATH"
        echo -e "${GREEN}✓ ดาวน์โหลดสำเร็จ${NC}"
    else
        echo -e "${YELLOW}ไม่สามารถดาวน์โหลดได้ กำลัง build จาก source...${NC}"
        build_from_source
    fi
}

# Build from source if download fails
build_from_source() {
    echo -e "${YELLOW}กำลัง build จาก source code...${NC}"
    
    # Install Go if not present
    if ! command -v go &> /dev/null; then
        echo "ติดตั้ง Go..."
        wget -q https://go.dev/dl/go1.21.6.linux-amd64.tar.gz -O /tmp/go.tar.gz
        tar -C /usr/local -xzf /tmp/go.tar.gz
        export PATH=$PATH:/usr/local/go/bin
        rm /tmp/go.tar.gz
    fi
    
    # Clone and build
    cd /tmp
    rm -rf mcontrolpanel
    git clone https://github.com/thanadon-dev/mcontrolpanel.git
    cd mcontrolpanel
    go build -ldflags="-s -w" -o "$BIN_PATH" .
    cd /
    rm -rf /tmp/mcontrolpanel
    
    echo -e "${GREEN}✓ Build สำเร็จ${NC}"
}

# Create config file
create_config() {
    echo -e "${YELLOW}[4/6] สร้างไฟล์ config...${NC}"
    
    # Generate random secret key
    SECRET_KEY=$(head -c 32 /dev/urandom | base64 | tr -dc 'a-zA-Z0-9' | head -c 32)
    
    cat > $CONFIG_FILE << EOF
server:
  host: 0.0.0.0
  port: 8080
  secret_key: ${SECRET_KEY}
  enable_https: false
  cert_file: ""
  key_file: ""

database:
  path: ${DATA_DIR}/panel.db
  mysql_host: localhost
  mysql_port: 3306
  mysql_user: root
  mysql_pass: ""

paths:
  www_root: /var/www
  backup_dir: /var/backups/mcontrolpanel
  log_dir: /var/log/mcontrolpanel
  ssl_dir: /etc/ssl/mcontrolpanel
  nginx_conf: /etc/nginx/sites-enabled

services:
  webserver: nginx

php:
  default_version: "8.2"
  versions: ["7.4", "8.0", "8.1", "8.2", "8.3"]

backup:
  retention_days: 7
  compress: true

rate_limit:
  enabled: true
  requests_per_minute: 60
  login_attempts: 5
EOF
    
    chmod 600 $CONFIG_FILE
    echo -e "${GREEN}✓ สร้าง config สำเร็จ${NC}"
}

# Create systemd service
create_service() {
    echo -e "${YELLOW}[5/6] สร้าง systemd service...${NC}"
    
    cat > $SERVICE_FILE << EOF
[Unit]
Description=mControlPanel - Lightweight Web Hosting Control Panel
Documentation=https://github.com/thanadon-dev/mcontrolpanel
After=network.target mysql.service nginx.service
Wants=mysql.service nginx.service

[Service]
Type=simple
User=root
Group=root
WorkingDirectory=${INSTALL_DIR}
ExecStart=${BIN_PATH} --config ${CONFIG_FILE}
ExecReload=/bin/kill -HUP \$MAINPID
Restart=always
RestartSec=5
StandardOutput=append:/var/log/mcontrolpanel/mcontrolpanel.log
StandardError=append:/var/log/mcontrolpanel/error.log

# Security hardening
NoNewPrivileges=false
ProtectSystem=false
ProtectHome=false
PrivateTmp=true

# Resource limits
LimitNOFILE=65535
MemoryMax=128M
CPUQuota=100%

[Install]
WantedBy=multi-user.target
EOF
    
    systemctl daemon-reload
    echo -e "${GREEN}✓ สร้าง service สำเร็จ${NC}"
}

# Run initial setup
run_setup() {
    echo -e "${YELLOW}[6/6] รัน initial setup...${NC}"
    
    $BIN_PATH --config $CONFIG_FILE --setup
    
    echo -e "${GREEN}✓ Setup สำเร็จ${NC}"
}

# Start mControlPanel
start_panel() {
    echo -e "${YELLOW}เริ่มต้น mControlPanel...${NC}"
    
    systemctl enable mcontrolpanel
    systemctl start mcontrolpanel
    
    echo -e "${GREEN}✓ mControlPanel กำลังทำงาน${NC}"
}

# Print completion message
print_completion() {
    LOCAL_IP=$(hostname -I | awk '{print $1}')
    
    echo ""
    echo -e "${GREEN}╔══════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║          ติดตั้ง mControlPanel สำเร็จแล้ว!           ║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "🌐 เข้าใช้งาน Panel ได้ที่:"
    echo -e "   ${BLUE}http://localhost:8080${NC}"
    echo -e "   ${BLUE}http://${LOCAL_IP}:8080${NC}"
    echo ""
    echo -e "📁 ไฟล์สำคัญ:"
    echo -e "   Config: ${CONFIG_FILE}"
    echo -e "   Data:   ${DATA_DIR}"
    echo -e "   Logs:   /var/log/mcontrolpanel/"
    echo ""
    echo -e "🛠️  คำสั่งที่ใช้บ่อย:"
    echo -e "   ${YELLOW}systemctl status mcontrolpanel${NC}  - ดูสถานะ"
    echo -e "   ${YELLOW}systemctl restart mcontrolpanel${NC} - รีสตาร์ท"
    echo -e "   ${YELLOW}journalctl -u mcontrolpanel -f${NC}  - ดู logs"
    echo ""
    echo -e "⚠️  หมายเหตุด้านความปลอดภัย:"
    echo -e "   - เปลี่ยนรหัสผ่านเริ่มต้นทันที"
    echo -e "   - ตั้งค่า firewall ให้เปิดเฉพาะ port ที่จำเป็น"
    echo -e "   - ใช้ HTTPS ใน production"
    echo ""
}

# Main installation
main() {
    print_banner
    check_root
    detect_os
    
    echo -e "${BLUE}ระบบปฏิบัติการ: $OS $VERSION_ID${NC}"
    echo ""
    
    install_dependencies
    start_services
    install_mcontrolpanel
    create_config
    create_service
    run_setup
    start_panel
    print_completion
}

# Run
main "$@"
