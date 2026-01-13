package http

import (
	"github.com/gin-gonic/gin"

	"github.com/m1ll3r1337/geo-notifications-service/internal/http/handlers"
	"github.com/m1ll3r1337/geo-notifications-service/internal/platform/logger"
	"github.com/m1ll3r1337/geo-notifications-service/internal/platform/middleware"
)

func NewRouter(log *logger.Logger, level logger.Level, incidents *handlers.Incidents) *gin.Engine {
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

	setupRoutes(r, incidents)
	return r
}

func setupRoutes(r *gin.Engine, incidents *handlers.Incidents) {
	v1 := r.Group("/v1")
	{
		v1.POST("/incidents", incidents.Create)
		v1.GET("/incidents", incidents.List)
		v1.GET("/incidents/:id", incidents.GetByID)
		v1.PATCH("/incidents/:id", incidents.Update)
		v1.DELETE("/incidents/:id", incidents.Deactivate)
	}
}
