package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/iamarpitzala/acareca/internal/shared/util"
	"github.com/iamarpitzala/acareca/pkg/config"
)

var (
	ErrEmailTaken      = errors.New("email already in use")
	ErrInvalidPassword = errors.New("invalid credentials")
)

type Service interface {
	Register(ctx context.Context, req *RqUser) (*RsUser, error)
	Login(ctx context.Context, req *RqLogin) (*RsToken, error)
}

type service struct {
	repo Repository
	cfg  *config.Config
}

func NewService(repo Repository, cfg *config.Config) Service {
	return &service{repo: repo, cfg: cfg}
}

func (s *service) Register(ctx context.Context, req *RqUser) (*RsUser, error) {
	if _, err := s.repo.FindByEmail(ctx, req.Email); err == nil {
		return nil, ErrEmailTaken
	}

	hashedPassword, err := util.GenerateHash(req.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	u := req.ToDBModel()
	u.ID = util.NewUUID()
	u.Password = hashedPassword

	created, err := s.repo.CreateUser(ctx, u)
	if err != nil {
		return nil, err
	}

	return created.ToRsUser(), nil
}

func (s *service) Login(ctx context.Context, req *RqLogin) (*RsToken, error) {
	user, err := s.repo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, ErrInvalidPassword
	}

	if err := util.CompareHash(user.Password, req.Password); err != nil {
		return nil, ErrInvalidPassword
	}

	if req.IsSuperadmin != nil && *req.IsSuperadmin {
		if user.IsSuperadmin == nil || !*user.IsSuperadmin {
			return nil, ErrInvalidPassword
		}
	}

	accessToken, err := util.SignToken(user.ID, 15*time.Minute, s.cfg.JWTSecret)
	if err != nil {
		return nil, err
	}

	refreshToken, err := util.SignToken(user.ID, 7*24*time.Hour, s.cfg.JWTSecret)
	if err != nil {
		return nil, err
	}

	return &RsToken{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		IsSuperadmin: user.IsSuperadmin,
	}, nil
}
