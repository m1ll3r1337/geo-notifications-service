package app

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Server struct {
	server *http.Server
	router *gin.Engine
}

func NewServer(addr string) *Server {
	r := gin.Default()
	setupRoutes(r)
	s := Server{
		server: &http.Server{
			Addr:              addr,
			ReadHeaderTimeout: 15 * time.Second,
			Handler:           r,
		},
		router: r,
	}

	return &s
}

func (s *Server) Start() error {
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown() error {
	return s.server.Close()
}
