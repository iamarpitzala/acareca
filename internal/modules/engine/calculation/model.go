package calculation

import "github.com/iamarpitzala/acareca/internal/modules/engine/method"

type Input struct {
	Name     string              `json:"name" validate:"omitempty,max=255"`
	Value    float64             `json:"value" validate:"omitempty,min=0"`
	TaxType  method.TaxTreatment `json:"tax" validate:"omitempty,oneof=inclusive exclusive manual"`
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
