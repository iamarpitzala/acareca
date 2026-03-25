package notification

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg *gin.RouterGroup, h *Handler) {
	rg.GET("/", h.ListNotifications)
	rg.PATCH("/:id/read", h.MarkRead)
	rg.PATCH("/:id/dismiss", h.MarkDismissed)
}
