package grpc

import (
	"github.com/quangdangfit/gocommon/validation"
	"google.golang.org/grpc"

	"freshcart/internal/user/repository"
	"freshcart/internal/user/service"
	"freshcart/pkg/dbs"
	pb "freshcart/proto/gen/go/user"
)

func RegisterHandlers(svr *grpc.Server, db dbs.Database, validator validation.Validation) {
	userRepo := repository.NewUserRepository(db)
	// gRPC stays on the local password flow; only the HTTP edge wires the
	// headless Authentik client when auth_mode=oidc.
	userSvc := service.NewUserService(validator, userRepo, nil)
	userHandler := NewUserHandler(userSvc)

	pb.RegisterUserServiceServer(svr, userHandler)
}
