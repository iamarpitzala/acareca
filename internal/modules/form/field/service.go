package field

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/iamarpitzala/acareca/internal/modules/business/clinic"
	"github.com/iamarpitzala/acareca/internal/modules/business/coa"
)

type IService interface {
	Create(ctx context.Context, formVersionID uuid.UUID, req *RqFormField) (*RsFormField, error)
	GetByID(ctx context.Context, id uuid.UUID) (*RsFormField, error)
	Update(ctx context.Context, id uuid.UUID, req *RqUpdateFormField) (*RsFormField, error)
	Delete(ctx context.Context, id uuid.UUID) error
	ListByFormVersionID(ctx context.Context, formVersionID uuid.UUID) ([]*RsFormField, error)
}

var ErrCoaNotFound = errors.New("chart of account not found or does not belong to this practice")

type Service struct {
	repo      IRepository
	coaSvc    coa.Service
	clinicSvc clinic.Service
}

func NewService(repo IRepository, coaSvc coa.Service, clinicSvc clinic.Service) IService {
	return &Service{
		repo:      repo,
		coaSvc:    coaSvc,
		clinicSvc: clinicSvc,
	}
}

// Create implements [IService].
func (s *Service) Create(ctx context.Context, formVersionID uuid.UUID, req *RqFormField) (*RsFormField, error) {
	clinic, err := s.clinicSvc.GetClinicByID(ctx, formVersionID)
	if err != nil {
		return nil, err
	}

	coaID, err := uuid.Parse(req.CoaID)
	if err != nil {
		return nil, err
	}
	practiceID, err := uuid.Parse(clinic.PracticeID)
	if err != nil {
		return nil, err
	}
	if _, err := s.coaSvc.GetChartOfAccount(ctx, coaID, practiceID); err != nil {
		if errors.Is(err, coa.ErrNotFound) {
			return nil, ErrCoaNotFound
		}
		return nil, err
	}
	f := req.ToDB(formVersionID)
	if err := s.repo.Create(ctx, f); err != nil {
		return nil, err
	}
	return f.ToRs(), nil
}

// GetByID implements [IService].
func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*RsFormField, error) {
	f, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return f.ToRs(), nil
}

// Update implements [IService].
func (s *Service) Update(ctx context.Context, id uuid.UUID, req *RqUpdateFormField) (*RsFormField, error) {
	existing, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	clinic, err := s.clinicSvc.GetClinicByID(ctx, existing.FormVersionID)
	if err != nil {
		return nil, err
	}
	if req.CoaID != nil {
		coaID, err := uuid.Parse(*req.CoaID)
		if err != nil {
			return nil, err
		}
		practiceID, err := uuid.Parse(clinic.PracticeID)
		if err != nil {
			return nil, err
		}
		if _, err := s.coaSvc.GetChartOfAccount(ctx, coaID, practiceID); err != nil {
			if errors.Is(err, coa.ErrNotFound) {
				return nil, ErrCoaNotFound
			}
			return nil, err
		}
		existing.CoaID = coaID
	}
	if req.Label != nil {
		existing.Label = *req.Label
	}
	if req.SectionType != nil {
		existing.SectionType = *req.SectionType
	}
	if req.PaymentResponsibility != nil {
		existing.PaymentResponsibility = *req.PaymentResponsibility
	}
	if req.TaxType != nil {
		existing.TaxType = *req.TaxType
	}
	updated, err := s.repo.Update(ctx, existing)
	if err != nil {
		return nil, err
	}
	return updated.ToRs(), nil
}

// Delete implements [IService].
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

// ListByFormVersionID implements [IService].
func (s *Service) ListByFormVersionID(ctx context.Context, formVersionID uuid.UUID) ([]*RsFormField, error) {
	list, err := s.repo.ListByFormVersionID(ctx, formVersionID)
	if err != nil {
		return nil, err
	}
	rs := make([]*RsFormField, 0, len(list))
	for _, f := range list {
		rs = append(rs, f.ToRs())
	}
	return rs, nil
}
