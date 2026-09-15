package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/quangdangfit/gocommon/logger"
	"github.com/quangdangfit/gocommon/validation"

	orderRepository "freshcart/internal/order/repository"
	orderService "freshcart/internal/order/service"
	grpcServer "freshcart/internal/server/grpc"
	httpServer "freshcart/internal/server/http"
	"freshcart/pkg/config"
	"freshcart/pkg/dbs"
	"freshcart/pkg/eventbus"
	"freshcart/pkg/notification"
	"freshcart/pkg/redis"
)

//	@title			FreshCart Swagger API
//	@version		1.0
//	@description	Swagger API for FreshCart.
//	@termsOfService	http://swagger.io/terms/

//	@contact.name	Quang Dang
//	@contact.email	quangdangfit@gmail.com

//	@license.name	MIT
//	@license.url	https://github.com/MartinHeinz/go-project-blueprint/blob/master/LICENSE

//	@securityDefinitions.apikey	ApiKeyAuth
//	@in							header
//	@name						Authorization

//	@BasePath	/api/v1

func main() {
	cfg := config.LoadConfig()
	logger.Initialize(cfg.Environment)

	db, err := dbs.NewDatabase(cfg.DatabaseURI)
	if err != nil {
		logger.Fatal("Cannot connect to database", err)
	}
	// Schema is managed externally via golang-migrate; see migrations/ and `make migrate-up`.

	validator := validation.New()

	// Wire the process-wide event bus and subscribe a logger sink for LowStock alerts.
	// Replace with an admin email channel once the notification service learns to consume
	// inventory events.
	bus := eventbus.New()
	eventbus.SetDefault(bus)
	bus.Subscribe(eventbus.TopicLowStock, func(_ context.Context, ev eventbus.Event) {
		ls := ev.(eventbus.LowStock)
		logger.Warnf("low stock: product=%s available=%d threshold=%d", ls.ProductID, ls.Available, ls.Threshold)
	})

	cache := redis.New(redis.Config{
		Address:  cfg.RedisURI,
		Password: cfg.RedisPassword,
		Database: cfg.RedisDB,
	})

	httpSvr := httpServer.NewServer(validator, db, cache)
	grpcSvr := grpcServer.NewServer(validator, db, cache)

	go func() {
		if err := httpSvr.Run(); err != nil {
			logger.Fatal(err)
		}
	}()

	go func() {
		if err := grpcSvr.Run(); err != nil {
			logger.Fatal(err)
		}
	}()

	// Background sweeper: release expired stock reservations and cancel their unpaid orders.
	sweeperCtx, sweeperCancel := context.WithCancel(context.Background())
	defer sweeperCancel()
	go runReservationSweeper(sweeperCtx, validator, db)

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("Shutting down servers...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	grpcSvr.Shutdown()

	if err := httpSvr.Shutdown(ctx); err != nil {
		logger.Error("HTTP server forced to shutdown: ", err)
	}

	sweeperCancel()
	logger.Info("Servers exited gracefully")
}

func runReservationSweeper(ctx context.Context, validator validation.Validation, db dbs.Database) {
	svc := orderService.NewOrderService(
		validator, db,
		orderRepository.NewOrderRepository(db),
		orderRepository.NewProductRepository(db),
		orderRepository.NewUserRepository(db),
		orderRepository.NewReservationRepository(db),
		orderService.NewCouponService(validator, orderRepository.NewCouponRepository(db)),
		notification.NewLoggerNotifier(),
	)
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			released, err := svc.SweepExpiredReservations(ctx, 100)
			if err != nil {
				logger.Error("reservation sweeper: ", err)
				continue
			}
			if released > 0 {
				logger.Infof("reservation sweeper released %d expired reservations", released)
			}
		}
	}
}
