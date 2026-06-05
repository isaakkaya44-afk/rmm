package main

import (
	"github.com/gin-gonic/gin"
	"github.com/rmm-platform/backend/internal/alerts"
	"github.com/rmm-platform/backend/internal/auth"
	"github.com/rmm-platform/backend/internal/customers"
	"github.com/rmm-platform/backend/internal/devices"
	"github.com/rmm-platform/backend/internal/monitoring"
	"github.com/rmm-platform/backend/internal/realtime"
	"github.com/rmm-platform/backend/internal/remote"
	"github.com/rmm-platform/backend/internal/shared/config"
	"github.com/rmm-platform/backend/internal/shared/database"
	"github.com/rmm-platform/backend/internal/shared/logging"
	"github.com/rmm-platform/backend/internal/tickets"
	"github.com/rs/zerolog/log"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	_ "github.com/rmm-platform/backend/docs"
)

// @title RMM Platform API
// @version 1.0
// @description Remote Monitoring & Management Platform API
// @host localhost:8080
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	cfg := config.Load()

	logging.Init(cfg.Server.Environment)
	log.Info().Msg("starting RMM Platform API")

	db, err := database.Connect(&cfg.Database)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}
	defer db.Close()

	hub := realtime.NewHub()
	go hub.Run()

	r := gin.Default()
	r.Use(corsMiddleware())
	r.Use(requestLogger())

	api := r.Group("/api/v1")
	{
		auth.RegisterRoutes(api, db, cfg)
		devices.RegisterRoutes(api, db, cfg, hub, alerts.NewRepository(db))
		monitoring.RegisterRoutes(api, db)
		alerts.RegisterRoutes(api, db, cfg, hub)
		tickets.RegisterRoutes(api, db, cfg, hub)
		customers.RegisterRoutes(api, db)
		remote.RegisterRoutes(api, db)
	}

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.GET("/ws", realtime.HandleWebSocket(hub, cfg))
	r.GET("/health", healthCheck(db))

	log.Info().Str("port", cfg.Server.Port).Msg("server listening")
	if err := r.Run(":" + cfg.Server.Port); err != nil {
		log.Fatal().Err(err).Msg("server failed to start")
	}
}

func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Info().
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("status", c.Writer.Status()).
			Dur("latency", 0).
			Msg("request")
		c.Next()
	}
}

func healthCheck(db interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "rmm-platform-api",
			"version": "1.0.0",
		})
	}
}
