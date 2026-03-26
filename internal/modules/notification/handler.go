package notification

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/iamarpitzala/acareca/internal/shared/response"
	"github.com/iamarpitzala/acareca/internal/shared/util"
)

type IHandler interface {
	ListNotifications(c *gin.Context)
	MarkRead(c *gin.Context)
	MarkDismissed(c *gin.Context)
}

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

// getEntityID reads the entity ID (practitioner or accountant) from context.

func (h *Handler) ListNotifications(c *gin.Context) {
	entityID, ok := util.GetEntityID(c)
	if !ok {
		return
	}

	var filter FilterNotification
	if err := util.BindAndValidate(c, &filter); err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}

	res, err := h.svc.ListNotifications(c.Request.Context(), entityID, filter)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err)
		return
	}

	response.JSON(c, http.StatusOK, res, "Notifications fetched successfully")
}

func (h *Handler) MarkRead(c *gin.Context) {
	entityID, ok := util.GetEntityID(c)
	if !ok {
		return
	}

	nid, ok := util.ParseUuidID(c, "id")
	if !ok {
		return
	}

	if err := h.svc.MarkRead(c.Request.Context(), entityID, nid); err != nil {
		response.Error(c, http.StatusNotFound, err)
		return
	}

	response.Message(c, http.StatusOK, "Notification marked as read")
}

func (h *Handler) MarkDismissed(c *gin.Context) {
	entityID, ok := util.GetEntityID(c)
	if !ok {
		return
	}

	nid, ok := util.ParseUuidID(c, "id")
	if !ok {
		return
	}

	if err := h.svc.MarkDismissed(c.Request.Context(), entityID, nid); err != nil {
		response.Error(c, http.StatusNotFound, err)
		return
	}

	response.Message(c, http.StatusOK, "Notification dismissed")
}
