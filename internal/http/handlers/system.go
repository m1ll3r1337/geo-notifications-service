package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Pinger interface {
	Ping(ctx context.Context) error
}

type Logger interface {
	Info(ctx context.Context, msg string, args ...any)
	Error(ctx context.Context, msg string, args ...any)
}

type Dependency struct {
	Name   string
	Pinger Pinger
}

type System struct {
	deps []Dependency
	log  Logger
}

func NewSystem(log Logger, deps ...Dependency) *System {
	return &System{deps: deps, log: log}
}

type dependencyStatus struct {
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}

type healthResponse struct {
	Status       string                      `json:"status"`
	Timestamp    time.Time                   `json:"timestamp"`
	Dependencies map[string]dependencyStatus `json:"dependencies"`
}

func (h *System) Health(ctx *gin.Context) {
	resp := healthResponse{
		Status:       "ok",
		Timestamp:    time.Now().UTC(),
		Dependencies: map[string]dependencyStatus{},
	}
	code := http.StatusOK

	for _, d := range h.deps {
		err := d.Pinger.Ping(ctx)

		if err != nil {
			resp.Status = "degraded"
			resp.Dependencies[d.Name] = dependencyStatus{Status: "down", Error: err.Error()}
			code = http.StatusServiceUnavailable
			if h.log != nil {
				h.log.Error(ctx, "health check failed", "component", d.Name, "error", err)
			}
		} else {
			resp.Dependencies[d.Name] = dependencyStatus{Status: "ok"}
			if h.log != nil {
				h.log.Info(ctx, "health check ok", "component", d.Name)
			}
		}
	}

	ctx.JSON(code, resp)
}
