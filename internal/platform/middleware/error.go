package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/m1ll3r1337/geo-notifications-service/internal/errs"
	"github.com/m1ll3r1337/geo-notifications-service/internal/platform/logger"
)

type APIError struct {
	Error     string            `json:"error"`
	Kind      errs.Kind         `json:"kind"`
	Code      string            `json:"code,omitempty"`
	Fields    map[string]string `json:"fields,omitempty"`
	RequestID string            `json:"request_id,omitempty"`
}

func Error(log *logger.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.Next()

		if len(ctx.Errors) == 0 {
			return
		}

		raw := ctx.Errors.Last().Err
		if raw == nil || ctx.Writer.Written() {
			return
		}

		status, resp, logLevel := mapErr(raw)
		resp.RequestID = GetRequestID(ctx)

		switch logLevel {
		case "error":
			log.Error(ctx.Request.Context(), "request failed",
				"error", raw,
				"status", status,
				"path", ctx.Request.URL.Path,
				"method", ctx.Request.Method,
				"request_id", resp.RequestID,
			)
		case "warn":
			log.Warn(ctx.Request.Context(), "request rejected",
				"error", raw,
				"status", status,
				"path", ctx.Request.URL.Path,
				"method", ctx.Request.Method,
				"request_id", resp.RequestID,
			)
		}

		ctx.AbortWithStatusJSON(status, resp)
	}
}

func mapErr(err error) (status int, resp APIError, logLevel string) {
	if e, ok := errs.As(err); ok {
		resp.Kind = e.Kind
		resp.Code = e.Code

		switch e.Kind {
		case errs.KindInvalid:
			resp.Error = "invalid request"
			resp.Fields = e.Fields
			return http.StatusBadRequest, resp, "warn"

		case errs.KindNotFound:
			resp.Error = "not found"
			return http.StatusNotFound, resp, "warn"

		case errs.KindUnauthorized:
			resp.Error = "unauthorized"
			return http.StatusUnauthorized, resp, "warn"

		case errs.KindForbidden:
			resp.Error = "forbidden"
			return http.StatusForbidden, resp, "warn"

		case errs.KindConflict:
			resp.Error = "conflict"
			return http.StatusConflict, resp, "warn"

		default:
			resp.Error = "internal server error"
			return http.StatusInternalServerError, resp, "error"
		}
	}

	resp.Kind = errs.KindInternal
	resp.Error = "internal server error"
	return http.StatusInternalServerError, resp, "error"
}
