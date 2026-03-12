package form

import (
	"github.com/google/uuid"
	"github.com/iamarpitzala/acareca/internal/modules/builder/field"
)

type RqBulkSyncFields struct {
	Create []field.RqFormField           `json:"create" validate:"omitempty,dive"`
	Update []field.RqFormFieldUpdateItem `json:"update" validate:"omitempty,dive"`
	Delete []uuid.UUID                   `json:"delete" validate:"omitempty,dive"`
}

type RsBulkSyncFields struct {
	Created []field.RsFormField `json:"created"`
	Updated []field.RsFormField `json:"updated"`
	Deleted []uuid.UUID         `json:"deleted"`
}

type RsFormWithFieldsSyncResult struct {
	CreatedCount int         `json:"created_count"`
	UpdatedCount int         `json:"updated_count"`
	DeletedCount int         `json:"deleted_count"`
	DeletedIDs   []uuid.UUID `json:"deleted_ids,omitempty"`
}

type RqCreateFormWithFields struct {
	Name        string            `json:"name" validate:"required"`
	Description *string           `json:"description" validate:"omitempty"`
	Status      string            `json:"status" validate:"omitempty,oneof=DRAFT PUBLISHED ARCHIVED"`
	Method      string            `json:"method" validate:"required,oneof=INDEPENDENT_CONTRACTOR SERVICE_FEE"`
	OwnerShare  int               `json:"owner_share" validate:"required,min=0,max=100"`
	ClinicShare int               `json:"clinic_share" validate:"required,min=0,max=100"`
	Fields      []RqFormFieldItem `json:"fields" validate:"omitempty,dive"`
}

type RqUpdateFormWithFields struct {
	ID          uuid.UUID         `json:"id" validate:"required"`
	Name        *string           `json:"name" validate:"omitempty"`
	Description *string           `json:"description" validate:"omitempty"`
	Status      *string           `json:"status" validate:"omitempty,oneof=DRAFT PUBLISHED ARCHIVED"`
	Method      *string           `json:"method" validate:"omitempty,oneof=INDEPENDENT_CONTRACTOR SERVICE_FEE"`
	OwnerShare  *int              `json:"owner_share" validate:"omitempty,min=0,max=100"`
	ClinicShare *int              `json:"clinic_share" validate:"omitempty,min=0,max=100"`
	Fields      []RqFormFieldItem `json:"fields" validate:"omitempty,dive"`
}

type RqFormFieldItem struct {
	ID                    *uuid.UUID `json:"id,omitempty"`
	Label                 string     `json:"label" validate:"required,max=255"`
	SectionType           string     `json:"section_type" validate:"required,oneof=COLLECTION COST OTHER_COST"`
	PaymentResponsibility string     `json:"payment_responsibility" validate:"required,oneof=OWNER CLINIC"`
	TaxType               string     `json:"tax_type" validate:"required,oneof=INCLUSIVE EXCLUSIVE MANUAL"`
	CoaID                 string     `json:"coa_id" validate:"required,uuid"`
	SortOrder             *int       `json:"sort_order" validate:"omitempty,min=0"`
}
