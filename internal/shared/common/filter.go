package common

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Operator defines possible filtering operations for building SQL conditions.
type Operator string

const (
	OpEq   Operator = "eq"
	OpLike Operator = "like"
	OpIn   Operator = "in"
	OpGt   Operator = "gt"
	OpLt   Operator = "lt"
)

// Condition represents a single filtering condition for a query.
type Condition struct {
	Field    string
	Operator Operator
	Value    interface{}
}

// Filter aggregates filtering, sorting, and pagination options for building queries.
type Filter struct {
	Search  *string
	Where   []Condition
	Limit   int
	Offset  int
	SortBy  string
	OrderBy string
}

// BuildQuery constructs a SQL query and arguments slice based on the provided Filter, allowedColumns, and searchCols.
// - allowedColumns: maps field names to actual DB columns, ensuring only permitted columns are accessible.
// - searchCols: columns to apply the search term to (using ILIKE).
// - count: if true, generate COUNT query.
func BuildQuery(base string, f Filter, allowedColumns map[string]string, searchCols []string, count bool) (string, []interface{}) {
	var (
		args       []interface{}
		conditions []string
	)

	// Build WHERE conditions from Filter.Where
	for _, cond := range f.Where {
		col, ok := allowedColumns[cond.Field]
		if !ok {
			continue
		}

		switch cond.Operator {
		case OpEq:
			conditions = append(conditions, fmt.Sprintf("%s = ?", col))
			args = append(args, cond.Value)
		case OpLike:
			conditions = append(conditions, fmt.Sprintf("%s LIKE ?", col))
			args = append(args, cond.Value)
		case OpGt:
			conditions = append(conditions, fmt.Sprintf("%s > ?", col))
			args = append(args, cond.Value)
		case OpLt:
			conditions = append(conditions, fmt.Sprintf("%s < ?", col))
			args = append(args, cond.Value)
		case OpIn:
			query, inArgs, _ := sqlx.In(fmt.Sprintf("%s IN (?)", col), cond.Value)
			conditions = append(conditions, query)
			args = append(args, inArgs...)
		}
	}

	// Search filter: build OR of LIKE on all searchable columns with the search term.
	if f.Search != nil && *f.Search != "" && len(searchCols) > 0 {
		searchPhrase := "%" + *f.Search + "%"
		var searchParts []string
		for _, col := range searchCols {
			searchParts = append(searchParts, fmt.Sprintf("%s ILIKE ?", col))
			args = append(args, searchPhrase)
		}
		conditions = append(conditions, "("+strings.Join(searchParts, " OR ")+")")
	}

	// Apply WHERE clause if there are any conditions
	if len(conditions) > 0 {
		if strings.Contains(strings.ToUpper(base), "WHERE") {
			base += " AND " + strings.Join(conditions, " AND ")
		} else {
			base += " WHERE " + strings.Join(conditions, " AND ")
		}
	}

	// COUNT query mode
	if count {
		return "SELECT COUNT(*) " + base, args
	}

	query := base

	// Append ORDER BY if requested and field is allowed
	if f.SortBy != "" {
		if col, ok := allowedColumns[f.SortBy]; ok {
			order := "ASC"
			if strings.ToUpper(f.OrderBy) == "DESC" {
				order = "DESC"
			}
			query += fmt.Sprintf(" ORDER BY %s %s", col, order)
		}
	}

	// LIMIT and OFFSET for pagination
	if f.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, f.Limit)
	}
	if f.Offset > 0 {
		query += " OFFSET ?"
		args = append(args, f.Offset)
	}

	return query, args
}

// NewFilter creates a Filter struct from the given criteria.
// - search: pointer to optional search term
// - filters: map of field to value, value may be scalar or a list for "IN" queries
// - operators: optional map of custom operators per field
// - limit/offset: optional pointers to limit and offset for pagination
func NewFilter(
	search *string,
	filters map[string]interface{},
	operators map[string]Operator,
	limit, offset *int,
) Filter {
	var where []Condition

	for field, value := range filters {
		if value == nil {
			continue
		}

		// Skip empty slices to avoid building invalid IN () queries.
		switch v := value.(type) {
		case []string:
			if len(v) == 0 {
				continue
			}
		case []int:
			if len(v) == 0 {
				continue
			}
		case []uuid.UUID:
			if len(v) == 0 {
				continue
			}
		}

		// Determine operator for this field: custom operator or inferred from value type.
		op := OpEq
		if operators != nil {
			if customOp, ok := operators[field]; ok {
				op = customOp
			}
		}
		if op == OpEq {
			switch value.(type) {
			case []string, []int, []int64, []uuid.UUID:
				op = OpIn
			}
		}

		where = append(where, Condition{
			Field:    field,
			Operator: op,
			Value:    value,
		})
	}

	// Safe pagination defaults: 10 limit, 0 offset, with range checks.
	l, o := 10, 0
	if limit != nil && *limit > 0 && *limit <= 100 {
		l = *limit
	}
	if offset != nil && *offset >= 0 {
		o = *offset
	}

	return Filter{
		Search:  search,
		Where:   where,
		Limit:   l,
		Offset:  o,
		SortBy:  "",
		OrderBy: "",
	}
}
