package app

import (
	"github.com/gin-gonic/gin"
	"github.com/m1ll3r1337/geo-notifications-service/internal/platform/logger"
	"github.com/m1ll3r1337/geo-notifications-service/internal/platform/middleware"
)

func NewRouter(log *logger.Logger, level logger.Level) *gin.Engine {
	if level == logger.LevelDebug {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	r.Use(middleware.GinStructuredLogger(log, level))
	r.Use(middleware.Recovery(log))

	setupRoutes(r)
	return r
}

func setupRoutes(r *gin.Engine) {
	r.GET("/hello", func(ctx *gin.Context) {
		ctx.String(200, "Hello, World!")
	})
}
