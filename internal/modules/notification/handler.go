package notification

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iamarpitzala/acareca/internal/shared/response"
	"github.com/iamarpitzala/acareca/internal/shared/util"
)

type Handler struct {
	svc Service
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) ListNotifications(c *gin.Context) {
	uid, ok := util.GetUserID(c)
	if !ok {
		return
	}

	page := 1
	limit := 20
	if v := strings.TrimSpace(c.Query("page")); v != "" {
		if p, err := strconv.Atoi(v); err == nil && p > 0 {
			page = p
		}
	}
	if v := strings.TrimSpace(c.Query("limit")); v != "" {
		if l, err := strconv.Atoi(v); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	var statusPtr *Status
	if v := strings.TrimSpace(c.Query("status")); v != "" {
		s := Status(strings.ToUpper(v))
		switch s {
		case StatusPending, StatusDelivered, StatusRead, StatusDismissed, StatusFailed:
			statusPtr = &s
		}
	}

	res, err := h.svc.ListNotifications(c.Request.Context(), uid, statusPtr, page, limit)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err)
		return
	}

	response.JSON(c, http.StatusOK, res, "Notifications fetched successfully")
}

func (h *Handler) MarkRead(c *gin.Context) {
	uid, ok := util.GetUserID(c)
	if !ok {
		return
	}

	idParam := c.Param("id")
	nid, err := uuid.Parse(idParam)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}

	if err := h.svc.MarkRead(c.Request.Context(), uid, nid); err != nil {
		response.Error(c, http.StatusNotFound, err)
		return
	}

	response.Message(c, http.StatusOK, "Notification marked as read")
}

func (h *Handler) MarkDismissed(c *gin.Context) {
	uid, ok := util.GetUserID(c)
	if !ok {
		return
	}

	idParam := c.Param("id")
	nid, err := uuid.Parse(idParam)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}

	if err := h.svc.MarkDismissed(c.Request.Context(), uid, nid); err != nil {
		response.Error(c, http.StatusNotFound, err)
		return
	}

	response.Message(c, http.StatusOK, "Notification dismissed")
}
