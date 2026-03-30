package calculation

import (
	"github.com/google/uuid"
	"github.com/iamarpitzala/acareca/internal/modules/builder/entry"
)

type Method string

const (
	IndependentContractor Method = "INDEPENDENT_CONTRACTOR"
	ServiceFee            Method = "SERVICE_FEE"
)

type GrossResult struct {
	NetAmount float64 `json:"net_amount"`

	ServiceFee      float64 `json:"service_fee"`
	GstServiceFee   float64 `json:"gst_service_fee"`
	TotalServiceFee float64 `json:"total_service_fee"`
	RemittedAmount  float64 `json:"remitted_amount"`

	ClinicExpenseGST float64 `json:"clinic_expense_gst"`
}

type NetResult struct {
	NetAmount float64 `json:"net_amount"`

	TotalRemuneration float64 `json:"total_remuneration"`

	BaseRemuneration *float64 `json:"base_remuneration,omitempty"`

	SuperComponent *float64 `json:"super_component,omitempty"`

	GstOnRemuneration float64 `json:"gst_on_remuneration"`

	InvoiceTotal float64 `json:"invoice_total"`

	OtherCostDeduction float64 `json:"other_cost_deduction"`
}

type NetFilter struct {
	SuperComponent *float64 `json:"super_component" validate:"omitempty,min=0,max=100"`
}

type RqCalculateFromEntries struct {
	FormID  string               `json:"form_id" validate:"required,uuid"`
	Entries []entry.RsEntryValue `json:"entries" validate:"required,min=1,dive"`

	SuperComponent *float64 `json:"super_component" validate:"omitempty,min=0,max=100"`
}

// RqFormulaCalculate binds manual field key→amount values from query params.
// e.g. GET /calculate/formula/form_id?A=5000&B=300&C=55&D=20
// Values is populated manually in the handler from c.QueryMap("values").
type RqFormulaCalculate struct {
	Values map[string]float64 // populated from query params in handler
}

// RsComputedFieldValue is the per-field result for a computed field.
type RsComputedFieldValue struct {
	FieldID   uuid.UUID `json:"field_id"`
	FieldKey  string    `json:"field_key"`
	Label     string    `json:"label"`
	Amount    float64   `json:"amount"`               // net amount (ex-GST when tax applies)
	GstAmount *float64  `json:"gst_amount,omitempty"` // only present when field has a tax_type
	Gross     *float64  `json:"gross,omitempty"`      // only present when field has a tax_type
}

// RsFormulaCalculate is the response for POST /calculate/formula/:form_id.
type RsFormulaCalculate struct {
	FormID         uuid.UUID              `json:"form_id"`
	ComputedFields []RsComputedFieldValue `json:"computed_fields"`
}
