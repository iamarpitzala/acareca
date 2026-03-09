package subscription

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg *gin.RouterGroup, h IHandler) {
	rg.POST("/subscriptions", h.CreateSubscription)
	rg.GET("/subscriptions", h.ListSubscriptions)
	rg.GET("/subscriptions/:id", h.GetSubscription)
	rg.PATCH("/subscriptions/:id", h.UpdateSubscription)
	rg.DELETE("/subscriptions/:id", h.DeleteSubscription)
}
