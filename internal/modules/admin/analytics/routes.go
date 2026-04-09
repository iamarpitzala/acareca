package analytics

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg *gin.RouterGroup, h *Handler) {
	rg.GET("/user-growth", h.GetUserGrowth)
	rg.GET("/subscriptions", h.GetSubscriptionMetrics)
	rg.GET("/active-users", h.GetActiveUsers)
	rg.GET("/practitioners", h.ListPractitionersWithDetails)
	rg.GET("/practitioners/:id", h.GetPractitionerDetails)
}
