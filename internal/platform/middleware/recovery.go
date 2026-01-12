package middleware

import (
	"runtime/debug"

	"github.com/gin-gonic/gin"

	"github.com/m1ll3r1337/geo-notifications-service/internal/errs"
	"github.com/m1ll3r1337/geo-notifications-service/internal/platform/logger"
)

func Recovery(log *logger.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				stack := debug.Stack()

				log.Error(ctx.Request.Context(), "panic recovered",
					"error", rec,
					"stack", string(stack),
					"target", ctx.Request.Method+" "+ctx.Request.URL.Path,
					"request_id", GetRequestID(ctx),
				)

				_ = ctx.Error(errs.E(
					errs.KindInternal,
					"PANIC",
					"http.middleware.panic_recovery",
					"internal server error",
					nil,
					nil,
				))
				ctx.Abort()
			}
		}()

		ctx.Next()
	}
}
