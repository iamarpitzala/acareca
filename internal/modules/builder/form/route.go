package form

import "github.com/gin-gonic/gin"

func RegisterRoutes(rg *gin.RouterGroup, h IHandler) {
	rg.POST("/sync", h.Sync)
	rg.POST("/create", h.CreateFormWithFields)
	rg.POST("/update", h.UpdateFormWithFields)
}
