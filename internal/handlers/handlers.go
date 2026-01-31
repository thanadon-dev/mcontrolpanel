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
