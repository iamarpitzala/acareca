package auth

import (
	"github.com/gin-gonic/gin"
)

// func RegisterRoutes(rg *gin.RouterGroup, h *Handler, cfg *config.Config) {
// 	auth := rg.Group("/auth")
// 	auth.Use(middleware.Auth(cfg))
// 	{
// 		auth.POST("/register", h.Register)
// 		auth.POST("/login", h.Login)
// 	}
// }

func RegisterRoutes(rg *gin.RouterGroup, h *Handler) {
	auth := rg.Group("/auth")

	auth.POST("/register", h.Register)
	auth.POST("/login", h.Login)
}
