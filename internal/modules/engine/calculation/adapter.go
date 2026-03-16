package calculation

import (
	"github.com/google/uuid"
	builderDetail "github.com/iamarpitzala/acareca/internal/modules/builder/detail"
	builderEntry "github.com/iamarpitzala/acareca/internal/modules/builder/entry"
	builderField "github.com/iamarpitzala/acareca/internal/modules/builder/field"
	"github.com/iamarpitzala/acareca/internal/modules/engine/method"
)

type formFieldIndex map[uuid.UUID]*builderField.FormField

func makeFormFieldIndex(fields []*builderField.FormField) formFieldIndex {
	idx := make(formFieldIndex, len(fields))
	for _, f := range fields {
		if f == nil {
			continue
		}
		idx[f.ID] = f
	}
	return idx
}

func mapTaxTypeFromField(t string) method.TaxTreatment {
	switch t {
	case builderField.TaxTypeInclusive:
		return method.TaxTreatmentInclusive
	case builderField.TaxTypeExclusive:
		return method.TaxTreatmentExclusive
	case builderField.TaxTypeManual:
		return method.TaxTreatmentManual
	default:
		return method.TaxTreatmentZero
	}
}

func mapPaidByFromField(p string) *PaidBy {
	switch p {
	case builderField.PaymentResponsibilityClinic:
		v := PaidByClinic
		return &v
	case builderField.PaymentResponsibilityOwner:
		v := PaidByOwner
		return &v
	default:
		return nil
	}
}

// BuildEntryFromForm composes a calculation Entry using form metadata,
// form fields, and stored entry values.
func BuildEntryFromForm(form *builderDetail.FormDetail, fields []*builderField.FormField, values []*builderEntry.FormEntryValue) (*Entry, error) {
	idx := makeFormFieldIndex(fields)

	var income, expense, otherCosts []Input

	for _, v := range values {
		if v == nil {
			continue
		}
		f, ok := idx[v.FormFieldID]
		if !ok || v.GrossAmount == nil {
			continue
		}

		in := Input{
			Name:    f.Label,
			Value:   *v.GrossAmount,
			TaxType: mapTaxTypeFromField(*f.TaxType),
			PaidBy:  mapPaidByFromField(*f.PaymentResponsibility),
		}

		// For manual tax type we can pass the explicit GST value.
		if in.TaxType == method.TaxTreatmentManual && v.GstAmount != nil {
			in.TaxValue = v.GstAmount
		}

		switch f.SectionType {
		case builderField.SectionTypeCollection:
			income = append(income, in)
		case builderField.SectionTypeCost:
			expense = append(expense, in)
		case builderField.SectionTypeOtherCost:
			otherCosts = append(otherCosts, in)
		}
	}

	ownerShare := float64(form.OwnerShare)
	clinicShare := float64(form.ClinicShare)

	entry := &Entry{
		OwnerShare:  &ownerShare,
		ClinicShare: &clinicShare,
		Income:      income,
		Expense:     expense,
		OtherCosts:  otherCosts,
	}
	return entry, nil
}
