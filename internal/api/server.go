package api

import (
	"html/template"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	xraypanel "xray-panel"
	"xray-panel/internal/config"
	"xray-panel/internal/nginx"
	"xray-panel/internal/web"
)

// Server represents the API server
type Server struct {
	config     *config.Config
	db         *gorm.DB
	router     *gin.Engine
	embedFiles fs.FS
	templates  *template.Template
	webHandler *web.Handler
	nginxGen   *nginx.ConfigGenerator
}

// NewServer creates a new API server
func NewServer(cfg *config.Config, db *gorm.DB) *Server {
	if !cfg.Server.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	// Use the embedded files from the root package
	webFS, _ := fs.Sub(xraypanel.WebFiles, "web")

	// Load templates
	templates, err := web.LoadTemplates(webFS)
	if err != nil {
		panic("Failed to load templates: " + err.Error())
	}

	// Create Nginx config generator
	nginxGen := nginx.NewGenerator(cfg.Nginx.ConfigDir, cfg.Nginx.StreamDir)
	nginxGen.SetDB(db)
	if cfg.Nginx.ReloadCmd != "" {
		nginxGen.SetReloadCmd(cfg.Nginx.ReloadCmd)
	}

	s := &Server{
		config:     cfg,
		db:         db,
		router:     gin.Default(),
		embedFiles: webFS,
		templates:  templates,
		nginxGen:   nginxGen,
	}

	// Set HTML templates
	s.router.SetHTMLTemplate(templates)

	// Create web handler with DB and Nginx generator
	s.webHandler = web.NewHandler(db, nginxGen)

	s.setupRoutes()
	return s
}

// Run starts the server
func (s *Server) Run() error {
	return s.router.Run(s.config.Server.Listen)
}

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Serve static files from embed
	s.router.StaticFS("/static", http.FS(mustSub(s.embedFiles, "static")))

	// Login/Logout routes (public)
	s.router.GET("/login", s.webHandler.LoginPage)
	s.router.POST("/login", s.handleWebLogin)
	s.router.GET("/logout", s.handleWebLogout)
	s.router.POST("/logout", s.handleWebLogout)

	// Page routes (require auth)
	pages := s.router.Group("/")
	pages.Use(s.webAuthMiddleware())
	{
		pages.GET("/", s.webHandler.DashboardPage)
		pages.GET("/dashboard", s.webHandler.DashboardPage)
		pages.GET("/users", s.webHandler.UsersPage)
		pages.GET("/inbounds", s.webHandler.InboundsPage)
		pages.GET("/outbounds", s.webHandler.OutboundsPage)
		pages.GET("/routing", s.webHandler.RoutingPage)
		pages.GET("/domains", s.webHandler.DomainsPage)
		pages.GET("/settings", s.webHandler.SettingsPage)
	}

	// Form routes (return HTML forms)
	forms := s.router.Group("/")
	forms.Use(s.webAuthMiddleware())
	{
		// User forms
		forms.GET("/users/new", s.webHandler.NewUserForm)
		forms.GET("/users/:id/edit", s.webHandler.EditUserForm)

		// Inbound forms
		forms.GET("/inbounds/new", s.webHandler.NewInboundForm)
		forms.GET("/inbounds/:id/edit", s.webHandler.EditInboundForm)

		// Outbound forms
		forms.GET("/outbounds/new", s.webHandler.NewOutboundForm)
		forms.GET("/outbounds/:id/edit", s.webHandler.EditOutboundForm)

		// Routing forms
		forms.GET("/routing/new", s.webHandler.NewRoutingForm)
		forms.GET("/routing/:id/edit", s.webHandler.EditRoutingForm)

		// Domain forms
		forms.GET("/domains/new", s.webHandler.NewDomainForm)
		forms.GET("/domains/:id/edit", s.webHandler.EditDomainForm)
	}

	// API routes - public (no auth required)
	apiPublic := s.router.Group("/api")
	{
		// Health check
		apiPublic.GET("/health", s.handleHealth)

		// Xray status (public for dashboard display)
		apiPublic.GET("/xray/status", s.handleXrayStatus)
	}

	// API routes - protected (require session auth)
	api := s.router.Group("/api")
	api.Use(s.webAuthMiddleware())
	{
		// Dashboard
		api.GET("/dashboard/stats", s.webHandler.DashboardStats)

		// Users
		api.GET("/users/table", s.webHandler.UsersTable)
		api.GET("/users/search", s.webHandler.SearchUsers)
		api.POST("/users", s.webHandler.CreateUser)
		api.POST("/users/:id", s.webHandler.UpdateUser)
		api.DELETE("/users/:id", s.webHandler.DeleteUser)

		// Inbounds
		api.GET("/inbounds/table", s.webHandler.InboundsTable)
		api.POST("/inbounds", s.webHandler.CreateInbound)
		api.POST("/inbounds/:id", s.webHandler.UpdateInbound)
		api.DELETE("/inbounds/:id", s.webHandler.DeleteInbound)

		// Outbounds
		api.GET("/outbounds/table", s.webHandler.OutboundsTable)
		api.POST("/outbounds", s.webHandler.CreateOutbound)
		api.POST("/outbounds/:id", s.webHandler.UpdateOutbound)
		api.DELETE("/outbounds/:id", s.webHandler.DeleteOutbound)

		// Routing
		api.GET("/routing/table", s.webHandler.RoutingTable)
		api.POST("/routing", s.webHandler.CreateRouting)
		api.POST("/routing/:id", s.webHandler.UpdateRouting)
		api.DELETE("/routing/:id", s.webHandler.DeleteRouting)
		api.POST("/routing/preset/:preset", s.handleImportPresetRules)

		// Domains
		api.GET("/domains/table", s.webHandler.DomainsTable)
		api.POST("/domains/scan-import", s.handleScanAndImportCertificates)
		api.POST("/domains", s.webHandler.CreateDomain)
		api.POST("/domains/:id", s.webHandler.UpdateDomain)
		api.DELETE("/domains/:id", s.webHandler.DeleteDomain)

		// Xray control (protected)
		api.POST("/xray/restart", s.handleXrayRestart)
		api.GET("/xray/config", s.handleGetXrayConfig)
		api.POST("/xray/apply", s.handleApplyXrayConfig)

		// Settings
		api.GET("/settings", s.handleGetSettings)
		api.PUT("/settings", s.handleUpdateSettings)
	}

	// Legacy API v1 routes (keep for backward compatibility)
	v1 := s.router.Group("/api/v1")
	{
		// Public routes
		v1.POST("/login", s.handleLogin)
		v1.GET("/health", s.handleHealth)

		// Protected routes (require JWT)
		protected := v1.Group("")
		protected.Use(s.authMiddleware())
		{
			// Dashboard
			protected.GET("/dashboard", s.handleDashboard)

			// Users management
			users := protected.Group("/users")
			{
				users.GET("", s.handleListUsers)
				users.POST("", s.handleCreateUser)
				users.GET("/:id", s.handleGetUser)
				users.PUT("/:id", s.handleUpdateUser)
				users.DELETE("/:id", s.handleDeleteUser)
				users.POST("/:id/reset-traffic", s.handleResetUserTraffic)
			}

			// Inbounds management
			inbounds := protected.Group("/inbounds")
			{
				inbounds.GET("", s.handleListInbounds)
				inbounds.POST("", s.handleCreateInbound)
				inbounds.GET("/:id", s.handleGetInbound)
				inbounds.PUT("/:id", s.handleUpdateInbound)
				inbounds.DELETE("/:id", s.handleDeleteInbound)
			}

			// Outbounds management
			outbounds := protected.Group("/outbounds")
			{
				outbounds.GET("", s.handleListOutbounds)
				outbounds.POST("", s.handleCreateOutbound)
				outbounds.GET("/:id", s.handleGetOutbound)
				outbounds.PUT("/:id", s.handleUpdateOutbound)
				outbounds.DELETE("/:id", s.handleDeleteOutbound)
				outbounds.POST("/:id/test", s.handleTestOutbound)
				outbounds.POST("/parse-wireguard", s.handleParseWireGuardConfig)
			}

			// Routing rules
			routing := protected.Group("/routing")
			{
				routing.GET("", s.handleListRoutingRules)
				routing.POST("", s.handleCreateRoutingRule)
				routing.PUT("/:id", s.handleUpdateRoutingRule)
				routing.DELETE("/:id", s.handleDeleteRoutingRule)
			}

			// Domains management
			domains := protected.Group("/domains")
			{
				domains.GET("", s.handleListDomains)
				domains.GET("/scan-certs", s.handleScanCertificates)
				domains.POST("/scan-import", s.handleScanAndImportCertificates)
				domains.POST("", s.handleCreateDomain)
				domains.PUT("/:id", s.handleUpdateDomain)
				domains.DELETE("/:id", s.handleDeleteDomain)
			}

			// Logout
			protected.POST("/logout", s.handleLogout)

			// Settings
			protected.GET("/settings", s.handleGetSettings)
			protected.PUT("/settings", s.handleUpdateSettings)

			// Xray control
			xray := protected.Group("/xray")
			{
				xray.GET("/status", s.handleXrayStatus)
				xray.POST("/restart", s.handleXrayRestart)
				xray.GET("/config", s.handleGetXrayConfig)
				xray.POST("/apply", s.handleApplyXrayConfig)
			}
		}
	}

	// Subscription routes (public, but require user path)
	s.router.GET("/sub/:path", s.handleSubscription)
	s.router.GET("/sub/:path/:format", s.handleSubscription)
}

// handleHealth returns server health status
func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

// JSON response helpers
func jsonOK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}

func jsonError(c *gin.Context, code int, message string) {
	c.JSON(code, gin.H{
		"success": false,
		"error":   message,
	})
}

func jsonCreated(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    data,
	})
}

func mustSub(f fs.FS, path string) fs.FS {
	sub, err := fs.Sub(f, path)
	if err != nil {
		panic(err)
	}
	return sub
}
