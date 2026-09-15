package http

import (
	"github.com/gin-gonic/gin"

	"freshcart/internal/notification/repository"
	"freshcart/internal/notification/service"
	"freshcart/pkg/dbs"
	"freshcart/pkg/middleware"
)

func Routes(r *gin.RouterGroup, db dbs.Database) {
	repo := repository.NewPreferenceRepository(db)
	svc := service.NewPreferenceService(repo)
	h := NewHandler(svc)

	g := r.Group("/me/notification-preferences", middleware.JWTAuth())
	g.GET("", h.ListPreferences)
	g.PUT("", h.SetPreference)
}
