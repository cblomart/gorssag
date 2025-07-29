package web

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type SwaggerServer struct {
	enabled bool
}

func NewSwaggerServer(enabled bool) *SwaggerServer {
	return &SwaggerServer{enabled: enabled}
}

func (s *SwaggerServer) RegisterRoutes(router *gin.Engine) {
	if !s.enabled {
		return
	}

	// Serve Swagger UI
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
