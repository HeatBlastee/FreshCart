package grpc

import (
	"github.com/quangdangfit/gocommon/validation"
	"google.golang.org/grpc"

	"freshcart/internal/product/repository"
	"freshcart/internal/product/service"
	"freshcart/pkg/dbs"
	pb "freshcart/proto/gen/go/product"
)

func RegisterHandlers(svr *grpc.Server, db dbs.Database, validator validation.Validation) {
	productRepo := repository.NewProductRepository(db)
	productSvc := service.NewProductService(validator, productRepo)
	productHandler := NewProductHandler(productSvc)

	pb.RegisterProductServiceServer(svr, productHandler)
}
