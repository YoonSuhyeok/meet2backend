package router

import (
	"meetBack/internal/handler"

	"github.com/gin-gonic/gin"
)

func RegisterHealthRoutes(r gin.IRouter, healthHandler *handler.HealthHandler) {
	r.GET("/", healthHandler.Hello)
	r.GET("/healthz", healthHandler.Health)
}
