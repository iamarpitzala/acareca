package orchestrate

import (
	"context"

	"github.com/google/uuid"
	"github.com/iamarpitzala/acareca/internal/modules/form/detail"
	"github.com/iamarpitzala/acareca/internal/modules/form/field"
	"github.com/iamarpitzala/acareca/internal/modules/form/version"
)

type Orchestrator struct {
	detailSvc  detail.IService
	versionSvc  version.IService
	fieldSvc   field.IService
}

func NewOrchestrator(detailSvc detail.IService, versionSvc version.IService, fieldSvc field.IService) *Orchestrator {
	return &Orchestrator{
		detailSvc: detailSvc,
		versionSvc: versionSvc,
		fieldSvc:  fieldSvc,
	}
}

// CreateWithFields creates form (DRAFT), version 1, and fields in one call.
func (o *Orchestrator) CreateWithFields(ctx context.Context, clinicID uuid.UUID, practitionerID uuid.UUID, req *detail.RqCreateFormWithFields) (*detail.RsFormDetail, *detail.RsFormWithFieldsSyncResult, error) {
	formReq := &detail.RqFormDetail{
		Name:        req.Name,
		Description: req.Description,
		Status:      req.Status,
		Method:      req.Method,
		OwnerShare:  req.OwnerShare,
		ClinicShare: req.ClinicShare,
	}
	if formReq.Status == "" {
		formReq.Status = detail.StatusDraft
	}
	created, err := o.detailSvc.Create(ctx, formReq, clinicID, practitionerID)
	if err != nil {
		return nil, nil, err
	}
	syncResult := &detail.RsFormWithFieldsSyncResult{}
	if len(req.Fields) == 0 {
		return created, syncResult, nil
	}
	versions, err := o.versionSvc.List(ctx, created.ID, clinicID)
	if err != nil {
		return nil, nil, err
	}
	var activeVersionID uuid.UUID
	for _, v := range versions {
		if v.IsActive {
			activeVersionID = v.Id
			break
		}
	}
	if activeVersionID == uuid.Nil {
		return created, syncResult, nil
	}
	createList := make([]field.RqFormField, 0, len(req.Fields))
	for i := range req.Fields {
		f := &req.Fields[i]
		r := field.RqFormField{
			Label:                 f.Label,
			SectionType:           f.SectionType,
			PaymentResponsibility: f.PaymentResponsibility,
			TaxType:               f.TaxType,
			CoaID:                 f.CoaID,
		}
		if f.SortOrder != nil {
			r.SortOrder = f.SortOrder
		}
		createList = append(createList, r)
	}
	bulk, err := o.fieldSvc.BulkSyncFields(ctx, activeVersionID, practitionerID, &field.RqBulkSyncFields{
		Create: createList,
		Update: nil,
		Delete: nil,
	})
	if err != nil {
		return nil, nil, err
	}
	syncResult.CreatedCount = len(bulk.Created)
	syncResult.UpdatedCount = len(bulk.Updated)
	syncResult.DeletedCount = len(bulk.Deleted)
	syncResult.DeletedIDs = bulk.Deleted
	return created, syncResult, nil
}

// UpdateWithFields updates form metadata and syncs fields for the active version when form is DRAFT.
func (o *Orchestrator) UpdateWithFields(ctx context.Context, formID uuid.UUID, clinicID uuid.UUID, practitionerID uuid.UUID, req *detail.RqUpdateFormWithFields) (*detail.RsFormDetail, *detail.RsFormWithFieldsSyncResult, error) {
	existing, err := o.detailSvc.GetByID(ctx, formID)
	if err != nil {
		return nil, nil, err
	}
	syncResult := &detail.RsFormWithFieldsSyncResult{}
	updateReq := &detail.RqUpdateFormDetail{
		ID:          formID,
		Name:        req.Name,
		Description: req.Description,
		Status:      req.Status,
		Method:      req.Method,
		OwnerShare:  req.OwnerShare,
		ClinicShare: req.ClinicShare,
	}
	updated, err := o.detailSvc.UpdateMetadata(ctx, updateReq)
	if err != nil {
		return nil, nil, err
	}
	if existing.Status != detail.StatusDraft && len(req.Fields) > 0 {
		return nil, nil, detail.ErrFormNotDraftForFields
	}
	if len(req.Fields) == 0 {
		return updated, syncResult, nil
	}
	versions, err := o.versionSvc.List(ctx, formID, clinicID)
	if err != nil {
		return nil, nil, err
	}
	var activeVersionID uuid.UUID
	for _, v := range versions {
		if v.IsActive {
			activeVersionID = v.Id
			break
		}
	}
	if activeVersionID == uuid.Nil {
		return updated, syncResult, nil
	}
	currentFields, err := o.fieldSvc.ListByFormVersionID(ctx, activeVersionID)
	if err != nil {
		return nil, nil, err
	}
	currentIDs := make(map[uuid.UUID]struct{})
	for _, f := range currentFields {
		currentIDs[f.ID] = struct{}{}
	}
	keepIDs := make(map[uuid.UUID]struct{})
	var createList []field.RqFormField
	var updateList []field.RqFormFieldUpdateItem
	for i := range req.Fields {
		f := &req.Fields[i]
		if f.ID != nil {
			keepIDs[*f.ID] = struct{}{}
			item := field.RqFormFieldUpdateItem{
				ID:                    *f.ID,
				Label:                 &f.Label,
				SectionType:           &f.SectionType,
				PaymentResponsibility: &f.PaymentResponsibility,
				TaxType:               &f.TaxType,
				CoaID:                 &f.CoaID,
			}
			if f.SortOrder != nil {
				item.SortOrder = f.SortOrder
			}
			updateList = append(updateList, item)
		} else {
			r := field.RqFormField{
				Label:                 f.Label,
				SectionType:           f.SectionType,
				PaymentResponsibility: f.PaymentResponsibility,
				TaxType:               f.TaxType,
				CoaID:                 f.CoaID,
			}
			if f.SortOrder != nil {
				r.SortOrder = f.SortOrder
			}
			createList = append(createList, r)
		}
	}
	var deleteList []uuid.UUID
	for id := range currentIDs {
		if _, keep := keepIDs[id]; !keep {
			deleteList = append(deleteList, id)
		}
	}
	bulk, err := o.fieldSvc.BulkSyncFields(ctx, activeVersionID, practitionerID, &field.RqBulkSyncFields{
		Create: createList,
		Update: updateList,
		Delete: deleteList,
	})
	if err != nil {
		return nil, nil, err
	}
	syncResult.CreatedCount = len(bulk.Created)
	syncResult.UpdatedCount = len(bulk.Updated)
	syncResult.DeletedCount = len(bulk.Deleted)
	syncResult.DeletedIDs = bulk.Deleted
	return updated, syncResult, nil
}
