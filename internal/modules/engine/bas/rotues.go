package bas

import (
	"github.com/gin-gonic/gin"
	"github.com/iamarpitzala/acareca/internal/shared/middleware"
	"github.com/iamarpitzala/acareca/pkg/config"
)

// RegisterRoutes mounts all BAS endpoints under the given router group.
//
// Resulting routes (relative to the group prefix, e.g. /api/v1):
//
//	GET /bas/clinic/:clinic_id/summary     → quarterly BAS (ATO labels)
//	GET /bas/clinic/:clinic_id/by-account  → per COA account breakdown
//	GET /bas/clinic/:clinic_id/monthly     → monthly GST accrual
func RegisterRoutes(rg *gin.RouterGroup, h IHandler, cfg *config.Config) {
	bas := rg.Group("/bas")
	bas.Use(middleware.Auth(cfg))

	clinic := bas.Group("/clinic/:clinic_id")
	{
		clinic.GET("/summary", h.GetQuarterlySummary)
		clinic.GET("/by-account", h.GetByAccount)
		clinic.GET("/monthly", h.GetMonthly)
	}
}
