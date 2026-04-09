package analytics

import (
	"time"

	"github.com/google/uuid"
	"github.com/iamarpitzala/acareca/internal/shared/common"
)

// User Growth & Retention Models

type RsUserGrowth struct {
	TotalUsers         int               `json:"total_users"`
	TotalPractitioners int               `json:"total_practitioners"`
	TotalAccountants   int               `json:"total_accountants"`
	NewUsers30Days     int               `json:"new_users_30_days"`
	ActiveUsers30Days  int               `json:"active_users_30_days"`
	GrowthRate         float64           `json:"growth_rate_percentage"`
	RetentionRate      float64           `json:"retention_rate_percentage"`
	Timeline           []UserGrowthPoint `json:"timeline"`
}

type UserGrowthPoint struct {
	Date        string `json:"date"`
	NewUsers    int    `json:"new_users"`
	TotalUsers  int    `json:"total_users"`
	ActiveUsers int    `json:"active_users"`
}

// Subscription Distribution & MRR Models

type RsSubscriptionMetrics struct {
	TotalActiveSubscriptions int                        `json:"total_active_subscriptions"`
	MRR                      float64                    `json:"mrr"`
	ARR                      float64                    `json:"arr"`
	ARPU                     float64                    `json:"arpu"`
	ChurnRate                float64                    `json:"churn_rate_percentage"`
	Distribution             []SubscriptionDistribution `json:"distribution"`
}

type SubscriptionDistribution struct {
	PlanID     int     `json:"plan_id"`
	PlanName   string  `json:"plan_name"`
	Count      int     `json:"count"`
	Revenue    float64 `json:"revenue"`
	Percentage float64 `json:"percentage"`
}

// Daily/Monthly Active Users Models

type RsActiveUsers struct {
	DAU            int                `json:"dau"`
	WAU            int                `json:"wau"`
	MAU            int                `json:"mau"`
	DAUToMAURatio  float64            `json:"dau_to_mau_ratio"`
	Timeline       []ActiveUsersPoint `json:"timeline"`
}

type ActiveUsersPoint struct {
	Date        string `json:"date"`
	ActiveUsers int    `json:"active_users"`
}

// Practitioner with Clinics and Accountants Models

type RsPractitionerDetail struct {
	ID               uuid.UUID         `json:"id"`
	Name             string            `json:"name"`
	Email            string            `json:"email"`
	Phone            *string           `json:"phone"`
	CreatedAt        time.Time         `json:"created_at"`
	Subscription     *SubscriptionInfo `json:"subscription,omitempty"`
	Clinics          []ClinicInfo      `json:"clinics"`
	TotalClinics     int               `json:"total_clinics"`
	TotalAccountants int               `json:"total_accountants"`
}

type SubscriptionInfo struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
}

type ClinicInfo struct {
	ID          uuid.UUID        `json:"id"`
	Name        string           `json:"name"`
	ABN         *string          `json:"abn,omitempty"`
	Accountants []AccountantInfo `json:"accountants"`
}

type AccountantInfo struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	Phone       *string   `json:"phone,omitempty"`
	Permissions string    `json:"permissions"`
}

// Filter for analytics queries
type Filter struct {
	StartDate *time.Time `form:"start_date"`
	EndDate   *time.Time `form:"end_date"`
	Interval  *string    `form:"interval"` // day, week, month
}

// PractitionerFilter for listing practitioners with details
type PractitionerFilter struct {
	Email                 *string `form:"email"`
	Name                  *string `form:"name"`
	Phone                 *string `form:"phone"`
	HasActiveSubscription *bool   `form:"has_active_subscription"`
	SubscriptionName      *string `form:"subscription_name"`
	common.Filter
}

var practitionerColumns = map[string]string{
	"id":                  "p.id",
	"email":               "u.email",
	"first_name":          "u.first_name",
	"last_name":           "u.last_name",
	"name":                "CONCAT(u.first_name, ' ', u.last_name)",
	"phone":               "u.phone",
	"subscription_name":   "s.name",
	"created_at":          "p.created_at",
}

var practitionerSearchCols = []string{
	"u.email",
	"CONCAT(u.first_name, ' ', u.last_name)",
}

func (f *PractitionerFilter) MapToFilter() common.Filter {
	filters := map[string]interface{}{}
	operators := map[string]common.Operator{}

	if f.Email != nil && *f.Email != "" {
		filters["email"] = *f.Email
	}

	if f.Name != nil && *f.Name != "" {
		filters["name"] = "%" + *f.Name + "%"
		operators["name"] = common.OpLike
	}

	if f.Phone != nil && *f.Phone != "" {
		filters["phone"] = "%" + *f.Phone + "%"
		operators["phone"] = common.OpLike
	}

	if f.SubscriptionName != nil && *f.SubscriptionName != "" {
		filters["subscription_name"] = "%" + *f.SubscriptionName + "%"
		operators["subscription_name"] = common.OpLike
	}

	return common.NewFilter(f.Search, filters, operators, f.Limit, f.Offset, f.SortBy, f.OrderBy)
}
