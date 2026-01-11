// Package app provides HTTP server setup and routing.
package app

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Server struct {
	server *http.Server
	router *gin.Engine
}

func NewServer(cfg Config, router *gin.Engine, stdLog *log.Logger) *Server {
	s := Server{
		server: &http.Server{
			Addr:              cfg.Addr,
			ReadTimeout:       10 * time.Second,
			WriteTimeout:      10 * time.Second,
			IdleTimeout:       60 * time.Second,
			ReadHeaderTimeout: 15 * time.Second,
			Handler:           router,
			ErrorLog:          stdLog,
		},
		router: router,
	}
	return &s
}

func (s *Server) Start() error {
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *Server) Close() error {
	return s.server.Close()
}
