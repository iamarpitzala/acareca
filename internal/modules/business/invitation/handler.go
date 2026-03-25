package invitation

import (
	"net/http"

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

// @Summary      Send an invitation
// @Description  Practitioner sends an invitation to a person's email.
// @Tags         invitation
// @Accept       json
// @Produce      json
// @Param        request body RqSendInvitation true "Email of the Accountant/Bookkeeper"
// @Success      201 {object} RsInvitation
// @Failure      400 {object} response.RsError
// @Failure      500 {object} response.RsError
// @Security     BearerToken
// @Router       /invite/ [post]
func (h *Handler) SendInvitation(c *gin.Context) {
	practID, ok := util.GetPractitionerID(c)
	if !ok {
		response.Error(c, http.StatusUnauthorized, nil)
		return
	}

	var req RqSendInvitation
	if err := util.BindAndValidate(c, &req); err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}

	res, err := h.svc.SendInvite(c.Request.Context(), practID, &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err)
		return
	}

	response.JSON(c, http.StatusCreated, res, "Invitation sent successfully")
}

// @Summary      Get invitation details
// @Description  Check if an invitation exists and if the user is already registered.
// @Tags         invitation
// @Produce      json
// @Param        id     path   string  true  "Invitation UUID"
// @Success      200 {object} RsInviteProcess
// @Failure      400 {object} response.RsError
// @Failure      500 {object} response.RsError
// @Router       /invite/{id} [get]
func (h *Handler) GetInvitation(c *gin.Context) {
	idParam := c.Param("id")
	inviteID, err := uuid.Parse(idParam)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}

	res, err := h.svc.GetInvitationDetails(c.Request.Context(), inviteID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err)
		return
	}

	response.JSON(c, http.StatusOK, res, "Invitation details fetched successfully")
}

// @Summary      Process invitation action
// @Description  Accept or Reject an invitation via POST.
// @Tags         invitation
// @Accept       json
// @Produce      json
// @Param        request body RqProcessAction true "Token ID and Action (accept/reject)"
// @Success      200 {object} RsInviteProcess
// @Failure      400 {object} response.RsError
// @Failure      500 {object} response.RsError
// @Router       /invite/process [post]
func (h *Handler) ProcessInvitation(c *gin.Context) {
	var req RqProcessAction
	if err := util.BindAndValidate(c, &req); err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}

	res, err := h.svc.ProcessInvitation(c.Request.Context(), &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err)
		return
	}

	response.JSON(c, http.StatusOK, res, "Invitation processed")
}
