package analytics

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/iamarpitzala/acareca/internal/shared/util"
)

type Service interface {
	GetUserGrowth(ctx context.Context, filter *Filter) (*RsUserGrowth, error)
	GetSubscriptionMetrics(ctx context.Context) (*RsSubscriptionMetrics, error)
	GetActiveUsers(ctx context.Context, filter *Filter) (*RsActiveUsers, error)
	GetPractitionerDetails(ctx context.Context, practitionerID uuid.UUID) (*RsPractitionerDetail, error)
	ListPractitionersWithDetails(ctx context.Context, filter *PractitionerFilter) (*util.RsList, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &service{repo: repo}
}

func (s *service) GetUserGrowth(ctx context.Context, filter *Filter) (*RsUserGrowth, error) {
	startDate, endDate := s.getDateRange(filter)
	return s.repo.GetUserGrowth(ctx, startDate, endDate)
}

func (s *service) GetSubscriptionMetrics(ctx context.Context) (*RsSubscriptionMetrics, error) {
	return s.repo.GetSubscriptionMetrics(ctx)
}

func (s *service) GetActiveUsers(ctx context.Context, filter *Filter) (*RsActiveUsers, error) {
	startDate, endDate := s.getDateRange(filter)
	return s.repo.GetActiveUsers(ctx, startDate, endDate)
}

func (s *service) GetPractitionerDetails(ctx context.Context, practitionerID uuid.UUID) (*RsPractitionerDetail, error) {
	return s.repo.GetPractitionerDetails(ctx, practitionerID)
}

func (s *service) ListPractitionersWithDetails(ctx context.Context, filter *PractitionerFilter) (*util.RsList, error) {
	results, total, err := s.repo.ListPractitionersWithDetails(ctx, filter)
	if err != nil {
		return nil, err
	}

	offset := 0
	limit := 10
	if filter.Offset != nil {
		offset = *filter.Offset
	}
	if filter.Limit != nil {
		limit = *filter.Limit
	}

	var rsList util.RsList
	rsList.MapToList(results, total, offset, limit)
	return &rsList, nil
}

// getDateRange returns start and end dates based on filter or defaults to last 30 days
func (s *service) getDateRange(filter *Filter) (time.Time, time.Time) {
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -30) // Default: last 30 days

	if filter != nil {
		if filter.StartDate != nil {
			startDate = *filter.StartDate
		}
		if filter.EndDate != nil {
			endDate = *filter.EndDate
		}
	}

	return startDate, endDate
}
