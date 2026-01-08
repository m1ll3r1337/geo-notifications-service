package main

import (
	"fmt"

	"github.com/m1ll3r1337/geo-notifications-service/internal/app"
)

func main() {
	s := app.NewServer("0.0.0.0:8080")
	if err := s.Start(); err != nil {
		fmt.Println("Error starting server:", err)
	}
}
