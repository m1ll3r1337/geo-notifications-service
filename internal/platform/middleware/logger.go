// Package middleware provides HTTP middleware for Gin router.
package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/m1ll3r1337/geo-notifications-service/internal/platform/logger"
)

func GinStructuredLogger(l *logger.Logger, minLevel logger.Level) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()
		path := ctx.Request.URL.Path
		query := ctx.Request.URL.RawQuery

		ctx.Next()

		status := ctx.Writer.Status()
		latency := time.Since(start)
		size := ctx.Writer.Size()

		level := logger.LevelInfo
		switch {
		case status >= 500:
			level = logger.LevelError
		case status >= 400:
			level = logger.LevelWarn
		}

		if level < minLevel {
			return
		}

		l.Log(ctx.Request.Context(), level, "http",
			"status", status,
			"target", ctx.Request.Method+" "+path,
			"query", query,
			"ip", ctx.ClientIP(),
			"ua", ctx.Request.UserAgent(),
			"latency_ms", float64(latency)/float64(time.Millisecond),
			"size", size,
		)
	}
}
