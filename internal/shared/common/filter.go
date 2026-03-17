package common

import (
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
)

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
