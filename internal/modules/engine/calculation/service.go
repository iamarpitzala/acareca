package calculation

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/iamarpitzala/acareca/internal/modules/builder/detail"
	"github.com/iamarpitzala/acareca/internal/modules/builder/entry"
	"github.com/iamarpitzala/acareca/internal/modules/builder/field"
	"github.com/iamarpitzala/acareca/internal/modules/builder/form"
	"github.com/iamarpitzala/acareca/internal/modules/builder/version"
	"github.com/iamarpitzala/acareca/internal/shared/util"
)

type Service interface {
	GrossMethod(ctx context.Context, formDetail *detail.RsFormDetail, formValue []entry.RsEntryValue) (*GrossResult, error)
	NetMethod(ctx context.Context, formDetail *detail.RsFormDetail, formValue []entry.RsEntryValue, filter *NetFilter) (*NetResult, error)
	Calculate(ctx context.Context, formId uuid.UUID, filter *NetFilter) (interface{}, error)

	CalculateFromEntries(ctx context.Context, req *RqCalculateFromEntries) (interface{}, error)
}

type service struct {
	formSvc    form.IService
	versionSvc version.IService
	fieldSvc   field.IService
	entries    entry.IService
}

func NewService(formSvc form.IService, versionSvc version.IService, fieldSvc field.IService, entries entry.IService) Service {
	return &service{formSvc: formSvc, versionSvc: versionSvc, fieldSvc: fieldSvc, entries: entries}
}

func (s *service) GrossMethod(ctx context.Context, formDetail *detail.RsFormDetail, formValue []entry.RsEntryValue) (*GrossResult, error) {
	var (
		incomeSum    float64 // sum of ex-GST collection amounts
		incomeGST    float64 // GST on collections (returned to practitioner for ATO remittance)
		expenseSum   float64 // sum of ex-GST clinic-paid costs
		expenseGST   float64 // GST on clinic-paid costs (claimable as input tax credit)
		otherCostSum float64 // OTHER_COST deductions

		paidByOwnerSum float64 // OWNER-paid costs passed through to practitioner
	)

	for _, v := range formValue {
		field, err := s.fieldSvc.GetByID(ctx, v.FormFieldID)
		if err != nil {
			return nil, err
		}

		switch field.SectionType {

		case "COLLECTION":
			if v.NetAmount != nil {
				incomeSum += *v.NetAmount
			}
			if v.GstAmount != nil {
				incomeGST += *v.GstAmount
			}

		case "COST":
			if field.PaymentResponsibility == nil {
				continue
			}

			switch *field.PaymentResponsibility {

			case "CLINIC":
				if v.NetAmount != nil {
					expenseSum += *v.NetAmount
				}
				if v.GstAmount != nil {
					expenseGST += *v.GstAmount
				}

			case "OWNER":
				if v.GrossAmount != nil {
					expenseSum += *v.GrossAmount
					paidByOwnerSum += *v.GrossAmount
				}
			}

		case "OTHER_COST":
			if v.NetAmount != nil {
				otherCostSum += *v.NetAmount
			}
		}
	}

	netIncome := incomeSum

	netAmount := netIncome - (expenseSum - paidByOwnerSum)

	clinicShare := float64(formDetail.ClinicShare)
	serviceFee := netAmount * (clinicShare / 100)
	gstServiceFee := serviceFee * 0.1
	totalServiceFee := serviceFee + gstServiceFee

	remittedAmount := netAmount - totalServiceFee - otherCostSum + incomeGST + paidByOwnerSum

	return &GrossResult{
		NetAmount:        util.Round(netAmount, 2),
		ServiceFee:       util.Round(serviceFee, 2),
		GstServiceFee:    util.Round(gstServiceFee, 2),
		TotalServiceFee:  util.Round(totalServiceFee, 2),
		RemittedAmount:   util.Round(remittedAmount, 2),
		ClinicExpenseGST: util.Round(expenseGST, 2),
	}, nil
}

func (s *service) NetMethod(ctx context.Context, formDetail *detail.RsFormDetail, formValue []entry.RsEntryValue, filter *NetFilter) (*NetResult, error) {
	var (
		incomeSum    float64
		expenseSum   float64
		otherCostSum float64
	)

	for _, v := range formValue {
		field, err := s.fieldSvc.GetByID(ctx, v.FormFieldID)
		if err != nil {
			return nil, err
		}

		switch field.SectionType {

		case "COLLECTION":
			if v.NetAmount != nil {
				incomeSum += *v.NetAmount
			}

		case "COST":
			if v.NetAmount != nil {
				expenseSum += *v.NetAmount
			}

		case "OTHER_COST":
			if v.NetAmount != nil {
				otherCostSum += *v.NetAmount
			}
		}
	}

	netAmount := incomeSum - expenseSum - otherCostSum

	ownerShare := float64(formDetail.OwnerShare)

	superDecimal := 0.0
	if filter != nil && filter.SuperComponent != nil {
		superDecimal = *filter.SuperComponent / 100
	}

	totalRemuneration := netAmount * (ownerShare / 100)

	commissionBase := totalRemuneration
	var superAmount float64
	if superDecimal > 0 {
		commissionBase = totalRemuneration / (1 + superDecimal)
		superAmount = commissionBase * superDecimal
	}

	gstOnRemuneration := commissionBase * 0.10
	invoiceTotal := commissionBase + gstOnRemuneration

	netResult := NetResult{
		NetAmount:          util.Round(netAmount, 2),
		TotalRemuneration:  util.Round(totalRemuneration, 2),
		GstOnRemuneration:  util.Round(gstOnRemuneration, 2),
		InvoiceTotal:       util.Round(invoiceTotal, 2),
		OtherCostDeduction: util.Round(otherCostSum, 2),
	}

	if superDecimal > 0 {
		sa := util.Round(superAmount, 2)
		netResult.SuperComponent = &sa

		br := util.Round(commissionBase, 2)
		netResult.BaseRemuneration = &br
	}

	return &netResult, nil
}

// Calculate implements [Service].
func (s *service) Calculate(ctx context.Context, formID uuid.UUID, filter *NetFilter) (interface{}, error) {

	form, err := s.formSvc.GetFormByID(ctx, formID)
	if err != nil {
		return nil, err
	}
	version, err := s.versionSvc.GetVersionByFormID(ctx, form.ID)
	if err != nil {
		return nil, err
	}
	entries, err := s.entries.GetByVersionID(ctx, version.Id)
	if err != nil {
		return nil, err
	}
	switch Method(form.Method) {
	case IndependentContractor:
		return s.NetMethod(ctx, form, entries.Values, filter)
	case ServiceFee:
		return s.GrossMethod(ctx, form, entries.Values)
	default:
		return nil, fmt.Errorf("unsupported method: %s", form.Method)
	}
}

// CalculateFromEntries implements [Service].
func (s *service) CalculateFromEntries(ctx context.Context, req *RqCalculateFromEntries) (interface{}, error) {
	formID, err := uuid.Parse(req.FormID)
	if err != nil {
		return nil, fmt.Errorf("invalid form_id: %w", err)
	}

	form, err := s.formSvc.GetFormByID(ctx, formID)
	if err != nil {
		return nil, err
	}

	filter := &NetFilter{SuperComponent: req.SuperComponent}

	switch Method(form.Method) {
	case IndependentContractor:
		return s.NetMethod(ctx, form, req.Entries, filter)
	case ServiceFee:
		return s.GrossMethod(ctx, form, req.Entries)
	default:
		return nil, fmt.Errorf("unsupported method: %s", form.Method)
	}
}
