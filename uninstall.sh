#!/bin/bash
#
# mControlPanel Uninstall Script
# ลบ mControlPanel ออกจากระบบ
#

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Variables
BIN_PATH="/usr/local/bin/mcontrolpanel"
SERVICE_FILE="/etc/systemd/system/mcontrolpanel.service"
CONFIG_DIR="/etc/mcontrolpanel"
DATA_DIR="/var/lib/mcontrolpanel"
LOG_DIR="/var/log/mcontrolpanel"
BACKUP_DIR="/var/backups/mcontrolpanel"
INSTALL_DIR="/opt/mcontrolpanel"

print_banner() {
    echo -e "${RED}"
    echo "╔══════════════════════════════════════════════════════╗"
    echo "║        mControlPanel Uninstaller                     ║"
    echo "╚══════════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

check_root() {
    if [ "$EUID" -ne 0 ]; then
        echo -e "${RED}กรุณารันด้วย root หรือ sudo${NC}"
        exit 1
    fi
}

confirm_uninstall() {
    echo -e "${YELLOW}คำเตือน: การดำเนินการนี้จะลบ mControlPanel ทั้งหมด${NC}"
    echo ""
    echo "สิ่งที่จะถูกลบ:"
    echo "  - Binary: $BIN_PATH"
    echo "  - Service: $SERVICE_FILE"
    echo "  - Config: $CONFIG_DIR"
    echo "  - Data: $DATA_DIR"
    echo "  - Logs: $LOG_DIR"
    echo ""
    
    read -p "ต้องการลบ Backups ด้วยหรือไม่? (y/N): " DELETE_BACKUPS
    read -p "ต้องการลบ /var/www ด้วยหรือไม่? (y/N): " DELETE_WWW
    
    echo ""
    read -p "ยืนยันการลบ mControlPanel? (y/N): " CONFIRM
    
    if [[ ! "$CONFIRM" =~ ^[Yy]$ ]]; then
        echo -e "${BLUE}ยกเลิกการลบ${NC}"
        exit 0
    fi
}

stop_service() {
    echo -e "${YELLOW}[1/5] หยุด service...${NC}"
    
    if systemctl is-active --quiet mcontrolpanel; then
        systemctl stop mcontrolpanel
    fi
    
    if systemctl is-enabled --quiet mcontrolpanel 2>/dev/null; then
        systemctl disable mcontrolpanel
    fi
    
    echo -e "${GREEN}✓ หยุด service สำเร็จ${NC}"
}

remove_files() {
    echo -e "${YELLOW}[2/5] ลบไฟล์...${NC}"
    
    # Remove binary
    [ -f "$BIN_PATH" ] && rm -f "$BIN_PATH" && echo "  ลบ $BIN_PATH"
    
    # Remove service file
    [ -f "$SERVICE_FILE" ] && rm -f "$SERVICE_FILE" && echo "  ลบ $SERVICE_FILE"
    
    # Remove install dir
    [ -d "$INSTALL_DIR" ] && rm -rf "$INSTALL_DIR" && echo "  ลบ $INSTALL_DIR"
    
    systemctl daemon-reload
    
    echo -e "${GREEN}✓ ลบไฟล์สำเร็จ${NC}"
}

remove_config() {
    echo -e "${YELLOW}[3/5] ลบ config...${NC}"
    
    [ -d "$CONFIG_DIR" ] && rm -rf "$CONFIG_DIR" && echo "  ลบ $CONFIG_DIR"
    
    echo -e "${GREEN}✓ ลบ config สำเร็จ${NC}"
}

remove_data() {
    echo -e "${YELLOW}[4/5] ลบ data...${NC}"
    
    [ -d "$DATA_DIR" ] && rm -rf "$DATA_DIR" && echo "  ลบ $DATA_DIR"
    [ -d "$LOG_DIR" ] && rm -rf "$LOG_DIR" && echo "  ลบ $LOG_DIR"
    
    if [[ "$DELETE_BACKUPS" =~ ^[Yy]$ ]]; then
        [ -d "$BACKUP_DIR" ] && rm -rf "$BACKUP_DIR" && echo "  ลบ $BACKUP_DIR"
    else
        echo "  เก็บ $BACKUP_DIR ไว้"
    fi
    
    if [[ "$DELETE_WWW" =~ ^[Yy]$ ]]; then
        [ -d "/var/www" ] && rm -rf "/var/www" && echo "  ลบ /var/www"
    else
        echo "  เก็บ /var/www ไว้"
    fi
    
    echo -e "${GREEN}✓ ลบ data สำเร็จ${NC}"
}

cleanup() {
    echo -e "${YELLOW}[5/5] ทำความสะอาด...${NC}"
    
    # Remove any leftover files
    rm -f /tmp/mcontrolpanel* 2>/dev/null || true
    
    echo -e "${GREEN}✓ ทำความสะอาดสำเร็จ${NC}"
}

print_completion() {
    echo ""
    echo -e "${GREEN}╔══════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║          ลบ mControlPanel สำเร็จแล้ว!                ║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════════════════════╝${NC}"
    echo ""
    
    if [[ ! "$DELETE_BACKUPS" =~ ^[Yy]$ ]]; then
        echo -e "${YELLOW}Backups ยังคงอยู่ที่: $BACKUP_DIR${NC}"
    fi
    
    if [[ ! "$DELETE_WWW" =~ ^[Yy]$ ]]; then
        echo -e "${YELLOW}Website files ยังคงอยู่ที่: /var/www${NC}"
    fi
    
    echo ""
    echo -e "${BLUE}หมายเหตุ: Nginx, MySQL และ PHP-FPM ยังคงติดตั้งอยู่${NC}"
    echo -e "${BLUE}หากต้องการลบ ให้รัน:${NC}"
    echo -e "  apt remove --purge nginx mysql-server php-fpm"
    echo ""
}

main() {
    print_banner
    check_root
    confirm_uninstall
    
    echo ""
    stop_service
    remove_files
    remove_config
    remove_data
    cleanup
    print_completion
}

main "$@"
