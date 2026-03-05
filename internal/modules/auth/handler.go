package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/iamarpitzala/acareca/internal/shared/response"
	"github.com/iamarpitzala/acareca/internal/shared/util"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Register(c *gin.Context) {
	var req RqUser
	if err := util.BindAndValidate(c, &req); err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}

	user, err := h.svc.Register(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, ErrEmailTaken) {
			response.Error(c, http.StatusConflict, err)
			return
		}
		response.Error(c, http.StatusInternalServerError, err)
		return
	}

	response.JSON(c, http.StatusCreated, user)
}

func (h *Handler) Login(c *gin.Context) {
	var req RqLogin
	if err := util.BindAndValidate(c, &req); err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}

	token, err := h.svc.Login(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, ErrInvalidPassword) {
			response.Error(c, http.StatusUnauthorized, err)
			return
		}
		response.Error(c, http.StatusInternalServerError, err)
		return
	}

	response.JSON(c, http.StatusOK, token)
}
