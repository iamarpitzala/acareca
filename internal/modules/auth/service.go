package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/iamarpitzala/acareca/internal/modules/admin/audit"
	"github.com/iamarpitzala/acareca/internal/modules/business/practitioner"
	auditctx "github.com/iamarpitzala/acareca/internal/shared/audit"
	"github.com/iamarpitzala/acareca/internal/shared/middleware"
	"github.com/iamarpitzala/acareca/internal/shared/util"
	"github.com/iamarpitzala/acareca/pkg/config"
	"github.com/jmoiron/sqlx"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const providerGoogle = "google"

var (
	ErrEmailTaken      = errors.New("email already in use")
	ErrInvalidPassword = errors.New("invalid credentials")
	ErrOAuthOnly       = errors.New("account uses Google sign-in; password login is not available")
)

type OnUserCreated func(ctx context.Context, userID string) error

type Service interface {
	Register(ctx context.Context, req *RqUser) (*RsUser, error)
	Login(ctx context.Context, req *RqLogin) (*RsToken, error)
	Logout(ctx context.Context, refreshToken string) error
	GoogleAuthURL(state string) *RsGoogleAuthURL
	GoogleCallback(ctx context.Context, code string) (*RsToken, error)

	VerifyEmail(ctx context.Context, tokenStr string) error
}

type service struct {
	repo            Repository
	cfg             *config.Config
	db              *sqlx.DB
	oauthConfig     *oauth2.Config
	practitionerSvc practitioner.IService
	auditSvc        audit.Service
}

func NewService(repo Repository, cfg *config.Config, db *sqlx.DB, practitionerSvc practitioner.IService, auditSvc audit.Service) Service {
	oauthCfg := &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  cfg.GoogleRedirectURL,
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
		},
		Endpoint: google.Endpoint,
	}
	return &service{
		repo:            repo,
		cfg:             cfg,
		oauthConfig:     oauthCfg,
		db:              db,
		practitionerSvc: practitionerSvc,
		auditSvc:        auditSvc,
	}
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
	u.Password = &hashedPassword

	var created *User
	var createdPractitioner *practitioner.RsPractitioner
	var tokenID uuid.UUID

	err = util.RunInTransaction(ctx, s.db, func(ctx context.Context, tx *sqlx.Tx) error {
		var err error
		created, err = s.repo.CreateUser(ctx, u, tx)
		if err != nil {
			return err
		}
		createdPractitioner, err = s.practitionerSvc.CreatePractitioner(ctx, &practitioner.RqCreatePractitioner{UserID: created.ID.String()}, tx)
		if err != nil {
			return fmt.Errorf("create practitioner: %w", err)
		}

		// Generate Verification Token
		tokenID = uuid.New()
		vToken := &VerificationToken{
			ID:        tokenID,
			EntityID:  createdPractitioner.ID,
			Status:    TokenStatusPending,
			ExpiresAt: time.Now().Add(10 * time.Minute),
		}

		if err := s.repo.CreateVerificationToken(ctx, tx, vToken); err != nil {
			return fmt.Errorf("create verification token: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Send Email Asynchronously
	go func() {
		if err := s.sendVerificationEmail(created.Email, created.FirstName, tokenID); err != nil {
			fmt.Printf("[AUTH ERROR] Failed to send verification email: %v\n", err)
		}
	}()

	// Audit log: user registration
	meta := auditctx.GetMetadata(ctx)
	userIDStr := created.ID.String()
	s.auditSvc.LogAsync(&audit.LogEntry{
		PracticeID: meta.PracticeID,
		UserID:     &userIDStr,
		Action:     auditctx.ActionUserRegistered,
		Module:     auditctx.ModuleAuth,
		EntityType: strPtr(auditctx.EntityUser),
		EntityID:   &userIDStr,
		AfterState: sanitizeUser(created),
		IPAddress:  meta.IPAddress,
		UserAgent:  meta.UserAgent,
	})

	return created.ToRsUser(), nil
}

func (s *service) Login(ctx context.Context, req *RqLogin) (*RsToken, error) {
	user, err := s.repo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, ErrInvalidPassword
	}

	if user.Password == nil || *user.Password == "" {
		return nil, ErrOAuthOnly
	}

	if err := util.CompareHash(req.Password, *user.Password); err != nil {
		return nil, ErrInvalidPassword
	}

	if req.IsSuperadmin != nil && *req.IsSuperadmin {
		if user.IsSuperadmin == nil || !*user.IsSuperadmin {
			return nil, ErrInvalidPassword
		}
	}

	practitionerID, err := s.practitionerSvc.GetPractitionerByUserID(ctx, user.ID.String())
	if err != nil {
		return nil, err
	}

	// Audit log: user login
	meta := auditctx.GetMetadata(ctx)
	userIDStr := user.ID.String()
	s.auditSvc.LogAsync(&audit.LogEntry{
		PracticeID: meta.PracticeID,
		UserID:     &userIDStr,
		Action:     auditctx.ActionUserLoggedIn,
		Module:     auditctx.ModuleAuth,
		EntityType: strPtr(auditctx.EntitySession),
		EntityID:   nil,
		AfterState: map[string]interface{}{"email": user.Email},
		IPAddress:  meta.IPAddress,
		UserAgent:  meta.UserAgent,
	})

	return s.issueTokens(ctx, user, practitionerID.ID.String())
}

func (s *service) Logout(ctx context.Context, refreshToken string) error {
	sess, err := s.repo.FindSessionByRefreshToken(ctx, refreshToken)
	if err != nil {
		return err
	}

	if err := s.repo.DeleteSession(ctx, sess.ID); err != nil {
		return err
	}

	// Audit log: user logged out
	meta := auditctx.GetMetadata(ctx)
	sessIDStr := sess.ID.String()
	userIDStr := sess.UserID.String()
	s.auditSvc.LogAsync(&audit.LogEntry{
		PracticeID: meta.PracticeID,
		UserID:     &userIDStr,
		Action:     auditctx.ActionUserLoggedOut,
		Module:     auditctx.ModuleAuth,
		EntityType: strPtr(auditctx.EntitySession),
		EntityID:   &sessIDStr,
		IPAddress:  meta.IPAddress,
		UserAgent:  meta.UserAgent,
	})

	return nil
}

func (s *service) GoogleAuthURL(state string) *RsGoogleAuthURL {
	url := s.oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
	return &RsGoogleAuthURL{URL: url}
}

func (s *service) GoogleCallback(ctx context.Context, code string) (*RsToken, error) {
	oauthToken, err := s.oauthConfig.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("exchange oauth code: %w", err)
	}

	googleUser, err := s.fetchGoogleUserInfo(ctx, oauthToken)
	if err != nil {
		return nil, err
	}

	user, err := s.repo.FindByEmail(ctx, googleUser.Email)
	isNewUser := false

	if err = util.RunInTransaction(ctx, s.db, func(ctx context.Context, tx *sqlx.Tx) error {
		if err != nil {
			if !errors.Is(err, ErrNotFound) {
				return err
			}
			user, err = s.repo.CreateUser(ctx, &User{
				Email:     googleUser.Email,
				FirstName: googleUser.FirstName,
				LastName:  googleUser.LastName,
			}, tx)
			if err != nil {
				return err
			}
			isNewUser = true
		}

		expiresAt := oauthToken.Expiry
		accessTokenStr := oauthToken.AccessToken
		refreshTokenStr := oauthToken.RefreshToken

		ap := &AuthProvider{
			UserID:         user.ID,
			Provider:       providerGoogle,
			AccessToken:    &accessTokenStr,
			TokenExpiresAt: &expiresAt,
		}
		if refreshTokenStr != "" {
			ap.RefreshToken = &refreshTokenStr
		}

		if _, err := s.repo.UpsertAuthProvider(ctx, ap, tx); err != nil {
			return err
		}

		if isNewUser {
			_, err = s.practitionerSvc.CreatePractitioner(ctx, &practitioner.RqCreatePractitioner{UserID: user.ID.String()}, tx)
			if err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("google oauth transaction: %w", err)
	}

	// Audit log: OAuth login/registration — only reached after a successful commit
	meta := auditctx.GetMetadata(ctx)
	userIDStr := user.ID.String()
	action := auditctx.ActionUserLoggedIn
	if isNewUser {
		action = auditctx.ActionUserRegistered
	}
	s.auditSvc.LogAsync(&audit.LogEntry{
		PracticeID: meta.PracticeID,
		UserID:     &userIDStr,
		Action:     action,
		Module:     auditctx.ModuleAuth,
		EntityType: strPtr(auditctx.EntityUser),
		EntityID:   &userIDStr,
		AfterState: map[string]interface{}{"email": user.Email, "provider": "google"},
		IPAddress:  meta.IPAddress,
		UserAgent:  meta.UserAgent,
	})

	practitionerID, err := s.practitionerSvc.GetPractitionerByUserID(ctx, user.ID.String())
	if err != nil {
		return nil, err
	}

	return s.issueTokens(ctx, user, practitionerID.ID.String())
}

func (s *service) fetchGoogleUserInfo(ctx context.Context, token *oauth2.Token) (*GoogleUserInfo, error) {
	client := s.oauthConfig.Client(ctx, token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v2/userinfo")
	if err != nil {
		return nil, fmt.Errorf("fetch google user info: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read google user info response: %w", err)
	}

	var info GoogleUserInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, fmt.Errorf("parse google user info: %w", err)
	}
	return &info, nil
}

func (s *service) issueTokens(ctx context.Context, user *User, practitionerID string) (*RsToken, error) {
	accessToken, err := util.SignToken(user.ID.String(), practitionerID, 15*time.Hour, s.cfg.JWTSecret)
	if err != nil {
		return nil, err
	}

	refreshToken, err := util.SignToken(user.ID.String(), practitionerID, 7*24*time.Hour, s.cfg.JWTSecret)
	if err != nil {
		return nil, err
	}

	ua := middleware.UserAgentFromCtx(ctx)
	ip := middleware.IPFromCtx(ctx)

	sess := &Session{
		ID:           uuid.New(),
		UserID:       user.ID,
		RefreshToken: refreshToken,
		ExpiresAt:    time.Now().Add(7 * 24 * time.Hour),
	}
	if ua != "" {
		sess.UserAgent = &ua
	}
	if ip != "" {
		sess.IPAddress = &ip
	}

	if _, err := s.repo.CreateSession(ctx, sess); err != nil {
		return nil, err
	}

	return &RsToken{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		IsSuperadmin: user.IsSuperadmin,
	}, nil
}

// Helper functions for audit logging

func strPtr(s string) *string {
	return &s
}

func sanitizeUser(u *User) map[string]interface{} {
	return map[string]interface{}{
		"id":         u.ID.String(),
		"email":      u.Email,
		"first_name": u.FirstName,
		"last_name":  u.LastName,
	}
}

// Helper function for sending verification email via Resend API
func (s *service) sendVerificationEmail(to string, firstName string, tokenID uuid.UUID) error {
	url := "https://api.resend.com/emails"
	apikey := os.Getenv("RESEND_API_KEY")

	verificationLink := fmt.Sprintf("https://acareca.com/verify-email?token=%s", tokenID)
	expiryTime := "10 minutes"

	payload := map[string]interface{}{
		"from":    "Acareca <hardik@zenithive.digital>",
		"to":      []string{to},
		"subject": "Verify your Acareca account",
		"html": fmt.Sprintf(`
			<div style="font-family: sans-serif; color: #333; max-width: 600px; margin: auto; border: 1px solid #eee; padding: 20px;">
				<h2 style="color: #1a73e8;">Verify your email</h2>
				<p>Hi %s,</p>
				<p>Thank you for signing up with Acareca! To complete your registration and activate your account, please verify your email address by clicking the button below:</p>
				<div style="text-align: center; margin: 30px 0;">
					<a href="%s" style="background-color: #1a73e8; color: white; padding: 14px 28px; text-decoration: none; border-radius: 4px; font-weight: bold; display: inline-block;">
						Verify My Account
					</a>
				</div>
				<p style="font-size: 14px; color: #666;">If the button above doesn’t work, you can also copy and paste the following link into your browser:</p>
				<p style="font-size: 12px; word-break: break-all; color: #1a73e8;">%s</p>
				<p style="font-size: 14px; color: #666;">This verification link will expire in <strong>%s</strong>.</p>
				<hr style="border: none; border-top: 1px solid #eee; margin: 20px 0;" />
				<p style="font-size: 12px; color: #888;">If you did not create this account, you can safely ignore this email.</p>
				<p style="font-size: 12px; color: #888;">Best regards,<br>The Acareca Team</p>
			</div>
		`, firstName, verificationLink, verificationLink, expiryTime),
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+apikey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("resend error: %s", string(body))
	}

	return nil
}

func (s *service) VerifyEmail(ctx context.Context, tokenStr string) error {
	tokenID, err := uuid.Parse(tokenStr)
	if err != nil {
		return errors.New("invalid token format")
	}

	token, err := s.repo.GetToken(ctx, tokenID)
	if err != nil {
		return errors.New("verification link not found")
	}

	if token.Status != TokenStatusPending {
		return fmt.Errorf("this link has already been %s", strings.ToLower(token.Status))
	}

	if time.Now().After(token.ExpiresAt) {
		return errors.New("verification link has expired")
	}

	return s.repo.MarkUserVerified(ctx, token.EntityID, token.ID)
}
