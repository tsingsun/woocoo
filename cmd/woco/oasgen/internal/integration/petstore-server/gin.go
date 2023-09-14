package main

import (
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/cmd/woco/oasgen/internal/integration/petstore"
	"github.com/tsingsun/woocoo/cmd/woco/oasgen/internal/integration/petstore/server"
	"github.com/tsingsun/woocoo/web/handler"
	"time"
)

func main() {
	// implement your server
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	router.Group("/")
	router.Use(handler.ErrorHandle().ApplyFunc(nil))
	imp := &Service{}
	server.RegisterValidator()
	server.RegisterUserHandlers(&router.RouterGroup, imp)
	server.RegisterStoreHandlers(&router.RouterGroup, imp)
	server.RegisterPetHandlers(&router.RouterGroup, imp)
	router.Run(":18080")
}

type Service struct {
	petstore.UnimplementedStoreService
	petstore.UnimplementedPetService
	petstore.UnimplementedUserService
}

func (s Service) AddPet(c *gin.Context, req *petstore.AddPetRequest) (_ *petstore.Pet, err error) {
	return &petstore.Pet{
		ID:   req.NewPet.ID,
		Name: req.NewPet.Name,
	}, nil
}

func (s Service) FindPetsByTags(ctx *gin.Context, req *petstore.FindPetsByTagsRequest) (petstore.Pets, error) {
	return []*petstore.Pet{
		{ID: 1, Name: "dog"},
	}, nil
}

func (s Service) FindPetsByStatus(c *gin.Context, req *petstore.FindPetsByStatusRequest) ([]*petstore.Pet, error) {
	st := "unknown"
	if len(req.Status) > 0 {
		st = req.Status[0]
	}
	return []*petstore.Pet{
		{ID: 1, Name: "dog", Status: st},
	}, nil
}

func (s Service) UpdatePetWithForm(ctx *gin.Context, req *petstore.UpdatePetWithFormRequest) (err error) {
	return errors.New("UpdatePetWithForm Error")
}

func (s Service) LoginUser(ctx *gin.Context, req *petstore.LoginUserRequest) (_ string, err error) {
	return "ok", nil
}

func (s Service) GetOrderById(ctx *gin.Context, req *petstore.GetOrderByIdRequest) (*petstore.Order, error) {
	return &petstore.Order{
		ID: 1, PetId: 1, Quantity: 1, ShipDate: time.Now(), Status: "placed", Complete: true,
	}, nil
}

func (s Service) GetPetById(ctx *gin.Context, req *petstore.GetPetByIdRequest) (_ *petstore.Pet, err error) {
	return &petstore.Pet{
		ID: 1, Name: "dog", PhotoUrls: []string{"http://github.com"},
		Tags: []*petstore.Tag{{ID: 1, Name: "blue"}},
	}, nil
}

func (Service) UpdateUser(ctx *gin.Context, req *petstore.UpdateUserRequest) (err error) {
	return nil
}

func (Service) PlaceOrder(ctx *gin.Context, req *petstore.PlaceOrderRequest) (res *petstore.Order, err error) {
	return nil, nil
}
