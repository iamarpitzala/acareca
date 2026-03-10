package clinic

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(rg *gin.RouterGroup, h IHandler) {
	clinic := rg.Group("/clinic")

	clinic.PUT("/create", h.CreateClinic)
	clinic.GET("/all", h.GetClinics)
	clinic.GET("/:id", h.GetClinicByID)
	clinic.DELETE("/:id", h.DeleteClinic)
}
