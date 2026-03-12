package form

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/iamarpitzala/acareca/internal/modules/builder/detail"
	"github.com/iamarpitzala/acareca/internal/shared/response"
	"github.com/iamarpitzala/acareca/internal/shared/util"
)

type IHandler interface {
	Sync(c *gin.Context)
	CreateFormWithFields(c *gin.Context)
	UpdateFormWithFields(c *gin.Context)
}

type handler struct {
	svc IService
}

func NewHandler(svc IService) IHandler {
	return &handler{svc: svc}
}

func (h *handler) Sync(c *gin.Context) {
	versionID, ok := util.ParseUuidID(c, "version_id")
	if !ok {
		return
	}
	practitionerID, ok := util.GetPractitionerID(c)
	if !ok {
		return
	}
	var req RqBulkSyncFields
	if err := util.BindAndValidate(c, &req); err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}
	result, err := h.svc.BulkSyncFields(c.Request.Context(), versionID, practitionerID, &req)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err)
		return
	}
	response.JSON(c, http.StatusOK, result)
}

func (h *handler) CreateFormWithFields(c *gin.Context) {
	clinicID, ok := util.GetClinicID(c)
	if !ok {
		return
	}
	practitionerID, ok := util.GetPractitionerID(c)
	if !ok {
		return
	}
	var req RqCreateFormWithFields
	if err := util.BindAndValidate(c, &req); err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}
	req.ClinicID = clinicID
	if req.Status == "" {
		req.Status = detail.StatusDraft
	}
	form, syncResult, err := h.svc.CreateWithFields(c.Request.Context(), &req, practitionerID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err)
		return
	}
	response.JSON(c, http.StatusCreated, gin.H{"form": form, "fields_sync": syncResult})
}

func (h *handler) UpdateFormWithFields(c *gin.Context) {

	clinicID, ok := util.GetClinicID(c)
	if !ok {
		return
	}
	practitionerID, ok := util.GetPractitionerID(c)
	if !ok {
		return
	}
	var req RqUpdateFormWithFields
	if err := util.BindAndValidate(c, &req); err != nil {
		response.Error(c, http.StatusBadRequest, err)
		return
	}
	req.ClinicID = clinicID
	form, syncResult, err := h.svc.UpdateWithFields(c.Request.Context(), &req, practitionerID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err)
		return
	}
	response.JSON(c, http.StatusOK, gin.H{"form": form, "fields_sync": syncResult})
}
