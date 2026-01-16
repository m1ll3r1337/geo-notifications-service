package middleware

import (
	"crypto/subtle"

	"github.com/gin-gonic/gin"

	"github.com/m1ll3r1337/geo-notifications-service/internal/errs"
)

func APIKey(expected string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		const op = "http.middleware.apikey"

		if expected == "" {
			_ = ctx.Error(errs.E(errs.KindInternal, "API_KEY_NOT_CONFIGURED", op, "api key not configured", nil, nil))
			ctx.Abort()
			return
		}

		got := ctx.GetHeader("X-API-Key")
		if got == "" || subtle.ConstantTimeCompare([]byte(got), []byte(expected)) != 1 {
			_ = ctx.Error(errs.E(errs.KindUnauthorized, "INVALID_API_KEY", op, "invalid api key", nil, nil))
			ctx.Abort()
			return
		}

		ctx.Next()
	}
}
