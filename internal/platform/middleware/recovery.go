package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/m1ll3r1337/geo-notifications-service/internal/platform/logger"
)

func Recovery(log *logger.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				stack := debug.Stack()
				log.Error(ctx.Request.Context(), "panic recovered",
					"error", err,
					"stack", string(stack),
					"target", ctx.Request.Method+" "+ctx.Request.URL.Path,
				)
				ctx.JSON(http.StatusInternalServerError, gin.H{
					"error": "internal server error",
				})
				ctx.Abort()
			}
		}()
		ctx.Next()
	}
}
