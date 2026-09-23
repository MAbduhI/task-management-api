package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/MAbduhI/task-management-api/internal/delivery/http/middleware"
)

type RouterConfig struct {
	JWTSecret     string
	AuthHandler   *AuthHandler
	TeamHandler   *TeamHandler
	TaskHandler   *TaskHandler
	SystemHandler *SystemHandler
}

func SetupRouter(cfg RouterConfig) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// Global middlewares
	r.Use(middleware.CORS())
	r.Use(middleware.RequestID())
	r.Use(middleware.StructuredLogger())
	r.Use(middleware.Recovery())

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// System / Maintenance routes
	if cfg.SystemHandler != nil {
		system := r.Group("/system")
		{
			system.POST("/migrate/up", cfg.SystemHandler.MigrateUp)
			system.POST("/migrate/down", cfg.SystemHandler.MigrateDown)
			system.POST("/seed/up", cfg.SystemHandler.SeedUp)
			system.POST("/seed/down", cfg.SystemHandler.SeedDown)
		}
	}

	// Public auth routes
	auth := r.Group("/auth")
	{
		auth.POST("/register", cfg.AuthHandler.Register)
		auth.POST("/login", cfg.AuthHandler.Login)
	}

	// Protected routes
	protected := r.Group("")
	protected.Use(middleware.Auth(cfg.JWTSecret))
	{
		// Teams
		teams := protected.Group("/teams")
		{
			teams.POST("", cfg.TeamHandler.Create)
			teams.GET("", cfg.TeamHandler.ListMyTeams)
			teams.POST("/:uuid/members", cfg.TeamHandler.AddMember)
		}

		// Tasks
		tasks := protected.Group("/tasks")
		{
			tasks.POST("", cfg.TaskHandler.Create)
			tasks.GET("", cfg.TaskHandler.List)
			tasks.GET("/:uuid", cfg.TaskHandler.GetDetail)
			tasks.PUT("/:uuid", cfg.TaskHandler.Update)
			tasks.DELETE("/:uuid", cfg.TaskHandler.Delete)
			tasks.POST("/:uuid/assign", cfg.TaskHandler.Assign)
			tasks.GET("/:uuid/logs", cfg.TaskHandler.ListLogs)
		}
	}

	return r
}
