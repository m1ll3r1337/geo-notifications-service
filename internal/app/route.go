package app

import "github.com/gin-gonic/gin"

func setupRoutes(r *gin.Engine) {
	r.GET("/hello", func(ctx *gin.Context) {
		ctx.String(200, "Hello, World!")
	})
}
