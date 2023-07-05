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
	imp := &Server{}
	server.RegisterValidator()
	server.RegisterUserHandlers(&router.RouterGroup, imp)
	server.RegisterStoreHandlers(&router.RouterGroup, imp)
	server.RegisterPetHandlers(&router.RouterGroup, imp)
	router.Run(":18080")
}

type Server struct {
	petstore.UnimplementedStoreServer
	petstore.UnimplementedPetServer
	petstore.UnimplementedUserServer
}

func (s Server) AddPet(c *gin.Context, req *petstore.AddPetRequest) (_ *petstore.Pet, err error) {
	return &petstore.Pet{
		ID:   req.NewPet.ID,
		Name: req.NewPet.Name,
	}, nil
}

func (s Server) FindPetsByTags(ctx *gin.Context, req *petstore.FindPetsByTagsRequest) (petstore.Pets, error) {
	return []*petstore.Pet{
		{ID: 1, Name: "dog"},
	}, nil
}

func (s Server) FindPetsByStatus(c *gin.Context, req *petstore.FindPetsByStatusRequest) ([]*petstore.Pet, error) {
	st := "unknown"
	if len(req.Status) > 0 {
		st = req.Status[0]
	}
	return []*petstore.Pet{
		{ID: 1, Name: "dog", Status: st},
	}, nil
}

func (s Server) UpdatePetWithForm(ctx *gin.Context, req *petstore.UpdatePetWithFormRequest) (err error) {
	return errors.New("UpdatePetWithForm Error")
}

func (s Server) LoginUser(ctx *gin.Context, req *petstore.LoginUserRequest) (_ string, err error) {
	return "ok", nil
}

func (s Server) GetOrderById(ctx *gin.Context, req *petstore.GetOrderByIdRequest) (*petstore.Order, error) {
	return &petstore.Order{
		ID: 1, PetId: 1, Quantity: 1, ShipDate: time.Now(), Status: "placed", Complete: true,
	}, nil
}

func (s Server) GetPetById(ctx *gin.Context, req *petstore.GetPetByIdRequest) (_ *petstore.Pet, err error) {
	return &petstore.Pet{
		ID: 1, Name: "dog", PhotoUrls: []string{"http://github.com"},
		Tags: []*petstore.Tag{{ID: 1, Name: "blue"}},
	}, nil
}

func (Server) UpdateUser(ctx *gin.Context, req *petstore.UpdateUserRequest) (err error) {
	return nil
}

func (Server) PlaceOrder(ctx *gin.Context, req *petstore.PlaceOrderRequest) (res *petstore.Order, err error) {
	return nil, nil
}
