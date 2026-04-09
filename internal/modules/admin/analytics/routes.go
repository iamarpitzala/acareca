package analytics

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg *gin.RouterGroup, h *Handler) {
	// Analytics routes
	rg.GET("/user-growth", h.GetUserGrowth)
	rg.GET("/subscriptions", h.GetSubscriptionMetrics)
	rg.GET("/active-users", h.GetActiveUsers)
	rg.GET("/practitioners", h.ListPractitionersWithDetails)
	rg.GET("/practitioners/:id", h.GetPractitionerDetails)
	
	// Dashboard routes - Practitioner
	dashboard := rg.Group("/dashboard")
	practitionerDashboard := dashboard.Group("/practitioner")
	practitionerDashboard.GET("/overview", h.GetPractitionerOverview)
	practitionerDashboard.GET("/resource-analytics", h.GetResourceAnalytics)
	
	// Dashboard routes - Accountant
	accountantDashboard := dashboard.Group("/accountant")
	accountantDashboard.GET("/overview", h.GetAccountantOverview)
	accountantDashboard.GET("/resource-access-timeseries", h.GetResourceAccessTimeseries)
	
	// Dashboard routes - Billing
	billingDashboard := dashboard.Group("/billing")
	billingDashboard.GET("/platform-revenue", h.GetPlatformRevenue)
	billingDashboard.GET("/plan-distribution", h.GetPlanDistribution)
	
	// Billing subscription records (separate from dashboard)
	billing := rg.Group("/billing")
	billing.GET("/subscriptions", h.ListSubscriptionRecords)
}
