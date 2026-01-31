package server

import (
	"context"
	"crypto/tls"
	"embed"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"mcontrolpanel/internal/config"
	"mcontrolpanel/internal/database"
	"mcontrolpanel/internal/handlers"
	"mcontrolpanel/internal/middleware"
)

type Server struct {
	config   *config.Config
	db       *database.DB
	router   *gin.Engine
	embedFS  embed.FS
	httpSrv  *http.Server
}

func New(cfg *config.Config, db *database.DB, embedFS embed.FS) *Server {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.Logger())
	router.Use(middleware.RateLimit()) // เพิ่ม rate limiting

	s := &Server{
		config:  cfg,
		db:      db,
		router:  router,
		embedFS: embedFS,
	}

	s.setupTemplates()
	s.setupRoutes()

	return s
}

func (s *Server) setupTemplates() {
	// Load templates from embedded filesystem
	tmpl := template.New("")

	// Walk through embedded templates
	fs.WalkDir(s.embedFS, "web/templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		
		data, _ := s.embedFS.ReadFile(path)
		tmpl.New(path[14:]).Parse(string(data)) // Remove "web/templates/" prefix
		return nil
	})

	s.router.SetHTMLTemplate(tmpl)

	// Serve static files
	staticFS, _ := fs.Sub(s.embedFS, "web/static")
	s.router.StaticFS("/static", http.FS(staticFS))
}

func (s *Server) setupRoutes() {
	h := handlers.New(s.config, s.db)

	// Health check routes (ไม่ต้อง auth)
	s.router.GET("/health", h.HealthCheck)
	s.router.GET("/healthz", h.HealthCheck)
	s.router.GET("/ready", h.ReadinessCheck)
	s.router.GET("/live", h.LivenessCheck)

	// Public routes
	s.router.GET("/login", h.LoginPage)
	s.router.POST("/login", middleware.LoginRateLimit(), h.Login)
	s.router.GET("/logout", h.Logout)

	// Protected routes
	auth := s.router.Group("/")
	auth.Use(middleware.Auth(s.db))
	{
		auth.GET("/", h.Dashboard)
		auth.GET("/dashboard", h.Dashboard)

		// Domains
		auth.GET("/domains", h.DomainsPage)
		auth.GET("/domains/add", h.DomainAddPage)
		auth.POST("/domains", h.DomainCreate)
		auth.POST("/domains/:id/delete", h.DomainDelete)

		// Databases
		auth.GET("/databases", h.DatabasesPage)
		auth.POST("/databases", h.DatabaseCreate)
		auth.POST("/databases/:id/delete", h.DatabaseDelete)

		// WordPress
		auth.GET("/wordpress", h.WordPressPage)
		auth.GET("/wordpress/install", h.WordPressInstallPage)
		auth.POST("/wordpress/install", h.WordPressInstall)
		auth.POST("/wordpress/:id/delete", h.WordPressDelete)

		// Backups
		auth.GET("/backups", h.BackupsPage)
		auth.POST("/backups", h.BackupCreate)
		auth.POST("/backups/:id/restore", h.BackupRestore)
		auth.POST("/backups/:id/delete", h.BackupDelete)

		// Services
		auth.GET("/services", h.ServicesPage)
		auth.POST("/services/:name/:action", h.ServiceAction)

		// Settings
		auth.GET("/settings", h.SettingsPage)
		auth.POST("/settings", h.SettingsSave)

		// Profile
		auth.GET("/profile", h.ProfilePage)
		auth.POST("/profile", h.ProfileUpdate)
		auth.POST("/profile/password", h.PasswordChange)
	}

	// API routes
	api := s.router.Group("/api")
	api.Use(middleware.Auth(s.db))
	{
		api.GET("/stats", h.APIStats)
		api.GET("/system", h.APISystem)
		api.GET("/services", h.APIServices)
		api.POST("/services/:name/:action", h.APIServiceAction)
		api.GET("/health", h.HealthCheck) // Health check ที่ต้อง auth
	}
}

func (s *Server) Run(addr string) error {
	s.httpSrv = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s.httpSrv.ListenAndServe()
}

// RunTLS - รัน HTTPS server
func (s *Server) RunTLS(addr, certFile, keyFile string) error {
	tlsConfig := &tls.Config{
		MinVersion:               tls.VersionTLS12,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		},
	}

	s.httpSrv = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		TLSConfig:    tlsConfig,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("Starting HTTPS server on %s", addr)
	return s.httpSrv.ListenAndServeTLS(certFile, keyFile)
}

func (s *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.httpSrv.Shutdown(ctx)
}
