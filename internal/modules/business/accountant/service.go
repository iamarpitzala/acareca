package accountant

import (
	"context"
)

type Service interface {
	ListUsers(ctx context.Context) ([]RsAccountantUser, error)
}

type service struct {
	repo Repository
}

func NewService(repo Repository, apiKey string) Service {
	return &service{
		repo: repo,
	}
}

func (s *service) ListUsers(ctx context.Context) ([]RsAccountantUser, error) {
	return s.repo.GetAllUsers(ctx)
}
