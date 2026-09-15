package http

import (
	"github.com/gin-gonic/gin"
	"github.com/quangdangfit/gocommon/validation"

	notificationRepo "freshcart/internal/notification/repository"
	notificationSvc "freshcart/internal/notification/service"
	"freshcart/internal/order/repository"
	"freshcart/internal/order/service"
	userRepository "freshcart/internal/user/repository"
	"freshcart/pkg/config"
	"freshcart/pkg/dbs"
	"freshcart/pkg/middleware"
	"freshcart/pkg/notification"
)

func Routes(r *gin.RouterGroup, db dbs.Database, validator validation.Validation) {
	productRepo := repository.NewProductRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	couponRepo := repository.NewCouponRepository(db)
	userRepo := repository.NewUserRepository(db)
	reservationRepo := repository.NewReservationRepository(db)

	cfg := config.GetConfig()
	couponSvc := service.NewCouponService(validator, couponRepo)
	prefChecker := notificationSvc.NewDBPreferenceChecker(
		notificationSvc.NewUserRepoLookup(userRepository.NewUserRepository(db)),
		notificationRepo.NewPreferenceRepository(db),
	)
	notifier := notification.BuildDefault(notification.Settings{
		SMTPHost:     cfg.SMTPHost,
		SMTPPort:     cfg.SMTPPort,
		SMTPUser:     cfg.SMTPUser,
		SMTPPassword: cfg.SMTPPassword,
		EmailFrom:    cfg.EmailFrom,
		Prefs:        prefChecker,
		DLQ:          notificationRepo.NewDeadLetterSink(db),
	})

	orderSvc := service.NewOrderService(validator, db, orderRepo, productRepo, userRepo, reservationRepo, couponSvc, notifier)
	orderHandler := NewOrderHandler(orderSvc)
	couponHandler := NewCouponHandler(couponSvc)

	authMiddleware := middleware.JWTAuth()
	adminMiddleware := middleware.AdminOnly()

	orderRoute := r.Group("/orders", authMiddleware)
	{
		orderRoute.POST("", orderHandler.PlaceOrder)
		orderRoute.GET("/:id", orderHandler.GetOrderByID)
		orderRoute.GET("", orderHandler.GetOrders)
		orderRoute.PUT("/:id/cancel", orderHandler.CancelOrder)
		orderRoute.PUT("/:id/status", adminMiddleware, orderHandler.UpdateOrderStatus)
	}

	couponRoute := r.Group("/coupons", authMiddleware)
	{
		couponRoute.POST("", adminMiddleware, couponHandler.CreateCoupon)
		couponRoute.GET("/:code", couponHandler.GetCouponByCode)
	}
}
