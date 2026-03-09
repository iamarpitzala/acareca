package route

import (
	"context"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/iamarpitzala/acareca/internal/modules/admin/subscription"
	"github.com/iamarpitzala/acareca/internal/modules/auth"
	"github.com/iamarpitzala/acareca/internal/modules/engine/calculation"
	"github.com/iamarpitzala/acareca/internal/modules/engine/method"
	tentant "github.com/iamarpitzala/acareca/internal/modules/tentant/setting"
	tentantSub "github.com/iamarpitzala/acareca/internal/modules/tentant/subscription"
	"github.com/iamarpitzala/acareca/internal/shared/db"
	"github.com/iamarpitzala/acareca/internal/shared/middleware"
	"github.com/iamarpitzala/acareca/pkg/config"
)

func RegisterRoutes(r *gin.Engine, cfg *config.Config) {
	v1 := r.Group("/api/v1")

	dbConn, err := db.DBConn(cfg)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	authRepo := auth.NewRepository(dbConn)
	authSvc := auth.NewService(authRepo, cfg)
	authHandler := auth.NewHandler(authSvc)
	auth.RegisterRoutes(v1, authHandler)

	calculationRepo := calculation.NewRepository(dbConn)
	calculationSvc := calculation.NewService(calculationRepo, method.NewService())
	calculationHandler := calculation.NewHandler(calculationSvc)
	calculation.RegisterRoutes(v1, calculationHandler)

	superadminCheck := func(ctx context.Context, userID string) (bool, error) {
		u, err := authRepo.FindByID(ctx, userID)
		if err != nil {
			return false, err
		}
		return u.IsSuperadmin != nil && *u.IsSuperadmin, nil
	}
	adminGroup := v1.Group("/admin")
	subscriptionGroup := adminGroup.Group("/subscription")
	subscriptionGroup.Use(middleware.Auth(cfg), middleware.RequireSuperadmin(superadminCheck))
	subscriptionRepo := subscription.NewRepository(dbConn)
	subscriptionSvc := subscription.NewService(subscriptionRepo)
	subscriptionHandler := subscription.NewHandler(subscriptionSvc)
	subscription.RegisterRoutes(subscriptionGroup, subscriptionHandler)

	tentantRepo := tentant.NewRepository(dbConn)
	tentantSvc := tentant.NewService(tentantRepo)
	tentantHandler := tentant.NewHandler(tentantSvc)
	tentantGroup := v1.Group("/tentant")
	tentant.RegisterRoutes(tentantGroup, tentantHandler)
	tentantSubRepo := tentantSub.NewRepository(dbConn)
	tentantSubSvc := tentantSub.NewService(tentantSubRepo)
	tentantSubHandler := tentantSub.NewHandler(tentantSubSvc)
	tentantSub.RegisterRoutes(tentantGroup.Group("/:id/subscription"), tentantSubHandler)
}
