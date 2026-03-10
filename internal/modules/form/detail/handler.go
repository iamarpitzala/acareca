package detail

import (
	"github.com/gin-gonic/gin"
)

type IHandler interface {
	CreateForm(c *gin.Context)
	GetForm(c *gin.Context)
	UpdateForm(c *gin.Context)
	DeleteForm(c *gin.Context)
}

// type handler struct {
// 	svc IService
// }
