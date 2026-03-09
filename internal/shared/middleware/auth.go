package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/iamarpitzala/acareca/internal/shared/response"
	"github.com/iamarpitzala/acareca/pkg/config"
)

const UserIDKey = "userID"

var (
	errUnauthorized = errors.New("unauthorized")
	errForbidden    = errors.New("forbidden")
)

type SuperadminChecker func(ctx context.Context, userID string) (bool, error)

func RequireSuperadmin(check SuperadminChecker) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get(UserIDKey)
		if !exists {
			response.Error(c, http.StatusUnauthorized, errUnauthorized)
			return
		}
		id, ok := userID.(string)
		if !ok || id == "" {
			response.Error(c, http.StatusUnauthorized, errUnauthorized)
			return
		}
		isAdmin, err := check(c.Request.Context(), id)
		if err != nil {
			response.Error(c, http.StatusInternalServerError, err)
			return
		}
		if !isAdmin {
			response.Error(c, http.StatusForbidden, errForbidden)
			return
		}
		c.Next()
	}
}

func Auth(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			response.Error(c, http.StatusUnauthorized, errUnauthorized)
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errUnauthorized
			}
			return []byte(cfg.JWTSecret), nil
		})
		if err != nil || !token.Valid {
			response.Error(c, http.StatusUnauthorized, errUnauthorized)
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			response.Error(c, http.StatusUnauthorized, errUnauthorized)
			return
		}

		sub, ok := claims["sub"].(string)
		if !ok || sub == "" {
			response.Error(c, http.StatusUnauthorized, errUnauthorized)
			return
		}

		c.Set(UserIDKey, sub)
		c.Next()
	}
}
