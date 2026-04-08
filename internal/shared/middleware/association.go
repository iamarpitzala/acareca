package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iamarpitzala/acareca/internal/modules/business/invitation"
	"github.com/iamarpitzala/acareca/internal/shared/response"
	"github.com/iamarpitzala/acareca/internal/shared/util"
)

func PermissionMiddleware(inv invitation.Service) gin.HandlerFunc {
	return func(ctx *gin.Context) {

		entityId, ok := ctx.Get(util.EntityIDKey)
		if !ok {
			response.Error(ctx, http.StatusUnauthorized, errUnauthorized)
			ctx.Abort()
			return
		}
		role, ok := ctx.Get("role")
		if !ok {
			response.Error(ctx, http.StatusUnauthorized, errUnauthorized)
			ctx.Abort()
			return
		}
		// PERMISSION CHECK
		if strings.EqualFold(role.(string), util.RoleAccountant) {
			accId, err := uuid.Parse(entityId.(string))
			if err != nil {
				response.Error(ctx, http.StatusUnauthorized, errors.New("Invalid accountant id"))
				ctx.Abort()
				return
			}
			perms, _, err := inv.ListAccountantPermission(ctx, accId)
			if err != nil {
				response.Error(ctx, http.StatusUnauthorized, errors.New("Authentication error: "+err.Error()))
				ctx.Abort()
				return
			}

			hasPermission := false
			if perms != nil {
				for _, v := range *perms {
					if v.HasAccess("all") || v.HasAccess("create") {
						hasPermission = true
						break
					}
				}
			}
			if !hasPermission {
				response.Error(ctx, http.StatusForbidden, errors.New("You do not have permission to create forms for this clinic."))
				ctx.Abort()
				return
			}
		}

		ctx.Next()
	}
}
