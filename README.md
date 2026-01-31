# mControlPanel

แผงควบคุมเว็บโฮสติ้งที่เบาและเร็ว เขียนด้วยภาษา Go - ติดตั้งง่าย ใช้ทรัพยากรน้อย

## ✨ คุณสมบัติ

- **ไฟล์เดียวจบ** - ไม่ต้องติดตั้ง runtime เพิ่มเติม แค่ไฟล์ exe/binary ไฟล์เดียว
- **ประหยัด RAM** - ใช้แค่ ~15MB (เทียบกับ Python/Node ที่ใช้ 200MB+)
- **เร็วมาก** - เริ่มต้นทันที ตอบสนองภายในมิลลิวินาที
- **ข้ามแพลตฟอร์ม** - ใช้ได้ทั้ง Linux และ Windows
- **โดเมนไม่จำกัด** - สร้างและจัดการ virtual hosts ได้เท่าที่ต้องการ
- **จัดการ MySQL** - สร้าง/ลบ/จัดการฐานข้อมูลได้ครบ
- **ติดตั้ง WordPress** - คลิกเดียวติดตั้ง WordPress ได้เลย
- **รองรับ PHP หลายเวอร์ชัน** - PHP 7.4, 8.0, 8.1, 8.2, 8.3
- **ระบบ Backup** - สำรองข้อมูลทั้งไฟล์และฐานข้อมูล
- **ควบคุม Services** - Start/Stop/Restart Nginx, MySQL, PHP-FPM
- **UI ทันสมัย** - หน้าตาสวย ใช้งานง่าย โทนสีเข้ม

## 📊 เปรียบเทียบทรัพยากร

| Panel | ภาษา | ขนาด Binary | RAM | เวลาเริ่มต้น |
|-------|------|-------------|-----|-------------|
| **mControlPanel** | Go | ~15MB | ~15MB | <1 วินาที |
| cPanel | Perl/C | N/A | 500MB+ | 30+ วินาที |
| Plesk | PHP | N/A | 400MB+ | 20+ วินาที |
| Python Panel อื่นๆ | Python | 50MB+ deps | 200MB+ | 5+ วินาที |

---

## 🚀 วิธีติดตั้ง

### ⚡ ติดตั้งด่วน (ดาวน์โหลด Binary สำเร็จรูป)

> **หมายเหตุ:** วิธีนี้ต้องรอสร้าง Release ก่อน หรือใช้วิธี Build จาก Source ด้านล่าง

### 🔧 Build จาก Source (แนะนำ)

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

## 📁 โครงสร้างโปรเจค

```
mcontrolpanel/
├── main.go                      # จุดเริ่มต้นโปรแกรม
├── go.mod                       # Go modules
├── config.example.yaml          # ตัวอย่าง config
├── internal/
│   ├── config/config.go         # จัดการ configuration
│   ├── database/database.go     # SQLite database และ models
│   ├── server/server.go         # HTTP server setup
│   ├── handlers/handlers.go     # จัดการ requests ทั้งหมด
│   └── middleware/middleware.go # Auth middleware
└── web/
    ├── templates/               # HTML templates
    └── static/                  # CSS/JS assets
```

---

## ⚙️ การตั้งค่า

สร้างไฟล์ `config.yaml` ในโฟลเดอร์เดียวกัน:

```yaml
server:
  host: 127.0.0.1
  port: 8080
  secret_key: เปลี่ยนเป็น-string-สุ่มของคุณ

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

---

## 🔧 Command Line Options

```
การใช้งาน: mcontrolpanel [options]

Options:
  --config string   พาธไปยังไฟล์ config (ค่าเริ่มต้น "config.yaml")
  --host string     กำหนด host แทนค่าใน config
  --port int        กำหนด port แทนค่าใน config
  --setup           รัน setup wizard ครั้งแรก
  --version         แสดงเวอร์ชัน
```

---

## 🖥️ วิธีใช้งาน

1. เปิด browser ไปที่ `http://127.0.0.1:8080`
2. Login ด้วย username/password ที่ตั้งตอน setup
3. เริ่มจัดการ domains, databases และ WordPress ได้เลย!

---

## 🔒 หมายเหตุด้านความปลอดภัย

- ⚠️ เปลี่ยน `secret_key` ในไฟล์ config ก่อนใช้งานจริง
- 🔐 ควรรันผ่าน reverse proxy (Nginx) พร้อม HTTPS
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
