# mControlPanel

แผงควบคุมเว็บโฮสติ้งที่เบาและเร็ว เขียนด้วยภาษา Go - ติดตั้งง่าย ใช้ทรัพยากรน้อย

## ✨ คุณสมบัติ

- **ไฟล์เดียวจบ** - ไม่ต้องติดตั้ง runtime เพิ่มเติม แค่ไฟล์ exe/binary ไฟล์เดียว
- **ประหยัด RAM** - ใช้แค่ ~10MB (เทียบกับ Python/Node ที่ใช้ 200MB+)
- **เร็วมาก** - เริ่มต้นทันที ตอบสนองภายในมิลลิวินาที
- **ข้ามแพลตฟอร์ม** - ใช้ได้ทั้ง Linux และ Windows
- **โดเมนไม่จำกัด** - สร้างและจัดการ virtual hosts ได้เท่าที่ต้องการ
- **จัดการ MySQL** - สร้าง/ลบ/จัดการฐานข้อมูลได้ครบ
- **ติดตั้ง WordPress** - คลิกเดียวติดตั้ง WordPress ได้เลย
- **รองรับ PHP หลายเวอร์ชัน** - PHP 7.4, 8.0, 8.1, 8.2, 8.3
- **ระบบ Backup** - สำรองข้อมูลทั้งไฟล์และฐานข้อมูล
- **ควบคุม Services** - Start/Stop/Restart Nginx, MySQL, PHP-FPM
- **UI ทันสมัย** - หน้าตาสวย ใช้งานง่าย โทนสีเข้ม
- **🆕 Rate Limiting** - ป้องกัน brute force อัตโนมัติ
- **🆕 HTTPS Support** - รองรับ SSL/TLS และ Let's Encrypt
- **🆕 Health Check API** - `/health`, `/ready`, `/live` สำหรับ monitoring

## 📊 เปรียบเทียบทรัพยากร

| Panel | ภาษา | ขนาด Binary | RAM | เวลาเริ่มต้น |
|-------|------|-------------|-----|-------------|
| **mControlPanel** | Go | ~5MB (UPX) | ~10MB | <1 วินาที |
| cPanel | Perl/C | N/A | 500MB+ | 30+ วินาที |
| Plesk | PHP | N/A | 400MB+ | 20+ วินาที |
| Python Panel อื่นๆ | Python | 50MB+ deps | 200MB+ | 5+ วินาที |

---

## 🚀 วิธีติดตั้ง

### ⚡ ติดตั้งด่วน (One-Line Install)

**Linux:**
```bash
curl -fsSL https://raw.githubusercontent.com/thanadon-dev/mcontrolpanel/main/install.sh | sudo bash
```

**หรือดาวน์โหลด Binary:**
```bash
# Linux (64-bit)
curl -Lo mcontrolpanel https://github.com/thanadon-dev/mcontrolpanel/releases/latest/download/mcontrolpanel-linux-amd64
chmod +x mcontrolpanel
sudo ./mcontrolpanel --setup

# Linux (ARM64 - Raspberry Pi, etc.)
curl -Lo mcontrolpanel https://github.com/thanadon-dev/mcontrolpanel/releases/latest/download/mcontrolpanel-linux-arm64
chmod +x mcontrolpanel
sudo ./mcontrolpanel --setup
```

**Windows (PowerShell):**
```powershell
Invoke-WebRequest -Uri "https://github.com/thanadon-dev/mcontrolpanel/releases/latest/download/mcontrolpanel-windows-amd64.exe" -OutFile "mcontrolpanel.exe"
.\mcontrolpanel.exe --setup
```

### 🔧 Build จาก Source

#### Linux (Ubuntu/Debian)

```bash
# 1. ติดตั้ง dependencies ระบบ
sudo apt update
sudo apt install -y nginx mysql-server php-fpm git curl tar

# 2. ติดตั้ง Go
wget https://go.dev/dl/go1.21.6.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# 3. เริ่มต้น services
sudo systemctl enable --now nginx mysql

# 4. Clone และ Build
git clone https://github.com/thanadon-dev/mcontrolpanel.git
cd mcontrolpanel
go build -ldflags="-s -w" -o mcontrolpanel .

# 5. รัน setup ครั้งแรก
sudo ./mcontrolpanel --setup

# 6. เริ่มใช้งาน
sudo ./mcontrolpanel
```

#### Linux (CentOS/RHEL/Fedora)

```bash
# 1. ติดตั้ง dependencies
sudo dnf install -y nginx mysql-server php-fpm git curl tar

# 2. ติดตั้ง Go
wget https://go.dev/dl/go1.21.6.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# 3. เริ่มต้น services
sudo systemctl enable --now nginx mysqld php-fpm

# 4. Clone และ Build
git clone https://github.com/thanadon-dev/mcontrolpanel.git
cd mcontrolpanel
go build -ldflags="-s -w" -o mcontrolpanel .

# 5. รัน setup ครั้งแรก
sudo ./mcontrolpanel --setup

# 6. เริ่มใช้งาน
sudo ./mcontrolpanel
```

#### Windows

```powershell
# 1. ติดตั้ง Chocolatey (ถ้ายังไม่มี)
Set-ExecutionPolicy Bypass -Scope Process -Force
[System.Net.ServicePointManager]::SecurityProtocol = [System.Net.ServicePointManager]::SecurityProtocol -bor 3072
iex ((New-Object System.Net.WebClient).DownloadString('https://community.chocolatey.org/install.ps1'))

# 2. ติดตั้ง dependencies
choco install nginx mysql php golang git -y

# 3. รีสตาร์ท PowerShell แล้วรัน:
git clone https://github.com/thanadon-dev/mcontrolpanel.git
cd mcontrolpanel
go build -ldflags="-s -w" -o mcontrolpanel.exe .

# 4. รัน setup ครั้งแรก
.\mcontrolpanel.exe --setup

# 5. เริ่มใช้งาน
.\mcontrolpanel.exe
```

---

## 🔄 ติดตั้งเป็น Service (แนะนำ)

### Linux (systemd)

```bash
# ติดตั้ง service file
sudo cp mcontrolpanel.service /etc/systemd/system/
sudo systemctl daemon-reload

# เปิดใช้งานและเริ่ม
sudo systemctl enable mcontrolpanel
sudo systemctl start mcontrolpanel

# ตรวจสอบสถานะ
sudo systemctl status mcontrolpanel
```

### ลบออก (Uninstall)

```bash
# Linux
curl -fsSL https://raw.githubusercontent.com/thanadon-dev/mcontrolpanel/main/uninstall.sh | sudo bash

# หรือ
sudo ./uninstall.sh
```

---

## 📁 โครงสร้างโปรเจค

```
mcontrolpanel/
├── main.go                      # จุดเริ่มต้นโปรแกรม + GC optimization
├── go.mod                       # Go modules
├── config.example.yaml          # ตัวอย่าง config
├── install.sh                   # Script ติดตั้งอัตโนมัติ
├── uninstall.sh                 # Script ลบออก
├── mcontrolpanel.service        # Systemd service file
├── internal/
│   ├── config/config.go         # จัดการ configuration
│   ├── database/database.go     # SQLite (WAL mode) และ models
│   ├── server/server.go         # HTTP/HTTPS server
│   ├── handlers/handlers.go     # Request handlers + Health checks
│   └── middleware/middleware.go # Auth + Rate Limiting
└── web/
    ├── templates/               # HTML templates
    └── static/                  # CSS/JS assets
```

---

## ⚙️ การตั้งค่า

สร้างไฟล์ `config.yaml` ในโฟลเดอร์เดียวกัน:

```yaml
server:
  host: 0.0.0.0
  port: 8080
  secret_key: เปลี่ยนเป็น-string-สุ่มของคุณ
  enable_https: false
  cert_file: "/etc/ssl/mcontrolpanel/cert.pem"
  key_file: "/etc/ssl/mcontrolpanel/key.pem"

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

rate_limit:
  enabled: true
  requests_per_minute: 60
  login_attempts: 5
```

---

## 🔧 Command Line Options

```
การใช้งาน: mcontrolpanel [options]

Options:
  --config string   พาธไปยังไฟล์ config (ค่าเริ่มต้น "config.yaml")
  --host string     กำหนด host แทนค่าใน config
  --port int        กำหนด port แทนค่าใน config
  --https           เปิดใช้งาน HTTPS
  --cert string     พาธไปยังไฟล์ SSL certificate
  --key string      พาธไปยังไฟล์ SSL private key
  --setup           รัน setup wizard ครั้งแรก
  --version         แสดงเวอร์ชัน
```

---

## 💓 Health Check API

mControlPanel มี endpoints สำหรับ monitoring:

| Endpoint | ใช้สำหรับ | ต้อง Auth |
|----------|----------|-----------|
| `GET /health` | Health check แบบละเอียด | ❌ |
| `GET /ready` | Readiness probe (K8s) | ❌ |
| `GET /live` | Liveness probe (K8s) | ❌ |
| `GET /api/health` | Health check (ต้อง login) | ✅ |

**ตัวอย่าง Response:**
```json
{
  "status": "healthy",
  "timestamp": "2026-02-01T10:30:00Z",
  "version": "1.0.0",
  "checks": {
    "database": "ok",
    "services": true,
    "memory": {"used_percent": 45.2, "ok": true},
    "disk": {"used_percent": 60.1, "ok": true}
  },
  "uptime": "24h30m15s"
}
```

---

## 🖥️ วิธีใช้งาน

1. เปิด browser ไปที่ `http://127.0.0.1:8080`
2. Login ด้วย username/password ที่ตั้งตอน setup
3. เริ่มจัดการ domains, databases และ WordPress ได้เลย!

---

## 🔒 ความปลอดภัย

### Rate Limiting (ป้องกัน Brute Force)
- API ทั่วไป: 60 requests/นาที
- Login: 5 attempts/นาที
- เมื่อเกินจะได้ HTTP 429 Too Many Requests

### HTTPS
```bash
# ใช้กับ Let's Encrypt
sudo ./mcontrolpanel --https --cert /etc/letsencrypt/live/yourdomain/fullchain.pem --key /etc/letsencrypt/live/yourdomain/privkey.pem
```

### หมายเหตุด้านความปลอดภัย
- ⚠️ เปลี่ยน `secret_key` ในไฟล์ config ก่อนใช้งานจริง
- 🔐 ใช้ HTTPS ใน production
- 🛡️ จำกัดการเข้าถึง panel เฉพาะ IP ที่เชื่อถือได้
- 🔑 ใช้รหัสผ่านที่แข็งแรงสำหรับ admin

---

## 🛠️ สำหรับนักพัฒนา

```bash
# รันในโหมด development
go run . --config config.yaml

# Build สำหรับ production (ขนาดเล็กลง)
go build -ldflags="-s -w" -o mcontrolpanel .

# Cross-compile สำหรับ platform อื่น
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o mcontrolpanel-linux-amd64 .
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o mcontrolpanel-windows-amd64.exe .
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o mcontrolpanel-linux-arm64 .
```

---

## 📝 License

MIT License - ใช้ได้ฟรีทั้งส่วนตัวและเชิงพาณิชย์

---

## 🤝 ร่วมพัฒนา

ยินดีรับ contributions! เปิด issue หรือ pull request ได้เลย

---

**mControlPanel** - โฮสติ้งง่ายๆ เบาๆ ไวๆ 🚀
