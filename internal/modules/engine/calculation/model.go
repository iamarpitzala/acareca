package calculation

import "github.com/iamarpitzala/acareca/internal/modules/engine/method"

type Input struct {
	Name     string              `json:"name" validate:"omitempty,max=255"`
	Value    float64             `json:"value" validate:"omitempty,min=0"`
	TaxType  method.TaxTreatment `json:"tax" validate:"omitempty,oneof=inclusive exclusive manual zero"`
	TaxValue *float64            `json:"tax_value" validate:"omitempty"`
}

type Entry struct {
	Income  []Input `json:"income" validate:"omitempty"`
	Expense []Input `json:"expense" validate:"omitempty"`
}

type Result struct {
	Income  []float64 `json:"income"`
	Expense []float64 `json:"expense"`
	Result  float64   `json:"result"`
}

type GrossResult struct {
	NetResult       float64 `json:"net_result"`
	ServiceFee      float64 `json:"service_fee"`
	GstServiceFee   float64 `json:"gst_service_fee"`
	TotalServiceFee float64 `json:"total_service_fee"`

	RemittedAmount float64 `json:"remitted_amount"`
}
