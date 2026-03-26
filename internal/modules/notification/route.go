package notification

import (
	"github.com/gin-gonic/gin"
	"github.com/iamarpitzala/acareca/internal/shared/middleware"
	"github.com/iamarpitzala/acareca/pkg/config"
)

func RegisterRoutes(rg *gin.RouterGroup, h IHandler, cfg *config.Config) {
	nft := rg.Group("/notification")
	nft.Use(middleware.Auth(cfg), middleware.AuditContext())

	nft.GET("", h.ListNotifications)
	nft.GET("/dismissed", h.MarkDismissed)
	nft.GET("/read", h.MarkRead)
}
