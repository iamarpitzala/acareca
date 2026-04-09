package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// buildTimeseriesQuery creates a reusable timeseries query
func buildTimeseriesQuery(table, groupField, dateField string, entityFilter string) string {
	return fmt.Sprintf(`
		SELECT 
			%s,
			DATE_TRUNC($1, %s) as ts,
			COUNT(*) as count
		FROM %s
		WHERE %s BETWEEN $2 AND $3
			%s
		GROUP BY %s, DATE_TRUNC($1, %s)
		ORDER BY %s, ts
	`, groupField, dateField, table, dateField, entityFilter, groupField, dateField, groupField)
}

// scanTimeseries scans timeseries data into a map
func scanTimeseries(rows *sqlx.Rows, dateFormat string) (map[string][]TimePoint, error) {
	seriesMap := make(map[string][]TimePoint)
	
	for rows.Next() {
		var key string
		var ts time.Time
		var count int
		if err := rows.Scan(&key, &ts, &count); err != nil {
			return nil, err
		}

		seriesMap[key] = append(seriesMap[key], TimePoint{
			Timestamp: ts.Format(dateFormat),
			Count:     count,
		})
	}
	
	return seriesMap, nil
}

// buildRevenueQuery creates a reusable revenue query
func buildRevenueQuery() string {
	return `
		SELECT 
			DATE_TRUNC($1, ps.created_at) as ts,
			SUM(s.price) as revenue
		FROM tbl_practitioner_subscription ps
		JOIN tbl_subscription s ON ps.subscription_id = s.id
		WHERE ps.created_at BETWEEN $2 AND $3
			AND ps.deleted_at IS NULL
		GROUP BY DATE_TRUNC($1, ps.created_at)
		ORDER BY ts
	`
}

// scanRevenue scans revenue data
func scanRevenue(rows *sqlx.Rows, dateFormat string) ([]RevenuePoint, error) {
	var points []RevenuePoint
	
	for rows.Next() {
		var ts time.Time
		var revenue float64
		if err := rows.Scan(&ts, &revenue); err != nil {
			return nil, err
		}
		points = append(points, RevenuePoint{
			Timestamp: ts.Format(dateFormat),
			Revenue:   revenue,
		})
	}
	
	return points, nil
}

// executeCountQuery executes a count query and returns the result
func executeCountQuery(ctx context.Context, db *sqlx.DB, query string, args ...interface{}) (int, error) {
	var count int
	err := db.GetContext(ctx, &count, query, args...)
	return count, err
}

// buildFilterConditions builds WHERE conditions from filter
func buildFilterConditions(filter *SubscriptionRecordFilter) ([]string, []interface{}, int) {
	var conditions []string
	var args []interface{}
	argCount := 1

	if filter == nil {
		return conditions, args, argCount
	}

	if filter.Search != nil && *filter.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(u.email ILIKE $%d OR CONCAT(u.first_name, ' ', u.last_name) ILIKE $%d)", argCount, argCount))
		args = append(args, "%"+*filter.Search+"%")
		argCount++
	}
	if filter.PlanName != nil && *filter.PlanName != "" {
		conditions = append(conditions, fmt.Sprintf("s.name ILIKE $%d", argCount))
		args = append(args, "%"+*filter.PlanName+"%")
		argCount++
	}
	if filter.Status != nil && *filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("ps.status = $%d", argCount))
		args = append(args, *filter.Status)
		argCount++
	}
	if filter.From != nil && *filter.From != "" {
		conditions = append(conditions, fmt.Sprintf("ps.created_at >= $%d", argCount))
		args = append(args, *filter.From)
		argCount++
	}
	if filter.To != nil && *filter.To != "" {
		conditions = append(conditions, fmt.Sprintf("ps.created_at <= $%d", argCount))
		args = append(args, *filter.To)
		argCount++
	}

	return conditions, args, argCount
}
