package detail

import (
	"github.com/google/uuid"
)

const (
	StatusDraft     = "DRAFT"
	StatusPublished = "PUBLISHED"
	StatusArchived  = "ARCHIVED"
)

// RqFormFieldItem is a single field in create/update-with-fields payloads. Id optional: omit = create, set = update.
type RqFormFieldItem struct {
	ID                    *uuid.UUID `json:"id,omitempty"`
	Label                 string     `json:"label" validate:"required,max=255"`
	SectionType           string     `json:"section_type" validate:"required,oneof=COLLECTION COST OTHER_COST"`
	PaymentResponsibility string     `json:"payment_responsibility" validate:"required,oneof=OWNER CLINIC"`
	TaxType               string     `json:"tax_type" validate:"required,oneof=INCLUSIVE EXCLUSIVE MANUAL"`
	CoaID                 string     `json:"coa_id" validate:"required,uuid"`
	SortOrder             *int       `json:"sort_order" validate:"omitempty,min=0"`
}

// RqCreateFormWithFields creates a form (default DRAFT), version 1, and the given fields in one request.
type RqCreateFormWithFields struct {
	Name        string  `json:"name" validate:"required"`
	Description *string `json:"description" validate:"omitempty"`
	Status      string  `json:"status" validate:"omitempty,oneof=DRAFT PUBLISHED ARCHIVED"`
	Method      string  `json:"method" validate:"required,oneof=INDEPENDENT_CONTRACTOR SERVICE_FEE"`
	OwnerShare  int     `json:"owner_share" validate:"required,min=0,max=100"`
	ClinicShare int     `json:"clinic_share" validate:"required,min=0,max=100"`
	Fields      []RqFormFieldItem `json:"fields" validate:"omitempty,dive"`
}

// RqUpdateFormWithFields updates form metadata and syncs fields for the active version (DRAFT only). Id in field = update, no id = create, missing ids = delete.
type RqUpdateFormWithFields struct {
	Name        *string `json:"name" validate:"omitempty"`
	Description *string `json:"description" validate:"omitempty"`
	Status      *string `json:"status" validate:"omitempty,oneof=DRAFT PUBLISHED ARCHIVED"`
	Method      *string `json:"method" validate:"omitempty,oneof=INDEPENDENT_CONTRACTOR SERVICE_FEE"`
	OwnerShare  *int    `json:"owner_share" validate:"omitempty,min=0,max=100"`
	ClinicShare *int    `json:"clinic_share" validate:"omitempty,min=0,max=100"`
	Fields      []RqFormFieldItem `json:"fields" validate:"omitempty,dive"`
}

type RqFormDetail struct {
	Name        string  `json:"name" validate:"required"`
	Description *string `json:"description" validate:"omitempty"`
	Status      string  `json:"status" validate:"required,oneof=DRAFT PUBLISHED ARCHIVED"`
	Method      string  `json:"method" validate:"required,oneof=INDEPENDENT_CONTRACTOR SERVICE_FEE"`
	OwnerShare  int     `json:"owner_share" validate:"required,min=0,max=100"`
	ClinicShare int     `json:"clinic_share" validate:"required,min=0,max=100"`
}

type RqUpdateFormDetail struct {
	ID          uuid.UUID `json:"id" validate:"required"`
	Name        *string   `json:"name" validate:"omitempty"`
	Description *string   `json:"description" validate:"omitempty"`
	Status      *string   `json:"status" validate:"omitempty,oneof=DRAFT PUBLISHED ARCHIVED"`
	Method      *string   `json:"method" validate:"omitempty,oneof=INDEPENDENT_CONTRACTOR SERVICE_FEE"`
	OwnerShare  *int      `json:"owner_share" validate:"omitempty,min=0,max=100"`
	ClinicShare *int      `json:"clinic_share" validate:"omitempty,min=0,max=100"`
}

type FormDetail struct {
	ID          uuid.UUID `db:"id" json:"id"`
	ClinicID    uuid.UUID `db:"clinic_id" json:"clinic_id"`
	Name        string    `db:"name" json:"name"`
	Description *string   `db:"description" json:"description,omitempty"`
	Status      string    `db:"status" json:"status"`
	Method      string    `db:"method" json:"method"`
	OwnerShare  int       `db:"owner_share" json:"owner_share"`
	ClinicShare int       `db:"clinic_share" json:"clinic_share"`
	CreatedAt   string    `db:"created_at" json:"created_at"`
	UpdatedAt   string    `db:"updated_at" json:"updated_at"`
}

func (r *RqFormDetail) ToDB(clinicID uuid.UUID) *FormDetail {
	return &FormDetail{
		ID:          uuid.New(),
		ClinicID:    clinicID,
		Name:        r.Name,
		Description: r.Description,
		Status:      r.Status,
		Method:      r.Method,
		OwnerShare:  r.OwnerShare,
		ClinicShare: r.ClinicShare,
	}
}

func (r *RqUpdateFormDetail) Update() *FormDetail {
	d := &FormDetail{ID: r.ID}
	if r.Name != nil {
		d.Name = *r.Name
	}
	if r.Description != nil {
		d.Description = r.Description
	}
	if r.Status != nil {
		d.Status = *r.Status
	}
	if r.Method != nil {
		d.Method = *r.Method
	}
	if r.OwnerShare != nil {
		d.OwnerShare = *r.OwnerShare
	}
	if r.ClinicShare != nil {
		d.ClinicShare = *r.ClinicShare
	}
	return d
}

func (d *FormDetail) ToRs() *RsFormDetail {
	return &RsFormDetail{
		ID:          d.ID,
		ClinicID:    d.ClinicID,
		Name:        d.Name,
		Description: d.Description,
		Status:      d.Status,
		Method:      d.Method,
		OwnerShare:  d.OwnerShare,
		ClinicShare: d.ClinicShare,
		CreatedAt:   d.CreatedAt,
		UpdatedAt:   d.UpdatedAt,
	}
}

type RsFormDetail struct {
	ID          uuid.UUID `json:"id"`
	ClinicID    uuid.UUID `json:"clinic_id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	Status      string    `json:"status"`
	Method      string    `json:"method"`
	OwnerShare  int       `json:"owner_share"`
	ClinicShare int       `json:"clinic_share"`
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
}

// RsFormWithFieldsSyncResult is the result of syncing fields in create/update-with-fields.
type RsFormWithFieldsSyncResult struct {
	CreatedCount int         `json:"created_count"`
	UpdatedCount int         `json:"updated_count"`
	DeletedCount int         `json:"deleted_count"`
	DeletedIDs   []uuid.UUID `json:"deleted_ids,omitempty"`
}

type Filter struct {
	Status   *string   `form:"status" validate:"omitempty,oneof=DRAFT PUBLISHED ARCHIVED"`
	Method   *string   `form:"method" validate:"omitempty,oneof=INDEPENDENT_CONTRACTOR SERVICE_FEE"`
	ClinicID uuid.UUID `form:"clinic_id" validate:"required"`
}
