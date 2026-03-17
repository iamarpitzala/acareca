package form

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iamarpitzala/acareca/internal/modules/builder/detail"
	"github.com/iamarpitzala/acareca/internal/shared/response"
	"github.com/iamarpitzala/acareca/internal/shared/util"
)

type IHandler interface {
	// Sync(c *gin.Context)
	GetById(c *gin.Context)
	CreateFormWithFields(c *gin.Context)
	UpdateFormWithFields(c *gin.Context)
	GetFormWithFields(c *gin.Context)
	List(c *gin.Context)
	Delete(c *gin.Context)
}

type handler struct {
	svc IService
}

func NewHandler(svc IService) IHandler {
	return &handler{svc: svc}
}

// // @Summary Bulk sync fields
// // @Description Synchronize multiple fields for a practitioner
// // @Tags form
// // @Accept json
// // @Produce json
// // @Param request body RqBulkSyncFields true "Sync request"
// // @Success 200 {object} RsBulkSyncFields
// // @Failure 400 {object} response.RsError
// // @Failure 500 {object} response.RsError
// // @Router /form/sync [post]
// func (h *handler) Sync(c *gin.Context) {
// 	practitionerID, ok := util.GetPractitionerID(c)
// 	if !ok {
// 		return
// 	}

// 	var req RqBulkSyncFields
// 	if err := util.BindAndValidate(c, &req); err != nil {
// 		response.Error(c, http.StatusBadRequest, err)
// 		return
// 	}
// 	result, err := h.svc.BulkSyncFields(c.Request.Context(), practitionerID, &req)
// 	if err != nil {
// 		response.Error(c, http.StatusInternalServerError, err)
// 		return
// 	}
// 	response.JSON(c, http.StatusOK, result, "Fields synchronized successfully")
// }

// @Summary fetch form details
// @Description fetch form detail
// @Tags form
// @Accept json
// @Produce json
// @Success 201 {object} RsFormWithFields
// @Failure 400 {object} response.RsError
// @Failure 500 {object} response.RsError
// @Router /form/{id} [post]
func (h *handler) GetById(c *gin.Context) {
	formId, ok := util.ParseUuidID(c, "id")
	if !ok {
		response.Error(c, http.StatusBadRequest, errors.New("invaild form id"))
		return
	}

	form, err := h.svc.GetFormByID(c.Request.Context(), formId)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err)
		return
	}

	response.JSON(c, http.StatusCreated, gin.H{"form": form}, "Form fetch successfully")
}

// @Summary Create form with fields
// @Description Create a new custom form along with its associated fields
// @Tags form
// @Accept json
// @Produce json
// @Param request body RqCreateFormWithFields true "Form creation request"
// @Success 201 {object} RsFormWithFields
// @Failure 400 {object} response.RsError
// @Failure 500 {object} response.RsError
// @Router /form [post]
func (h *handler) CreateFormWithFields(c *gin.Context) {
	practitionerID, ok := util.GetPractitionerID(c)
	if !ok {
		return
	}

	var req RqCreateFormWithFields
	if err := util.BindAndValidate(c, &req); err != nil {

		response.Error(c, http.StatusBadRequest, err)
		return
	}

	if req.Status == "" {
		req.Status = detail.StatusDraft
	}
	form, syncResult, err := h.svc.CreateWithFields(c.Request.Context(), &req, practitionerID)

	if err != nil {
		response.Error(c, http.StatusInternalServerError, err)
		return
	}

	response.JSON(c, http.StatusCreated, gin.H{"form": form, "fields_sync": syncResult}, "Form created successfully")
}

// @Summary Update form with fields
// @Description Update an existing form and sync its fields
// @Tags form
// @Accept json
// @Produce json
// @Param request body RqUpdateFormWithFields true "Form update request"
// @Success 200 {object} RsFormWithFields
// @Failure 400 {object} response.RsError
// @Failure 500 {object} response.RsError
// @Router /form [put]
func (h *handler) UpdateFormWithFields(c *gin.Context) {
	formID, ok := util.ParseUuidID(c, "id")
	if !ok {
		return
	}

	if formID == uuid.Nil {
		response.Error(c, http.StatusBadRequest, errors.New("form id is required"))
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
	req.ID = &formID
	form, syncResult, err := h.svc.UpdateWithFields(c.Request.Context(), &req, practitionerID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err)
		return
	}
	response.JSON(c, http.StatusOK, gin.H{"form": form, "fields_sync": syncResult}, "Form updated successfully")
}

// @Summary Get form by ID
// @Description Retrieve a specific form and its fields by ID
// @Tags form
// @Accept json
// @Produce json
// @Param id path string true "Form ID"
// @Success 200 {object} RsFormWithFields
// @Failure 404 {object} response.RsError
// @Failure 500 {object} response.RsError
// @Router /form/{id} [get]
func (h *handler) GetFormWithFields(c *gin.Context) {
	formID, ok := util.ParseUuidID(c, "id")
	if !ok {
		return
	}

	out, err := h.svc.GetFormWithFields(c.Request.Context(), formID)
	if err != nil {
		if errors.Is(err, detail.ErrNotFound) {
			response.Error(c, http.StatusNotFound, err)
			return
		}
		response.Error(c, http.StatusInternalServerError, err)
		return
	}
	response.JSON(c, http.StatusOK, out, "Form fetched successfully")
}

// @Summary List forms
// @Description List forms filtered by clinic, clinic name, method type, and sorted by clinic/owner share. If clinic_id not provided, lists all forms for practitioner's clinics.
// @Tags form
// @Accept json
// @Produce json
// @Param clinic_id query string false "Clinic ID (optional - if not provided, lists all practitioner's clinics)"
// @Param clinic_name query string false "Filter by clinic name (partial match)"
// @Param method query string false "Filter by method type (INDEPENDENT_CONTRACTOR or SERVICE_FEE)"
// @Param status query string false "Filter by status (DRAFT, PUBLISHED, or ARCHIVED)"
// @Param sort_by query string false "Sort by field (clinic_share or owner_share) - required if sort_order is provided"
// @Param sort_order query string false "Sort order (asc or desc) - required if sort_by is provided"
// @Success 200 {array} detail.RsFormDetail
// @Failure 400 {object} response.RsError
// @Failure 500 {object} response.RsError
// @Router /form [get]
func (h *handler) List(c *gin.Context) {
	practitionerID, ok := util.GetPractitionerID(c)
	if !ok {
		return
	}

	clinicIdStr := c.Query("clinic_id")
	clinicName := c.Query("clinic_name")
	method := c.Query("method")
	status := c.Query("status")
	sortBy := c.Query("sort_by")
	sortOrder := c.Query("sort_order")

	// Validate that both sort_by and sort_order are provided together
	if (sortBy != "" && sortOrder == "") || (sortBy == "" && sortOrder != "") {
		response.Error(c, http.StatusBadRequest, errors.New("both sort_by and sort_order must be provided together"))
		return
	}

	var filter Filter

	// Parse clinic_id if provided
	if clinicIdStr != "" {
		clinicId, err := util.ParseUUID(clinicIdStr)
		if err != nil {
			response.Error(c, http.StatusBadRequest, errors.New("invalid clinic_id format"))
			return
		}
		filter.ClinicID = &clinicId
	}
	// If clinic_id not provided, filter.ClinicID remains nil

	if clinicName != "" {
		filter.ClinicName = &clinicName
	}
	if method != "" {
		filter.Method = &method
	}
	if status != "" {
		filter.Status = &status
	}
	if sortBy != "" && sortOrder != "" {
		filter.SortBy = &sortBy
		filter.SortOrder = &sortOrder
	}

	if err := util.BindAndValidate(c, &filter); err != nil {
		fmt.Println(err.Error())
		response.Error(c, http.StatusBadRequest, err)
		return
	}

	list, err := h.svc.List(c.Request.Context(), filter, practitionerID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, err)
		return
	}
	response.JSON(c, http.StatusOK, list, "Forms fetched successfully")
}

// @Summary Delete form
// @Description Remove a form by its ID
// @Tags form
// @Accept json
// @Produce json
// @Param id path string true "Form ID"
// @Success 204 "No Content"
// @Failure 404 {object} response.RsError
// @Failure 500 {object} response.RsError
// @Router /form/{id} [delete]
func (h *handler) Delete(c *gin.Context) {
	formID, ok := util.ParseUuidID(c, "id")
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), formID); err != nil {
		if errors.Is(err, detail.ErrNotFound) {
			response.Error(c, http.StatusNotFound, err)
			return
		}
		response.Error(c, http.StatusInternalServerError, err)
		return
	}
	response.JSON(c, http.StatusNoContent, nil, "Form deleted successfully")
}
