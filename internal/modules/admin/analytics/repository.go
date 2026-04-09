package analytics

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/iamarpitzala/acareca/internal/shared/common"
	"github.com/jmoiron/sqlx"
)

type Repository interface {
	GetUserGrowth(ctx context.Context, startDate, endDate time.Time) (*RsUserGrowth, error)
	GetSubscriptionMetrics(ctx context.Context) (*RsSubscriptionMetrics, error)
	GetActiveUsers(ctx context.Context, startDate, endDate time.Time) (*RsActiveUsers, error)
	GetPractitionerDetails(ctx context.Context, practitionerID uuid.UUID) (*RsPractitionerDetail, error)
	ListPractitionersWithDetails(ctx context.Context, filter *PractitionerFilter) ([]*RsPractitionerDetail, int, error)
}

type repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) Repository {
	return &repository{db: db}
}

// GetUserGrowth retrieves user growth and retention metrics
func (r *repository) GetUserGrowth(ctx context.Context, startDate, endDate time.Time) (*RsUserGrowth, error) {
	var result RsUserGrowth

	// Total users
	query := `
		SELECT 
			COUNT(*) as total_users,
			COUNT(CASE WHEN role = 'PRACTITIONER' THEN 1 END) as total_practitioners,
			COUNT(CASE WHEN role = 'ACCOUNTANT' THEN 1 END) as total_accountants,
			COUNT(CASE WHEN created_at >= NOW() - INTERVAL '30 days' THEN 1 END) as new_users_30_days
		FROM tbl_user
		WHERE deleted_at IS NULL
	`
	err := r.db.GetContext(ctx, &result, query)
	if err != nil {
		return nil, fmt.Errorf("get user counts: %w", err)
	}

	// Active users (users with audit log entries in last 30 days)
	activeQuery := `
		SELECT COUNT(DISTINCT user_id) as active_users
		FROM tbl_audit_log
		WHERE created_at >= NOW() - INTERVAL '30 days'
	`
	err = r.db.GetContext(ctx, &result.ActiveUsers30Days, activeQuery)
	if err != nil {
		return nil, fmt.Errorf("get active users: %w", err)
	}

	// Calculate growth rate
	if result.TotalUsers > 0 {
		result.GrowthRate = (float64(result.NewUsers30Days) / float64(result.TotalUsers-result.NewUsers30Days)) * 100
	}

	// Calculate retention rate
	if result.TotalUsers > 0 {
		result.RetentionRate = (float64(result.ActiveUsers30Days) / float64(result.TotalUsers)) * 100
	}

	// Timeline data
	timelineQuery := `
		SELECT 
			DATE(created_at) as date,
			COUNT(*) as new_users,
			SUM(COUNT(*)) OVER (ORDER BY DATE(created_at)) as total_users
		FROM tbl_user
		WHERE created_at BETWEEN $1 AND $2
			AND deleted_at IS NULL
		GROUP BY DATE(created_at)
		ORDER BY date
	`
	rows, err := r.db.QueryxContext(ctx, timelineQuery, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("get timeline: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var point UserGrowthPoint
		var date time.Time
		err := rows.Scan(&date, &point.NewUsers, &point.TotalUsers)
		if err != nil {
			return nil, fmt.Errorf("scan timeline: %w", err)
		}
		point.Date = date.Format("2006-01-02")
		result.Timeline = append(result.Timeline, point)
	}

	return &result, nil
}

// GetSubscriptionMetrics retrieves subscription distribution and MRR
func (r *repository) GetSubscriptionMetrics(ctx context.Context) (*RsSubscriptionMetrics, error) {
	var result RsSubscriptionMetrics

	// Total active subscriptions and MRR
	query := `
		SELECT 
			COUNT(*) as total_active,
			SUM(s.price / s.duration_days * 30) as mrr
		FROM tbl_practitioner_subscription ps
		JOIN tbl_subscription s ON ps.subscription_id = s.id
		WHERE ps.status = 'ACTIVE'
			AND ps.deleted_at IS NULL
			AND s.deleted_at IS NULL
	`
	var totalActive int
	var mrr *float64
	err := r.db.QueryRowContext(ctx, query).Scan(&totalActive, &mrr)
	if err != nil {
		return nil, fmt.Errorf("get subscription totals: %w", err)
	}

	result.TotalActiveSubscriptions = totalActive
	if mrr != nil {
		result.MRR = *mrr
		result.ARR = *mrr * 12
	}

	// ARPU
	if totalActive > 0 && mrr != nil {
		result.ARPU = *mrr / float64(totalActive)
	}

	// Churn rate (cancelled in last 30 days / total active 30 days ago)
	churnQuery := `
		WITH active_30_days_ago AS (
			SELECT COUNT(*) as count
			FROM tbl_practitioner_subscription
			WHERE status = 'ACTIVE'
				AND created_at <= NOW() - INTERVAL '30 days'
				AND deleted_at IS NULL
		),
		churned_last_30 AS (
			SELECT COUNT(*) as count
			FROM tbl_practitioner_subscription
			WHERE status IN ('CANCELLED', 'EXPIRED')
				AND updated_at >= NOW() - INTERVAL '30 days'
				AND deleted_at IS NULL
		)
		SELECT 
			CASE WHEN a.count > 0 THEN (c.count::float / a.count::float) * 100 ELSE 0 END as churn_rate
		FROM active_30_days_ago a, churned_last_30 c
	`
	err = r.db.GetContext(ctx, &result.ChurnRate, churnQuery)
	if err != nil {
		return nil, fmt.Errorf("get churn rate: %w", err)
	}

	// Distribution by plan
	distQuery := `
		SELECT 
			s.id as plan_id,
			s.name as plan_name,
			COUNT(*) as count,
			SUM(s.price / s.duration_days * 30) as revenue
		FROM tbl_practitioner_subscription ps
		JOIN tbl_subscription s ON ps.subscription_id = s.id
		WHERE ps.status = 'ACTIVE'
			AND ps.deleted_at IS NULL
			AND s.deleted_at IS NULL
		GROUP BY s.id, s.name
		ORDER BY count DESC
	`
	rows, err := r.db.QueryxContext(ctx, distQuery)
	if err != nil {
		return nil, fmt.Errorf("get distribution: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var dist SubscriptionDistribution
		err := rows.Scan(&dist.PlanID, &dist.PlanName, &dist.Count, &dist.Revenue)
		if err != nil {
			return nil, fmt.Errorf("scan distribution: %w", err)
		}
		if totalActive > 0 {
			dist.Percentage = (float64(dist.Count) / float64(totalActive)) * 100
		}
		result.Distribution = append(result.Distribution, dist)
	}

	return &result, nil
}

// GetActiveUsers retrieves DAU/WAU/MAU metrics
func (r *repository) GetActiveUsers(ctx context.Context, startDate, endDate time.Time) (*RsActiveUsers, error) {
	var result RsActiveUsers

	// DAU (today)
	dauQuery := `
		SELECT COUNT(DISTINCT user_id)
		FROM tbl_audit_log
		WHERE DATE(created_at) = CURRENT_DATE
	`
	err := r.db.GetContext(ctx, &result.DAU, dauQuery)
	if err != nil {
		return nil, fmt.Errorf("get DAU: %w", err)
	}

	// WAU (last 7 days)
	wauQuery := `
		SELECT COUNT(DISTINCT user_id)
		FROM tbl_audit_log
		WHERE created_at >= NOW() - INTERVAL '7 days'
	`
	err = r.db.GetContext(ctx, &result.WAU, wauQuery)
	if err != nil {
		return nil, fmt.Errorf("get WAU: %w", err)
	}

	// MAU (last 30 days)
	mauQuery := `
		SELECT COUNT(DISTINCT user_id)
		FROM tbl_audit_log
		WHERE created_at >= NOW() - INTERVAL '30 days'
	`
	err = r.db.GetContext(ctx, &result.MAU, mauQuery)
	if err != nil {
		return nil, fmt.Errorf("get MAU: %w", err)
	}

	// DAU/MAU ratio
	if result.MAU > 0 {
		result.DAUToMAURatio = (float64(result.DAU) / float64(result.MAU)) * 100
	}

	// Timeline
	timelineQuery := `
		SELECT 
			DATE(created_at) as date,
			COUNT(DISTINCT user_id) as active_users
		FROM tbl_audit_log
		WHERE created_at BETWEEN $1 AND $2
		GROUP BY DATE(created_at)
		ORDER BY date
	`
	rows, err := r.db.QueryxContext(ctx, timelineQuery, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("get timeline: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var point ActiveUsersPoint
		var date time.Time
		err := rows.Scan(&date, &point.ActiveUsers)
		if err != nil {
			return nil, fmt.Errorf("scan timeline: %w", err)
		}
		point.Date = date.Format("2006-01-02")
		result.Timeline = append(result.Timeline, point)
	}

	return &result, nil
}

// GetPractitionerDetails retrieves detailed practitioner info with clinics and accountants
func (r *repository) GetPractitionerDetails(ctx context.Context, practitionerID uuid.UUID) (*RsPractitionerDetail, error) {
	// Get practitioner basic info
	practQuery := `
		SELECT 
			p.id,
			u.first_name || ' ' || u.last_name as name,
			u.email,
			u.phone,
			u.created_at
		FROM tbl_practitioner p
		JOIN tbl_user u ON p.user_id = u.id
		WHERE p.id = $1 AND p.deleted_at IS NULL
	`
	var result RsPractitionerDetail
	err := r.db.GetContext(ctx, &result, practQuery, practitionerID)
	if err != nil {
		return nil, fmt.Errorf("get practitioner: %w", err)
	}

	// Get subscription info
	subQuery := `
		SELECT 
			s.id,
			s.name,
			ps.status,
			ps.start_date,
			ps.end_date
		FROM tbl_practitioner_subscription ps
		JOIN tbl_subscription s ON ps.subscription_id = s.id
		WHERE ps.practitioner_id = $1
			AND ps.status = 'ACTIVE'
			AND ps.deleted_at IS NULL
		ORDER BY ps.created_at DESC
		LIMIT 1
	`
	var sub SubscriptionInfo
	err = r.db.GetContext(ctx, &sub, subQuery, practitionerID)
	if err == nil {
		result.Subscription = &sub
	}

	// Get clinics with accountants
	clinicQuery := `
		SELECT 
			c.id,
			c.name,
			c.abn
		FROM tbl_clinic c
		WHERE c.practitioner_id = $1 AND c.deleted_at IS NULL
		ORDER BY c.created_at DESC
	`
	rows, err := r.db.QueryxContext(ctx, clinicQuery, practitionerID)
	if err != nil {
		return nil, fmt.Errorf("get clinics: %w", err)
	}
	defer rows.Close()

	accountantMap := make(map[uuid.UUID]bool)
	for rows.Next() {
		var clinic ClinicInfo
		err := rows.Scan(&clinic.ID, &clinic.Name, &clinic.ABN)
		if err != nil {
			return nil, fmt.Errorf("scan clinic: %w", err)
		}

		// Get accountants for this clinic
		accQuery := `
			SELECT DISTINCT
				a.id,
				u.first_name || ' ' || u.last_name as name,
				u.email,
				u.phone,
				'read,update' as permissions
			FROM tbl_invite_permissions ip
			JOIN tbl_accountant a ON ip.accountant_id = a.id
			JOIN tbl_user u ON a.user_id = u.id
			WHERE ip.practitioner_id = $1
				AND ip.entity_id = $2
				AND ip.deleted_at IS NULL
				AND a.deleted_at IS NULL
		`
		accRows, err := r.db.QueryxContext(ctx, accQuery, practitionerID, clinic.ID)
		if err != nil {
			return nil, fmt.Errorf("get accountants: %w", err)
		}

		for accRows.Next() {
			var acc AccountantInfo
			err := accRows.Scan(&acc.ID, &acc.Name, &acc.Email, &acc.Phone, &acc.Permissions)
			if err != nil {
				accRows.Close()
				return nil, fmt.Errorf("scan accountant: %w", err)
			}
			clinic.Accountants = append(clinic.Accountants, acc)
			accountantMap[acc.ID] = true
		}
		accRows.Close()

		result.Clinics = append(result.Clinics, clinic)
	}

	result.TotalClinics = len(result.Clinics)
	result.TotalAccountants = len(accountantMap)

	return &result, nil
}

// ListPractitionersWithDetails retrieves filtered and paginated list of practitioners with details
func (r *repository) ListPractitionersWithDetails(ctx context.Context, filter *PractitionerFilter) ([]*RsPractitionerDetail, int, error) {
	cf := filter.MapToFilter()

	// Base query for practitioners with subscription join
	baseQuery := `
		FROM tbl_practitioner p
		JOIN tbl_user u ON p.user_id = u.id
		LEFT JOIN tbl_practitioner_subscription ps ON ps.practitioner_id = p.id AND ps.status = 'ACTIVE' AND ps.deleted_at IS NULL
		LEFT JOIN tbl_subscription s ON ps.subscription_id = s.id
		WHERE p.deleted_at IS NULL
	`

	// Add subscription filter if specified
	if filter.HasActiveSubscription != nil {
		if *filter.HasActiveSubscription {
			baseQuery += " AND ps.id IS NOT NULL"
		} else {
			baseQuery += " AND ps.id IS NULL"
		}
	}

	// Get total count
	countQuery, countArgs := common.BuildQuery(baseQuery, cf, practitionerColumns, practitionerSearchCols, true)
	var total int
	err := r.db.GetContext(ctx, &total, r.db.Rebind(countQuery), countArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("count practitioners: %w", err)
	}

	// Get practitioner IDs with sorting and pagination
	selectQuery := "SELECT DISTINCT p.id " + baseQuery
	listQuery, listArgs := common.BuildQuery(selectQuery, cf, practitionerColumns, practitionerSearchCols, false)
	
	rows, err := r.db.QueryxContext(ctx, r.db.Rebind(listQuery), listArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("get practitioner ids: %w", err)
	}
	defer rows.Close()

	var results []*RsPractitionerDetail
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, 0, fmt.Errorf("scan id: %w", err)
		}

		detail, err := r.GetPractitionerDetails(ctx, id)
		if err != nil {
			return nil, 0, err
		}
		results = append(results, detail)
	}

	return results, total, nil
}
