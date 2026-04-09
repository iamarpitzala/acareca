package analytics

import "time"

// parseDateRange extracts and validates date range from filter
func parseDateRange(from, to *string, defaultDays int) (string, string) {
	endDate := time.Now().Format("2006-01-02")
	startDate := time.Now().AddDate(0, 0, -defaultDays).Format("2006-01-02")

	if from != nil && *from != "" {
		startDate = *from
	}
	if to != nil && *to != "" {
		endDate = *to
	}

	return startDate, endDate
}

// parseBucket extracts bucket value with default
func parseBucket(bucket *string, defaultBucket string) string {
	if bucket != nil && *bucket != "" {
		return *bucket
	}
	return defaultBucket
}

// getBucketConfig returns date truncation and format based on bucket type
func getBucketConfig(bucket string) (dateTrunc, dateFormat string) {
	switch bucket {
	case "day":
		return "day", "2006-01-02"
	case "week":
		return "week", "2006-01-02"
	case "month":
		return "month", "2006-01"
	default:
		return "month", "2006-01"
	}
}

// parsePaginationParams extracts pagination parameters with defaults
func parsePaginationParams(limit, offset *int) (int, int) {
	l, o := 20, 0
	if limit != nil && *limit > 0 && *limit <= 100 {
		l = *limit
	}
	if offset != nil && *offset >= 0 {
		o = *offset
	}
	return l, o
}

// parseSortParams extracts sort parameters with defaults
func parseSortParams(sortBy, orderBy *string, defaultSort, defaultOrder string) (string, string) {
	sort := defaultSort
	order := defaultOrder

	if sortBy != nil && *sortBy != "" {
		sort = *sortBy
	}
	if orderBy != nil && *orderBy != "" {
		order = *orderBy
	}

	return sort, order
}
