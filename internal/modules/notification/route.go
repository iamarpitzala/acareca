package notification

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(rg *gin.RouterGroup, h IHandler) {
	nft := rg.Group("/notification")
	nft.GET("", h.ListNotifications)
	nft.GET("/dismissed", h.MarkDismissed)
	nft.GET("/read", h.MarkRead)
}
