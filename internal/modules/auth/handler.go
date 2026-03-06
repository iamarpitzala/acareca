package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/iamarpitzala/acareca/internal/shared/response"
	"github.com/iamarpitzala/acareca/internal/shared/util"
)

type IHandler interface {
	Register(c *gin.Context)
	Login(c *gin.Context)
	GoogleAuthURL(c *gin.Context)
	GoogleCallback(c *gin.Context)
}

type handler struct {
	svc Service
}

func NewHandler(svc Service) IHandler {
	return &handler{svc: svc}
}

func (h *handler) Register(c *gin.Context) {
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

func (h *handler) Login(c *gin.Context) {
	var req RqLogin
	if err := util.BindAndValidate(c, &req); err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}

	token, err := h.svc.Login(c.Request.Context(), &req)
	if err != nil {
		if errors.Is(err, ErrInvalidPassword) || errors.Is(err, ErrOAuthOnly) {
			response.Error(c, http.StatusUnauthorized, err)
			return
		}
		response.Error(c, http.StatusInternalServerError, err)
		return
	}

	response.JSON(c, http.StatusOK, token)
}

// GoogleLogin returns the Google OAuth consent-screen URL.
func (h *handler) GoogleLogin(c *gin.Context) {
	state := util.NewUUID()
	result := h.svc.GoogleAuthURL(state)
	response.JSON(c, http.StatusOK, result)
}

// GoogleCallback handles the redirect from Google after the user consents.
func (h *handler) GoogleCallback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		response.Error(c, http.StatusBadRequest, errors.New("missing oauth code"))
		return
	}

	token, err := h.svc.GoogleCallback(c.Request.Context(), code)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err)
		return
	}

	response.JSON(c, http.StatusOK, token)
}

// GoogleAuthURL implements [IHandler].
func (h *handler) GoogleAuthURL(c *gin.Context) {
	state := util.NewUUID()
	result := h.svc.GoogleAuthURL(state)
	response.JSON(c, http.StatusOK, result)
}
