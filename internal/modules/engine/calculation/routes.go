package calculation

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg *gin.RouterGroup, h IHandler) {
	rg.POST("/calculate", h.Calculate)
}
