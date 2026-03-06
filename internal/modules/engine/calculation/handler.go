package calculation

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/iamarpitzala/acareca/internal/shared/response"
	"github.com/iamarpitzala/acareca/internal/shared/util"
)

type IHandler interface {
	Calculate(c *gin.Context)
}

type handler struct {
	svc Service
}

func NewHandler(svc Service) IHandler {
	return &handler{svc: svc}
}

// GetEngine implements [IHandler].
func (h *handler) Calculate(c *gin.Context) {
	var entry Entry
	if err := util.BindAndValidate(c, &entry); err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}
	result, err := h.svc.Calculate(c.Request.Context(), &entry)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err)
		return
	}
	response.JSON(c, http.StatusOK, gin.H{
		"income":  result.Income,
		"expense": result.Expense,
		"result":  result.Result,
	})
}
