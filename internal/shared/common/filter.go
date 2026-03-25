package common

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// QueryFilter is a reusable struct for binding common list query params.
// Embed this in module-level filter structs or use ParseQueryFilter directly.
type QueryFilter struct {
	Search  *string `form:"search"`
	SortBy  *string `form:"sort_by"`
	OrderBy *string `form:"order_by"`
	Limit   *int    `form:"limit"`
	Page    *int    `form:"page"` // 1-based page number
}

// ParseQueryFilter converts QueryFilter into a common.Filter.
// fields: map of field -> value (nil values are skipped).
// operators: optional custom operators per field.
// defaultSort: column name to sort by when SortBy is not provided.
func ParseQueryFilter(q QueryFilter, fields map[string]interface{}, operators map[string]Operator, defaultSort string) Filter {
	// Convert page to offset
	var offsetPtr *int
	if q.Page != nil && *q.Page > 1 {
		l := 10
		if q.Limit != nil && *q.Limit > 0 {
			l = *q.Limit
		}
		offset := (*q.Page - 1) * l
		offsetPtr = &offset
	}

	f := NewFilter(q.Search, fields, operators, q.Limit, offsetPtr)

	if q.SortBy != nil && *q.SortBy != "" {
		f.SortBy = *q.SortBy
	} else if defaultSort != "" {
		f.SortBy = defaultSort
	}

	if q.OrderBy != nil && *q.OrderBy != "" {
		f.OrderBy = *q.OrderBy
	} else {
		f.OrderBy = "DESC"
	}

	return f
}

type Operator string

const (
	OpEq   Operator = "eq"
	OpLike Operator = "like"
	OpIn   Operator = "in"
	OpGt   Operator = "gt"
	OpLt   Operator = "lt"
)

type Condition struct {
	Field    string
	Operator Operator
	Value    interface{}
}

type Filter struct {
	Search  *string
	Where   []Condition
	Limit   int
	Offset  int
	SortBy  string
	OrderBy string
}

func BuildQuery(base string, f Filter, allowedColumns map[string]string, searchCols []string, count bool) (string, []interface{}) {

	var args []interface{}
	var conditions []string

	for _, c := range f.Where {
		col, ok := allowedColumns[c.Field]
		if !ok {
			continue
		}

		switch c.Operator {
		case OpEq:
			conditions = append(conditions, fmt.Sprintf("%s = ?", col))
			args = append(args, c.Value)

		case OpLike:
			conditions = append(conditions, fmt.Sprintf("%s LIKE ?", col))
			args = append(args, c.Value)

		case OpGt:
			conditions = append(conditions, fmt.Sprintf("%s > ?", col))
			args = append(args, c.Value)

		case OpLt:
			conditions = append(conditions, fmt.Sprintf("%s < ?", col))
			args = append(args, c.Value)

		case OpIn:
			query, inArgs, _ := sqlx.In(fmt.Sprintf("%s IN (?)", col), c.Value)
			conditions = append(conditions, query)
			args = append(args, inArgs...)
		}
	}

	// 🔥 Search
	if f.Search != nil && *f.Search != "" && len(searchCols) > 0 {
		var searchParts []string

		for _, col := range searchCols {
			searchParts = append(searchParts, fmt.Sprintf("%s ILIKE ?", col))
			args = append(args, "%"+*f.Search+"%")
		}

		conditions = append(conditions, "("+strings.Join(searchParts, " OR ")+")")
	}

	// 🔥 Apply WHERE
	if len(conditions) > 0 {
		if strings.Contains(strings.ToUpper(base), "WHERE") {
			base += " AND " + strings.Join(conditions, " AND ")
		} else {
			base += " WHERE " + strings.Join(conditions, " AND ")
		}
	}

	// 🔥 COUNT MODE
	if count {
		query := "SELECT COUNT(*) " + base
		return query, args
	}

	// 🔥 NORMAL MODE
	query := base

	// Sorting
	if col, ok := allowedColumns[f.SortBy]; ok {
		order := "ASC"
		if strings.ToUpper(f.OrderBy) == "DESC" {
			order = "DESC"
		}
		query += fmt.Sprintf(" ORDER BY %s %s", col, order)
	}

	// Pagination
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

func NewFilter(search *string, filters map[string]interface{}, operators map[string]Operator, limit, offset *int) Filter {

	var where []Condition

	for field, value := range filters {

		if value == nil {
			continue
		}

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

	// 🔥 safe defaults
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
