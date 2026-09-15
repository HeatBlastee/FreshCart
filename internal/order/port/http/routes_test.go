package http

import (
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/quangdangfit/gocommon/validation"

	dbMocks "freshcart/pkg/dbs/mocks"
)

func TestRoutes(t *testing.T) {
	mockDB := dbMocks.NewDatabase(t)
	Routes(gin.New().Group("/"), mockDB, validation.New())
}
