package calculation

import (
	"context"
	"fmt"

	"github.com/iamarpitzala/acareca/internal/modules/engine/method"
)

type Service interface {
	Calculate(ctx context.Context, entry *Entry) (*Result, error)
	TaxCalculate(ctx context.Context, taxType method.TaxTreatment, input *Input) (*method.Result, error)
}

type service struct {
	repo   Repository
	method method.IService
}

func NewService(repo Repository, method method.IService) Service {
	return &service{repo: repo, method: method}
}

func (s *service) TaxCalculate(ctx context.Context, taxType method.TaxTreatment, input *Input) (*method.Result, error) {
	return s.method.Calculate(ctx, taxType, &method.Input{
		Amount:    input.Value,
		GstAmount: input.TaxValue,
	})
}

func (s *service) Calculate(ctx context.Context, entry *Entry) (*Result, error) {
	var incomeTotals []float64
	var expenseTotals []float64
	var netResult float64

	// Calculate total for each income entry
	for _, inc := range entry.Income {
		res, err := s.TaxCalculate(ctx, inc.TaxType, &inc)
		if err != nil {
			return nil, fmt.Errorf("calculate income: %w", err)
		}
		incomeTotals = append(incomeTotals, res.TotalAmount)
		netResult += res.TotalAmount
	}

	// Calculate total for each expense entry
	for _, exp := range entry.Expense {
		res, err := s.TaxCalculate(ctx, exp.TaxType, &exp)
		if err != nil {
			return nil, fmt.Errorf("calculate expense: %w", err)
		}
		expenseTotals = append(expenseTotals, res.TotalAmount)
		netResult -= res.TotalAmount
	}

	return &Result{
		Income:  incomeTotals,
		Expense: expenseTotals,
		Result:  netResult,
	}, nil
}
