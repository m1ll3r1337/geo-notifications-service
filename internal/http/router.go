package http

import (
	"github.com/gin-gonic/gin"

	"github.com/m1ll3r1337/geo-notifications-service/internal/http/handlers"
	"github.com/m1ll3r1337/geo-notifications-service/internal/platform/logger"
	"github.com/m1ll3r1337/geo-notifications-service/internal/platform/middleware"
)

func NewRouter(log *logger.Logger, level logger.Level, incidents *handlers.Incidents, apiKey string) *gin.Engine {
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

	setupRoutes(r, incidents, apiKey)
	return r
}

func setupRoutes(r *gin.Engine, incidents *handlers.Incidents, apiKey string) {
	v1 := r.Group("/api/v1")

	inc := v1.Group("/incidents", middleware.APIKey(apiKey))
	{
		inc.POST("", incidents.Create)
		inc.GET("", incidents.List)
		inc.GET("/:id", incidents.GetByID)
		inc.PUT("/:id", incidents.Update)
		inc.PATCH("/:id", incidents.Update)
		inc.DELETE("/:id", incidents.Deactivate)
	}

	v1.POST("/location/check", incidents.Check)

}
