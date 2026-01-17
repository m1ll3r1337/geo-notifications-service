package http

import (
	"github.com/gin-gonic/gin"

	"github.com/m1ll3r1337/geo-notifications-service/internal/http/handlers"
	"github.com/m1ll3r1337/geo-notifications-service/internal/platform/logger"
	"github.com/m1ll3r1337/geo-notifications-service/internal/platform/middleware"
)

func NewRouter(log *logger.Logger, level logger.Level, incidents *handlers.Incidents, system *handlers.System, apiKey string) *gin.Engine {
	if level == logger.LevelDebug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// Order matters
	r.Use(middleware.RequestID())
	r.Use(middleware.GinStructuredLogger(log, level))
	r.Use(middleware.Error(log))
	r.Use(middleware.Recovery(log))

	setupRoutes(r, incidents, system, apiKey)
	return r
}

func setupRoutes(r *gin.Engine, incidents *handlers.Incidents, system *handlers.System, apiKey string) {
	v1 := r.Group("/api/v1")

	v1.GET("/health", system.Health)

	protected := v1.Group("", middleware.APIKey(apiKey))
	inc := protected.Group("/incidents")
	{
		inc.POST("", incidents.Create)
		inc.GET("", incidents.List)
		inc.GET("/:id", incidents.GetByID)
		inc.PUT("/:id", incidents.Update)
		inc.PATCH("/:id", incidents.Update)
		inc.DELETE("/:id", incidents.Deactivate)
		inc.GET("/stats", incidents.Stats)
	}

	v1.POST("/location/check", incidents.Check)

}
