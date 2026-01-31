package handlers

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"

	"mcontrolpanel/internal/config"
	"mcontrolpanel/internal/database"
)

type Handler struct {
	config *config.Config
	db     *database.DB
}

func New(cfg *config.Config, db *database.DB) *Handler {
	return &Handler{config: cfg, db: db}
}

// Auth handlers
func (h *Handler) LoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", gin.H{})
}

func (h *Handler) Login(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	user, err := h.db.AuthenticateUser(username, password)
	if err != nil {
		c.HTML(http.StatusOK, "login.html", gin.H{"error": "Invalid credentials"})
		return
	}

	// Set session cookie
	c.SetCookie("session", fmt.Sprintf("user_%d", user.ID), 86400*7, "/", "", false, true)
	c.Redirect(http.StatusFound, "/dashboard")
}

func (h *Handler) Logout(c *gin.Context) {
	c.SetCookie("session", "", -1, "/", "", false, true)
	c.Redirect(http.StatusFound, "/login")
}

// Dashboard
func (h *Handler) Dashboard(c *gin.Context) {
	user := c.MustGet("user").(*database.User)
	stats := h.db.GetStats()
	services := h.getServicesStatus()
	system := h.getSystemInfo()

	domains, _ := h.db.GetAllDomains()
	wordpress, _ := h.db.GetAllWordPress()

	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"user":      user,
		"stats":     stats,
		"services":  services,
		"system":    system,
		"domains":   domains,
		"wordpress": wordpress,
	})
}

// Domains
func (h *Handler) DomainsPage(c *gin.Context) {
	user := c.MustGet("user").(*database.User)
	domains, _ := h.db.GetAllDomains()

	c.HTML(http.StatusOK, "domains.html", gin.H{
		"user":    user,
		"domains": domains,
	})
}

func (h *Handler) DomainAddPage(c *gin.Context) {
	user := c.MustGet("user").(*database.User)

	c.HTML(http.StatusOK, "domain_add.html", gin.H{
		"user":        user,
		"phpVersions": h.config.PHP.Versions,
	})
}

func (h *Handler) DomainCreate(c *gin.Context) {
	name := c.PostForm("name")
	phpVersion := c.PostForm("php_version")

	if name == "" {
		c.Redirect(http.StatusFound, "/domains/add?error=name_required")
		return
	}

	// Create document root
	docRoot := filepath.Join(h.config.Paths.WWWRoot, name)
	os.MkdirAll(docRoot, 0755)

	// Create index.html
	indexPath := filepath.Join(docRoot, "index.html")
	os.WriteFile(indexPath, []byte(fmt.Sprintf("<html><body><h1>Welcome to %s</h1></body></html>", name)), 0644)

	// Save to database
	domain, err := h.db.CreateDomain(name, docRoot, phpVersion)
	if err != nil {
		c.Redirect(http.StatusFound, "/domains/add?error="+err.Error())
		return
	}

	// Generate Nginx config
	h.generateNginxConfig(domain)

	c.Redirect(http.StatusFound, "/domains?success=created")
}

func (h *Handler) DomainDelete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	domain, err := h.db.GetDomain(id)
	if err != nil {
		c.Redirect(http.StatusFound, "/domains?error=not_found")
		return
	}

	// Remove Nginx config
	configPath := filepath.Join(h.config.Paths.NginxConf, domain.Name+".conf")
	os.Remove(configPath)

	h.db.DeleteDomain(id)
	c.Redirect(http.StatusFound, "/domains?success=deleted")
}

// Databases
func (h *Handler) DatabasesPage(c *gin.Context) {
	user := c.MustGet("user").(*database.User)
	databases, _ := h.db.GetAllDatabases()

	c.HTML(http.StatusOK, "databases.html", gin.H{
		"user":      user,
		"databases": databases,
	})
}

func (h *Handler) DatabaseCreate(c *gin.Context) {
	name := c.PostForm("name")
	username := c.PostForm("username")
	password := c.PostForm("password")

	if username == "" {
		username = name
	}

	// Create MySQL database
	if err := h.createMySQLDatabase(name, username, password); err != nil {
		c.Redirect(http.StatusFound, "/databases?error="+err.Error())
		return
	}

	h.db.CreateDatabase(name, username, nil)
	c.Redirect(http.StatusFound, "/databases?success=created")
}

func (h *Handler) DatabaseDelete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	dbInfo, err := h.db.GetDatabase(id)
	if err == nil {
		h.dropMySQLDatabase(dbInfo.Name, dbInfo.Username)
	}

	h.db.DeleteDatabase(id)
	c.Redirect(http.StatusFound, "/databases?success=deleted")
}

// WordPress
func (h *Handler) WordPressPage(c *gin.Context) {
	user := c.MustGet("user").(*database.User)
	sites, _ := h.db.GetAllWordPress()
	domains, _ := h.db.GetAllDomains()

	// Map domain names
	domainMap := make(map[int64]string)
	for _, d := range domains {
		domainMap[d.ID] = d.Name
	}

	c.HTML(http.StatusOK, "wordpress.html", gin.H{
		"user":      user,
		"sites":     sites,
		"domainMap": domainMap,
	})
}

func (h *Handler) WordPressInstallPage(c *gin.Context) {
	user := c.MustGet("user").(*database.User)
	domains, _ := h.db.GetAllDomains()

	c.HTML(http.StatusOK, "wordpress_install.html", gin.H{
		"user":    user,
		"domains": domains,
	})
}

func (h *Handler) WordPressInstall(c *gin.Context) {
	domainID, _ := strconv.ParseInt(c.PostForm("domain_id"), 10, 64)
	title := c.PostForm("title")
	adminUser := c.PostForm("admin_user")
	adminPass := c.PostForm("admin_pass")
	adminEmail := c.PostForm("admin_email")

	domain, err := h.db.GetDomain(domainID)
	if err != nil {
		c.Redirect(http.StatusFound, "/wordpress/install?error=domain_not_found")
		return
	}

	// Create database for WordPress
	dbName := "wp_" + strings.ReplaceAll(domain.Name, ".", "_")
	dbUser := dbName
	dbPass := generatePassword(16)

	h.createMySQLDatabase(dbName, dbUser, dbPass)

	// Download and extract WordPress
	if err := h.downloadWordPress(domain.DocumentRoot); err != nil {
		c.Redirect(http.StatusFound, "/wordpress/install?error=download_failed")
		return
	}

	// Generate wp-config.php
	h.generateWPConfig(domain.DocumentRoot, dbName, dbUser, dbPass, adminUser, adminPass, adminEmail, title)

	// Save to database
	h.db.CreateWordPress(domain.ID, title, adminUser, dbName, "latest")
	h.db.CreateDatabase(dbName, dbUser, &domain.ID)

	c.Redirect(http.StatusFound, "/wordpress?success=installed")
}

func (h *Handler) WordPressDelete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	h.db.DeleteWordPress(id)
	c.Redirect(http.StatusFound, "/wordpress?success=deleted")
}

// Backups
func (h *Handler) BackupsPage(c *gin.Context) {
	user := c.MustGet("user").(*database.User)
	backups, _ := h.db.GetAllBackups()
	domains, _ := h.db.GetAllDomains()

	c.HTML(http.StatusOK, "backups.html", gin.H{
		"user":    user,
		"backups": backups,
		"domains": domains,
	})
}

func (h *Handler) BackupCreate(c *gin.Context) {
	backupType := c.PostForm("type")
	domainIDStr := c.PostForm("domain_id")

	var domainID *int64
	if domainIDStr != "" {
		id, _ := strconv.ParseInt(domainIDStr, 10, 64)
		domainID = &id
	}

	// Create backup
	name := fmt.Sprintf("backup_%s_%d", backupType, time.Now().Unix())
	filePath := filepath.Join(h.config.Paths.BackupDir, name+".tar.gz")

	os.MkdirAll(h.config.Paths.BackupDir, 0755)

	var size int64 = 0
	if domainID != nil {
		domain, _ := h.db.GetDomain(*domainID)
		if domain != nil {
			h.createBackup(domain.DocumentRoot, filePath)
			if info, err := os.Stat(filePath); err == nil {
				size = info.Size()
			}
		}
	}

	h.db.CreateBackup(name, backupType, domainID, filePath, size)
	c.Redirect(http.StatusFound, "/backups?success=created")
}

func (h *Handler) BackupRestore(c *gin.Context) {
	// Restore logic
	c.Redirect(http.StatusFound, "/backups?success=restored")
}

func (h *Handler) BackupDelete(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)

	backup, err := h.db.GetBackup(id)
	if err == nil {
		os.Remove(backup.FilePath)
	}

	h.db.DeleteBackup(id)
	c.Redirect(http.StatusFound, "/backups?success=deleted")
}

// Services
func (h *Handler) ServicesPage(c *gin.Context) {
	user := c.MustGet("user").(*database.User)
	services := h.getServicesStatus()

	c.HTML(http.StatusOK, "services.html", gin.H{
		"user":     user,
		"services": services,
	})
}

func (h *Handler) ServiceAction(c *gin.Context) {
	name := c.Param("name")
	action := c.Param("action")

	h.controlService(name, action)
	c.Redirect(http.StatusFound, "/services")
}

// Settings
func (h *Handler) SettingsPage(c *gin.Context) {
	user := c.MustGet("user").(*database.User)

	c.HTML(http.StatusOK, "settings.html", gin.H{
		"user":   user,
		"config": h.config,
	})
}

func (h *Handler) SettingsSave(c *gin.Context) {
	// Save settings logic
	c.Redirect(http.StatusFound, "/settings?success=saved")
}

// Profile
func (h *Handler) ProfilePage(c *gin.Context) {
	user := c.MustGet("user").(*database.User)

	c.HTML(http.StatusOK, "profile.html", gin.H{
		"user": user,
	})
}

func (h *Handler) ProfileUpdate(c *gin.Context) {
	c.Redirect(http.StatusFound, "/profile?success=updated")
}

func (h *Handler) PasswordChange(c *gin.Context) {
	c.Redirect(http.StatusFound, "/profile?success=password_changed")
}

// API handlers
func (h *Handler) APIStats(c *gin.Context) {
	stats := h.db.GetStats()
	c.JSON(http.StatusOK, stats)
}

func (h *Handler) APISystem(c *gin.Context) {
	c.JSON(http.StatusOK, h.getSystemInfo())
}

func (h *Handler) APIServices(c *gin.Context) {
	c.JSON(http.StatusOK, h.getServicesStatus())
}

func (h *Handler) APIServiceAction(c *gin.Context) {
	name := c.Param("name")
	action := c.Param("action")

	err := h.controlService(name, action)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

// Helper functions
func (h *Handler) getSystemInfo() gin.H {
	memInfo, _ := mem.VirtualMemory()
	cpuPercent, _ := cpu.Percent(0, false)
	diskInfo, _ := disk.Usage("/")

	cpuUsage := 0.0
	if len(cpuPercent) > 0 {
		cpuUsage = cpuPercent[0]
	}

	return gin.H{
		"os":          runtime.GOOS,
		"arch":        runtime.GOARCH,
		"cpus":        runtime.NumCPU(),
		"cpu_usage":   cpuUsage,
		"mem_total":   memInfo.Total,
		"mem_used":    memInfo.Used,
		"mem_percent": memInfo.UsedPercent,
		"disk_total":  diskInfo.Total,
		"disk_used":   diskInfo.Used,
		"disk_percent": diskInfo.UsedPercent,
	}
}

func (h *Handler) getServicesStatus() map[string]bool {
	services := map[string]bool{
		"nginx": false,
		"mysql": false,
		"php":   false,
	}

	if runtime.GOOS == "windows" {
		for name := range services {
			cmd := exec.Command("sc", "query", name)
			output, _ := cmd.Output()
			services[name] = strings.Contains(string(output), "RUNNING")
		}
	} else {
		for name := range services {
			cmd := exec.Command("systemctl", "is-active", name)
			output, _ := cmd.Output()
			services[name] = strings.TrimSpace(string(output)) == "active"
		}
	}

	return services
}

func (h *Handler) controlService(name, action string) error {
	var cmd *exec.Cmd

	if runtime.GOOS == "windows" {
		switch action {
		case "start":
			cmd = exec.Command("net", "start", name)
		case "stop":
			cmd = exec.Command("net", "stop", name)
		case "restart":
			exec.Command("net", "stop", name).Run()
			cmd = exec.Command("net", "start", name)
		}
	} else {
		cmd = exec.Command("systemctl", action, name)
	}

	return cmd.Run()
}

func (h *Handler) generateNginxConfig(domain *database.Domain) {
	config := fmt.Sprintf(`server {
    listen 80;
    server_name %s;
    root %s;
    index index.php index.html;

    location / {
        try_files $uri $uri/ /index.php?$args;
    }

    location ~ \.php$ {
        fastcgi_pass unix:/run/php/php%s-fpm.sock;
        fastcgi_index index.php;
        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
        include fastcgi_params;
    }
}
`, domain.Name, domain.DocumentRoot, domain.PHPVersion)

	configPath := filepath.Join(h.config.Paths.NginxConf, domain.Name+".conf")
	os.MkdirAll(h.config.Paths.NginxConf, 0755)
	os.WriteFile(configPath, []byte(config), 0644)
}

func (h *Handler) createMySQLDatabase(name, user, password string) error {
	// Using mysql command line
	commands := []string{
		fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`;", name),
		fmt.Sprintf("CREATE USER IF NOT EXISTS '%s'@'localhost' IDENTIFIED BY '%s';", user, password),
		fmt.Sprintf("GRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'localhost';", name, user),
		"FLUSH PRIVILEGES;",
	}

	for _, sql := range commands {
		cmd := exec.Command("mysql", "-u", h.config.Database.MySQLUser, "-e", sql)
		if h.config.Database.MySQLPass != "" {
			cmd = exec.Command("mysql", "-u", h.config.Database.MySQLUser, "-p"+h.config.Database.MySQLPass, "-e", sql)
		}
		cmd.Run()
	}

	return nil
}

func (h *Handler) dropMySQLDatabase(name, user string) error {
	commands := []string{
		fmt.Sprintf("DROP DATABASE IF EXISTS `%s`;", name),
		fmt.Sprintf("DROP USER IF EXISTS '%s'@'localhost';", user),
	}

	for _, sql := range commands {
		cmd := exec.Command("mysql", "-u", h.config.Database.MySQLUser, "-e", sql)
		if h.config.Database.MySQLPass != "" {
			cmd = exec.Command("mysql", "-u", h.config.Database.MySQLUser, "-p"+h.config.Database.MySQLPass, "-e", sql)
		}
		cmd.Run()
	}

	return nil
}

func (h *Handler) downloadWordPress(destPath string) error {
	wpURL := "https://wordpress.org/latest.tar.gz"
	tarPath := filepath.Join(os.TempDir(), "wordpress.tar.gz")

	// Download
	cmd := exec.Command("curl", "-sL", "-o", tarPath, wpURL)
	if err := cmd.Run(); err != nil {
		// Try wget
		cmd = exec.Command("wget", "-q", "-O", tarPath, wpURL)
		if err := cmd.Run(); err != nil {
			return err
		}
	}

	// Extract
	os.MkdirAll(destPath, 0755)
	cmd = exec.Command("tar", "-xzf", tarPath, "-C", destPath, "--strip-components=1")
	return cmd.Run()
}

func (h *Handler) generateWPConfig(docRoot, dbName, dbUser, dbPass, adminUser, adminPass, adminEmail, title string) {
	config := fmt.Sprintf(`<?php
define('DB_NAME', '%s');
define('DB_USER', '%s');
define('DB_PASSWORD', '%s');
define('DB_HOST', 'localhost');
define('DB_CHARSET', 'utf8mb4');
define('DB_COLLATE', '');

define('AUTH_KEY',         '%s');
define('SECURE_AUTH_KEY',  '%s');
define('LOGGED_IN_KEY',    '%s');
define('NONCE_KEY',        '%s');
define('AUTH_SALT',        '%s');
define('SECURE_AUTH_SALT', '%s');
define('LOGGED_IN_SALT',   '%s');
define('NONCE_SALT',       '%s');

$table_prefix = 'wp_';
define('WP_DEBUG', false);

if ( ! defined( 'ABSPATH' ) ) {
    define( 'ABSPATH', __DIR__ . '/' );
}

require_once ABSPATH . 'wp-settings.php';
`, dbName, dbUser, dbPass,
		generatePassword(64), generatePassword(64), generatePassword(64), generatePassword(64),
		generatePassword(64), generatePassword(64), generatePassword(64), generatePassword(64))

	os.WriteFile(filepath.Join(docRoot, "wp-config.php"), []byte(config), 0644)
}

func (h *Handler) createBackup(sourcePath, destPath string) error {
	cmd := exec.Command("tar", "-czf", destPath, "-C", filepath.Dir(sourcePath), filepath.Base(sourcePath))
	return cmd.Run()
}

func generatePassword(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		time.Sleep(time.Nanosecond)
	}
	return string(b)
}

// Health Check API - สำหรับ monitoring
func (h *Handler) HealthCheck(c *gin.Context) {
	// Check database
	dbStatus := "ok"
	if err := h.db.Ping(); err != nil {
		dbStatus = "error"
	}

	// Check services
	services := h.getServicesStatus()
	servicesOK := true
	for _, svc := range services {
		if !svc["running"].(bool) {
			servicesOK = false
			break
		}
	}

	// Get system info
	memInfo, _ := mem.VirtualMemory()
	diskInfo, _ := disk.Usage("/")

	status := "healthy"
	httpCode := http.StatusOK

	if dbStatus != "ok" || !servicesOK {
		status = "degraded"
		httpCode = http.StatusServiceUnavailable
	}

	// Critical thresholds
	if memInfo != nil && memInfo.UsedPercent > 95 {
		status = "critical"
		httpCode = http.StatusServiceUnavailable
	}
	if diskInfo != nil && diskInfo.UsedPercent > 95 {
		status = "critical"
		httpCode = http.StatusServiceUnavailable
	}

	c.JSON(httpCode, gin.H{
		"status":    status,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"version":   "1.0.0",
		"checks": gin.H{
			"database": dbStatus,
			"services": servicesOK,
			"memory": gin.H{
				"used_percent": memInfo.UsedPercent,
				"ok":           memInfo.UsedPercent < 90,
			},
			"disk": gin.H{
				"used_percent": diskInfo.UsedPercent,
				"ok":           diskInfo.UsedPercent < 90,
			},
		},
		"uptime": time.Since(startTime).String(),
	})
}

// Readiness probe - ตรวจสอบว่าพร้อมรับ traffic
func (h *Handler) ReadinessCheck(c *gin.Context) {
	if err := h.db.Ping(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not_ready",
			"error":  "database not available",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
	})
}

// Liveness probe - ตรวจสอบว่ายังทำงานอยู่
func (h *Handler) LivenessCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "alive",
	})
}

var startTime = time.Now()

// ==================== Resource Monitoring ====================

// ResourceHistory - เก็บประวัติ resource usage
type ResourceHistory struct {
	Timestamp   time.Time `json:"timestamp"`
	CPUPercent  float64   `json:"cpu_percent"`
	MemPercent  float64   `json:"mem_percent"`
	DiskPercent float64   `json:"disk_percent"`
	NetIn       uint64    `json:"net_in"`
	NetOut      uint64    `json:"net_out"`
}

var (
	resourceHistory     []ResourceHistory
	resourceHistoryLock sync.Mutex
	monitoringStarted   bool
)

// StartResourceMonitoring - เริ่มเก็บข้อมูล resource
func StartResourceMonitoring() {
	if monitoringStarted {
		return
	}
	monitoringStarted = true

	go func() {
		ticker := time.NewTicker(10 * time.Second) // เก็บทุก 10 วินาที
		for range ticker.C {
			collectResourceData()
		}
	}()
}

func collectResourceData() {
	cpuPercent, _ := cpu.Percent(time.Second, false)
	memInfo, _ := mem.VirtualMemory()
	diskInfo, _ := disk.Usage("/")

	cpuUsage := 0.0
	if len(cpuPercent) > 0 {
		cpuUsage = cpuPercent[0]
	}

	record := ResourceHistory{
		Timestamp:   time.Now(),
		CPUPercent:  cpuUsage,
		MemPercent:  memInfo.UsedPercent,
		DiskPercent: diskInfo.UsedPercent,
	}

	resourceHistoryLock.Lock()
	resourceHistory = append(resourceHistory, record)
	// เก็บแค่ 1 ชั่วโมง (360 records @ 10s interval)
	if len(resourceHistory) > 360 {
		resourceHistory = resourceHistory[1:]
	}
	resourceHistoryLock.Unlock()
}

// MonitoringPage - หน้า Resource Monitoring
func (h *Handler) MonitoringPage(c *gin.Context) {
	user := c.MustGet("user").(*database.User)

	c.HTML(http.StatusOK, "monitoring.html", gin.H{
		"user": user,
	})
}

// APIResourceHistory - API สำหรับดึงประวัติ resource
func (h *Handler) APIResourceHistory(c *gin.Context) {
	period := c.DefaultQuery("period", "1h") // 5m, 15m, 30m, 1h

	resourceHistoryLock.Lock()
	defer resourceHistoryLock.Unlock()

	var filtered []ResourceHistory
	var cutoff time.Time

	switch period {
	case "5m":
		cutoff = time.Now().Add(-5 * time.Minute)
	case "15m":
		cutoff = time.Now().Add(-15 * time.Minute)
	case "30m":
		cutoff = time.Now().Add(-30 * time.Minute)
	default:
		cutoff = time.Now().Add(-1 * time.Hour)
	}

	for _, r := range resourceHistory {
		if r.Timestamp.After(cutoff) {
			filtered = append(filtered, r)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"period":  period,
		"data":    filtered,
		"current": h.getSystemInfo(),
	})
}

// APIResourceRealtime - API สำหรับ realtime data
func (h *Handler) APIResourceRealtime(c *gin.Context) {
	cpuPercent, _ := cpu.Percent(time.Second, false)
	memInfo, _ := mem.VirtualMemory()
	diskInfo, _ := disk.Usage("/")

	cpuUsage := 0.0
	if len(cpuPercent) > 0 {
		cpuUsage = cpuPercent[0]
	}

	c.JSON(http.StatusOK, gin.H{
		"timestamp":    time.Now().UTC().Format(time.RFC3339),
		"cpu_percent":  cpuUsage,
		"cpu_cores":    runtime.NumCPU(),
		"mem_total":    memInfo.Total,
		"mem_used":     memInfo.Used,
		"mem_free":     memInfo.Free,
		"mem_percent":  memInfo.UsedPercent,
		"disk_total":   diskInfo.Total,
		"disk_used":    diskInfo.Used,
		"disk_free":    diskInfo.Free,
		"disk_percent": diskInfo.UsedPercent,
		"uptime":       time.Since(startTime).String(),
	})
}

// ==================== Auto SSL (Let's Encrypt) ====================

// SSLPage - หน้าจัดการ SSL
func (h *Handler) SSLPage(c *gin.Context) {
	user := c.MustGet("user").(*database.User)
	domains, _ := h.db.GetAllDomains()

	// ดึงสถานะ SSL ของแต่ละ domain
	sslStatus := make(map[string]gin.H)
	for _, d := range domains {
		certPath := filepath.Join(h.config.Paths.SSLDir, d.Name, "fullchain.pem")
		keyPath := filepath.Join(h.config.Paths.SSLDir, d.Name, "privkey.pem")

		status := gin.H{
			"enabled":    false,
			"expires":    "",
			"issuer":     "",
			"auto_renew": false,
		}

		if _, err := os.Stat(certPath); err == nil {
			status["enabled"] = true
			// ดึงข้อมูล certificate
			if certInfo, err := getCertificateInfo(certPath); err == nil {
				status["expires"] = certInfo["expires"]
				status["issuer"] = certInfo["issuer"]
				status["days_left"] = certInfo["days_left"]
			}
		}

		sslStatus[d.Name] = status
		_ = keyPath // suppress unused warning
	}

	c.HTML(http.StatusOK, "ssl.html", gin.H{
		"user":      user,
		"domains":   domains,
		"sslStatus": sslStatus,
	})
}

// SSLIssue - ขอ SSL Certificate จาก Let's Encrypt
func (h *Handler) SSLIssue(c *gin.Context) {
	domain := c.PostForm("domain")
	email := c.PostForm("email")

	if domain == "" || email == "" {
		c.Redirect(http.StatusFound, "/ssl?error=domain_and_email_required")
		return
	}

	// สร้างโฟลเดอร์ SSL
	sslDir := filepath.Join(h.config.Paths.SSLDir, domain)
	os.MkdirAll(sslDir, 0755)

	// ใช้ certbot ขอ certificate
	err := h.issueLetsEncrypt(domain, email)
	if err != nil {
		c.Redirect(http.StatusFound, "/ssl?error="+err.Error())
		return
	}

	// อัพเดท Nginx config เพื่อใช้ SSL
	h.updateNginxSSL(domain)

	// อัพเดท database
	h.db.UpdateDomainSSL(domain, true)

	c.Redirect(http.StatusFound, "/ssl?success=ssl_issued")
}

// SSLRenew - ต่ออายุ SSL Certificate
func (h *Handler) SSLRenew(c *gin.Context) {
	domain := c.Param("domain")

	err := h.renewLetsEncrypt(domain)
	if err != nil {
		c.Redirect(http.StatusFound, "/ssl?error="+err.Error())
		return
	}

	c.Redirect(http.StatusFound, "/ssl?success=ssl_renewed")
}

// SSLRevoke - ยกเลิก SSL Certificate
func (h *Handler) SSLRevoke(c *gin.Context) {
	domain := c.Param("domain")

	// ลบ certificate files
	sslDir := filepath.Join(h.config.Paths.SSLDir, domain)
	os.RemoveAll(sslDir)

	// อัพเดท Nginx config กลับเป็น HTTP
	h.updateNginxNoSSL(domain)

	// อัพเดท database
	h.db.UpdateDomainSSL(domain, false)

	c.Redirect(http.StatusFound, "/ssl?success=ssl_revoked")
}

// issueLetsEncrypt - ขอ certificate จาก Let's Encrypt
func (h *Handler) issueLetsEncrypt(domain, email string) error {
	webroot := filepath.Join(h.config.Paths.WWWRoot, domain)
	sslDir := filepath.Join(h.config.Paths.SSLDir, domain)

	// ใช้ certbot webroot mode
	cmd := exec.Command("certbot", "certonly",
		"--webroot",
		"--webroot-path", webroot,
		"--email", email,
		"--agree-tos",
		"--no-eff-email",
		"--cert-path", filepath.Join(sslDir, "cert.pem"),
		"--key-path", filepath.Join(sslDir, "privkey.pem"),
		"--fullchain-path", filepath.Join(sslDir, "fullchain.pem"),
		"-d", domain,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		// ลอง standalone mode ถ้า webroot ไม่ได้
		cmd2 := exec.Command("certbot", "certonly",
			"--standalone",
			"--email", email,
			"--agree-tos",
			"--no-eff-email",
			"--cert-path", filepath.Join(sslDir, "cert.pem"),
			"--key-path", filepath.Join(sslDir, "privkey.pem"),
			"--fullchain-path", filepath.Join(sslDir, "fullchain.pem"),
			"-d", domain,
		)
		output2, err2 := cmd2.CombinedOutput()
		if err2 != nil {
			return fmt.Errorf("certbot failed: %s, %s", string(output), string(output2))
		}
	}
	_ = output

	// Copy certificates to our SSL directory
	letsencryptDir := fmt.Sprintf("/etc/letsencrypt/live/%s", domain)
	if _, err := os.Stat(letsencryptDir); err == nil {
		exec.Command("cp", filepath.Join(letsencryptDir, "fullchain.pem"), filepath.Join(sslDir, "fullchain.pem")).Run()
		exec.Command("cp", filepath.Join(letsencryptDir, "privkey.pem"), filepath.Join(sslDir, "privkey.pem")).Run()
	}

	return nil
}

// renewLetsEncrypt - ต่ออายุ certificate
func (h *Handler) renewLetsEncrypt(domain string) error {
	cmd := exec.Command("certbot", "renew", "--cert-name", domain, "--force-renewal")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("renewal failed: %s", string(output))
	}

	// Reload nginx
	exec.Command("systemctl", "reload", "nginx").Run()

	return nil
}

// updateNginxSSL - อัพเดท Nginx config สำหรับ SSL
func (h *Handler) updateNginxSSL(domain string) {
	sslDir := filepath.Join(h.config.Paths.SSLDir, domain)
	configPath := filepath.Join(h.config.Paths.NginxConf, domain+".conf")

	config := fmt.Sprintf(`server {
    listen 80;
    server_name %s;
    return 301 https://$server_name$request_uri;
}

server {
    listen 443 ssl http2;
    server_name %s;
    root %s/%s;
    index index.php index.html;

    ssl_certificate %s/fullchain.pem;
    ssl_certificate_key %s/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-ECDSA-AES128-GCM-SHA256:ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES256-GCM-SHA384:ECDHE-RSA-AES256-GCM-SHA384;
    ssl_prefer_server_ciphers off;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 1d;

    # HSTS
    add_header Strict-Transport-Security "max-age=63072000" always;

    location / {
        try_files $uri $uri/ /index.php?$query_string;
    }

    location ~ \.php$ {
        fastcgi_pass unix:/var/run/php/php-fpm.sock;
        fastcgi_index index.php;
        fastcgi_param SCRIPT_FILENAME $realpath_root$fastcgi_script_name;
        include fastcgi_params;
    }

    location ~ /\.ht {
        deny all;
    }

    # Let's Encrypt challenge
    location ^~ /.well-known/acme-challenge/ {
        root %s/%s;
    }
}
`, domain, domain, h.config.Paths.WWWRoot, domain, sslDir, sslDir, h.config.Paths.WWWRoot, domain)

	os.WriteFile(configPath, []byte(config), 0644)
	exec.Command("systemctl", "reload", "nginx").Run()
}

// updateNginxNoSSL - อัพเดท Nginx config กลับเป็น HTTP only
func (h *Handler) updateNginxNoSSL(domain string) {
	configPath := filepath.Join(h.config.Paths.NginxConf, domain+".conf")

	config := fmt.Sprintf(`server {
    listen 80;
    server_name %s;
    root %s/%s;
    index index.php index.html;

    location / {
        try_files $uri $uri/ /index.php?$query_string;
    }

    location ~ \.php$ {
        fastcgi_pass unix:/var/run/php/php-fpm.sock;
        fastcgi_index index.php;
        fastcgi_param SCRIPT_FILENAME $realpath_root$fastcgi_script_name;
        include fastcgi_params;
    }

    location ~ /\.ht {
        deny all;
    }
}
`, domain, h.config.Paths.WWWRoot, domain)

	os.WriteFile(configPath, []byte(config), 0644)
	exec.Command("systemctl", "reload", "nginx").Run()
}

// getCertificateInfo - ดึงข้อมูล SSL certificate
func getCertificateInfo(certPath string) (map[string]interface{}, error) {
	cmd := exec.Command("openssl", "x509", "-in", certPath, "-noout", "-enddate", "-issuer")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	info := make(map[string]interface{})
	lines := strings.Split(string(output), "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "notAfter=") {
			dateStr := strings.TrimPrefix(line, "notAfter=")
			if t, err := time.Parse("Jan 2 15:04:05 2006 MST", dateStr); err == nil {
				info["expires"] = t.Format("2006-01-02")
				info["days_left"] = int(time.Until(t).Hours() / 24)
			}
		}
		if strings.HasPrefix(line, "issuer=") {
			info["issuer"] = strings.TrimPrefix(line, "issuer=")
		}
	}

	return info, nil
}

// StartAutoRenew - เริ่ม auto renew certificates
func StartAutoRenew() {
	go func() {
		// ตรวจสอบทุกวัน
		ticker := time.NewTicker(24 * time.Hour)
		for range ticker.C {
			// รัน certbot renew
			exec.Command("certbot", "renew", "--quiet").Run()
			exec.Command("systemctl", "reload", "nginx").Run()
		}
	}()
}
