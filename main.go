package main

import (
	"embed"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"

	"mcontrolpanel/internal/config"
	"mcontrolpanel/internal/database"
	"mcontrolpanel/internal/server"
)

//go:embed web/templates/* web/static/*
var embedFS embed.FS

var (
	version   = "1.0.0"
	buildTime = "unknown"
)

func init() {
	// ตั้งค่า GC ให้ประหยัด memory
	debug.SetGCPercent(50) // GC บ่อยขึ้น ประหยัด RAM
	debug.SetMemoryLimit(100 * 1024 * 1024) // จำกัด 100MB
}

func main() {
	// Command line flags
	configPath := flag.String("config", "config.yaml", "Path to configuration file")
	host := flag.String("host", "", "Override host address")
	port := flag.Int("port", 0, "Override port number")
	showVersion := flag.Bool("version", false, "Show version information")
	setup := flag.Bool("setup", false, "Run initial setup")
	enableHTTPS := flag.Bool("https", false, "Enable HTTPS")
	certFile := flag.String("cert", "", "Path to SSL certificate file")
	keyFile := flag.String("key", "", "Path to SSL key file")
	flag.Parse()

	// Show version
	if *showVersion {
		fmt.Printf("mControlPanel v%s (built: %s)\n", version, buildTime)
		os.Exit(0)
	}

	// Print banner
	printBanner()

	// Load configuration
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Printf("Warning: Could not load config file, using defaults: %v", err)
		cfg = config.Default()
	}

	// Override from command line
	if *host != "" {
		cfg.Server.Host = *host
	}
	if *port != 0 {
		cfg.Server.Port = *port
	}
	if *enableHTTPS {
		cfg.Server.EnableHTTPS = true
	}
	if *certFile != "" {
		cfg.Server.CertFile = *certFile
	}
	if *keyFile != "" {
		cfg.Server.KeyFile = *keyFile
	}

	// Initialize database
	db, err := database.New(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// Run setup if requested or first time
	if *setup || !db.HasAdmin() {
		if err := runSetup(db); err != nil {
			log.Fatalf("Setup failed: %v", err)
		}
	}

	// Create and start server
	srv := server.New(cfg, db, embedFS)

	// Handle graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("\nShutting down server...")
		srv.Shutdown()
	}()

	// Start server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	
	if cfg.Server.EnableHTTPS && cfg.Server.CertFile != "" && cfg.Server.KeyFile != "" {
		log.Printf("Starting mControlPanel on https://%s", addr)
		if err := srv.RunTLS(addr, cfg.Server.CertFile, cfg.Server.KeyFile); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	} else {
		log.Printf("Starting mControlPanel on http://%s", addr)
		if err := srv.Run(addr); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}
}

func printBanner() {
	fmt.Println(`
  ┌──────────────────────────────────────────┐
  │         mControlPanel v` + version + `            │
  │     Lightweight Web Hosting Panel        │
  └──────────────────────────────────────────┘`)
}

func runSetup(db *database.DB) error {
	fmt.Println("\n=== Initial Setup ===")

	var username, password, email string

	fmt.Print("Admin username [admin]: ")
	fmt.Scanln(&username)
	if username == "" {
		username = "admin"
	}

	fmt.Print("Admin email: ")
	fmt.Scanln(&email)

	fmt.Print("Admin password: ")
	fmt.Scanln(&password)
	if password == "" {
		password = "admin123"
		fmt.Println("Using default password: admin123")
	}

	return db.CreateUser(username, password, email, "admin")
}
