package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/quangdangfit/gocommon/validation"

	notificationRepo "freshcart/internal/notification/repository"
	notificationSvc "freshcart/internal/notification/service"
	orderRepository "freshcart/internal/order/repository"
	orderService "freshcart/internal/order/service"
	"freshcart/internal/payment/repository"
	"freshcart/internal/payment/service"
	userRepo "freshcart/internal/user/repository"
	"freshcart/pkg/config"
	"freshcart/pkg/dbs"
	"freshcart/pkg/middleware"
	"freshcart/pkg/notification"
	"freshcart/pkg/payment"
	stripeProvider "freshcart/pkg/payment/stripe"
	"freshcart/pkg/response"
)

// Routes wires the payment domain. Uses the live config to construct a Stripe provider; the
// webhook route deliberately sits outside the JWT middleware (Stripe authenticates via the
// signature header instead).
func Routes(r *gin.RouterGroup, db dbs.Database, validator validation.Validation) {
	cfg := config.GetConfig()
	provider := stripeProvider.NewProvider(stripeProvider.Config{
		SecretKey:     cfg.StripeSecretKey,
		WebhookSecret: cfg.StripeWebhookSecret,
		APIBase:       cfg.StripeAPIBase,
	})

	paymentRepo := repository.NewPaymentRepository(db)

	// Build a minimal OrderService for MarkOrderPaid / UpdateOrderStatus on webhook events.
	orderSvc := orderService.NewOrderService(
		validator, db,
		orderRepository.NewOrderRepository(db),
		orderRepository.NewProductRepository(db),
		orderRepository.NewUserRepository(db),
		orderRepository.NewReservationRepository(db),
		orderService.NewCouponService(validator, orderRepository.NewCouponRepository(db)),
		notification.BuildDefault(notification.Settings{
			SMTPHost:     cfg.SMTPHost,
			SMTPPort:     cfg.SMTPPort,
			SMTPUser:     cfg.SMTPUser,
			SMTPPassword: cfg.SMTPPassword,
			EmailFrom:    cfg.EmailFrom,
			Prefs: notificationSvc.NewDBPreferenceChecker(
				notificationSvc.NewUserRepoLookup(userRepo.NewUserRepository(db)),
				notificationRepo.NewPreferenceRepository(db),
			),
			DLQ: notificationRepo.NewDeadLetterSink(db),
		}),
	)

	paymentSvc := service.NewPaymentService(provider, paymentRepo, orderSvc, orderSvc)
	handler := NewHandler(paymentSvc)

	// /orders/:id/payment-intent — authenticated, used by the customer to start checkout.
	r.POST("/orders/:id/payment-intent", middleware.JWTAuth(), handler.CreatePaymentIntent)

	// /webhooks/stripe — public, signature-verified.
	r.POST("/webhooks/stripe", handler.StripeWebhook)

	// /config/public — exposes non-secret config that the FE needs at boot
	// (Stripe publishable key, auth mode so the login UI can decide whether
	// to render the password form or SSO buttons).
	r.GET("/config/public", func(c *gin.Context) {
		response.JSON(c, http.StatusOK, gin.H{
			"stripe_publishable_key": cfg.StripePublishableKey,
			"auth_mode":              string(cfg.AuthMode),
		})
	})

	// Touch payment to silence the unused import lint when the package is imported but no
	// type from it is referenced directly. (Used transitively via the provider.)
	_ = payment.ErrInvalidSignature
}
