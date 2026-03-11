package coa

import (
	"context"
	"log"

	"github.com/google/uuid"
)

// DefaultChartRow defines one default chart-of-account row (e.g. Xero-style defaults).
type DefaultChartRow struct {
	Code          int16  // 3–4 digit code (100–9999)
	Name          string
	AccountTypeID int16  // 1=Asset, 2=Liability, 3=Equity, 4=Revenue, 5=Expense
	AccountTaxID  int16  // 1=GST on Income, 2=GST on Expenses, etc.
}

// DefaultChartOfAccounts returns the 4–5 default accounts created for each practitioner.
func DefaultChartOfAccounts() []DefaultChartRow {
	return []DefaultChartRow{
		{Code: 1000, Name: "Bank", AccountTypeID: 1, AccountTaxID: 1},
		{Code: 2000, Name: "Accounts Receivable", AccountTypeID: 1, AccountTaxID: 1},
		{Code: 3000, Name: "Equity", AccountTypeID: 3, AccountTaxID: 3},
		{Code: 4000, Name: "Revenue", AccountTypeID: 4, AccountTaxID: 1},
		{Code: 5000, Name: "Expenses", AccountTypeID: 5, AccountTaxID: 2},
	}
}

// SeedDefaultsForPractitioner creates default chart-of-account rows for a practitioner.
// practice_id = practitionerID, is_system = true.
func SeedDefaultsForPractitioner(ctx context.Context, repo Repository, practitionerID uuid.UUID) error {
	for _, row := range DefaultChartOfAccounts() {
		chart := &ChartOfAccount{
			CreatedBy:     practitionerID,
			AccountTypeID: row.AccountTypeID,
			AccountTaxID:  row.AccountTaxID,
			Code:          row.Code,
			Name:          row.Name,
			IsSystem:      true,
		}
		_, err := repo.CreateChart(ctx, chart)
		if err != nil {
			log.Printf("coa: seed default %d for practitioner %s: %v", row.Code, practitionerID, err)
			return err
		}
	}
	return nil
}
