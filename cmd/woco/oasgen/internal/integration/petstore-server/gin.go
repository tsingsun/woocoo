package main

import (
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/cmd/woco/oasgen/internal/integration/petstore"
	"github.com/tsingsun/woocoo/cmd/woco/oasgen/internal/integration/petstore/server"
)

func main() {
	// implement your server
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	imp := &Server{}
	server.RegisterUserHandlers(router, imp)
	server.RegisterStoreHandlers(router, imp)
	server.RegisterPetHandlers(router, imp)
	router.Run(":18080")
}

type Server struct {
}

func (s Server) AddPet(c *gin.Context, req petstore.AddPetRequest) (petstore.Pet, error) {
	//TODO implement me
	panic("implement me")
}

func (s Server) DeletePet(c *gin.Context, req petstore.DeletePetRequest) error {
	//TODO implement me
	panic("implement me")
}

func (s Server) FindPetsByStatus(c *gin.Context, req petstore.FindPetsByStatusRequest) ([]petstore.Pet, error) {
	//TODO implement me
	panic("implement me")
}

func (s Server) FindPetsByTags(c *gin.Context, req petstore.FindPetsByTagsRequest) ([]petstore.Pet, error) {
	//TODO implement me
	panic("implement me")
}

func (s Server) GetPetById(c *gin.Context, req petstore.GetPetByIdRequest) (petstore.Pet, error) {
	//TODO implement me
	panic("implement me")
}

func (s Server) UpdatePet(c *gin.Context, req petstore.UpdatePetRequest) (petstore.Pet, error) {
	//TODO implement me
	panic("implement me")
}

func (s Server) UpdatePetWithForm(c *gin.Context, req petstore.UpdatePetWithFormRequest) error {
	//TODO implement me
	panic("implement me")
}

func (s Server) UploadFile(c *gin.Context, req petstore.UploadFileRequest) (petstore.ApiResponse, error) {
	//TODO implement me
	panic("implement me")
}

func (s Server) CreateUser(c *gin.Context, req petstore.CreateUserRequest) error {
	//TODO implement me
	panic("implement me")
}

func (s Server) CreateUsersWithArrayInput(c *gin.Context) error {
	//TODO implement me
	panic("implement me")
}

func (s Server) CreateUsersWithListInput(c *gin.Context) error {
	//TODO implement me
	panic("implement me")
}

func (s Server) DeleteUser(c *gin.Context, req petstore.DeleteUserRequest) error {
	//TODO implement me
	panic("implement me")
}

func (s Server) GetUserByName(c *gin.Context, req petstore.GetUserByNameRequest) (petstore.User, error) {
	//TODO implement me
	panic("implement me")
}

func (s Server) LoginUser(c *gin.Context, req petstore.LoginUserRequest) (string, error) {
	//TODO implement me
	panic("implement me")
}

func (s Server) LogoutUser(c *gin.Context) error {
	//TODO implement me
	panic("implement me")
}

func (s Server) UpdateUser(c *gin.Context, req petstore.UpdateUserRequest) error {
	//TODO implement me
	panic("implement me")
}

func (s Server) DeleteOrder(c *gin.Context, req petstore.DeleteOrderRequest) error {
	//TODO implement me
	panic("implement me")
}

func (s Server) GetInventory(c *gin.Context) (any, error) {
	//TODO implement me
	panic("implement me")
}

func (s Server) GetOrderById(c *gin.Context, req petstore.GetOrderByIdRequest) (petstore.Order, error) {
	//TODO implement me
	panic("implement me")
}

func (s Server) PlaceOrder(c *gin.Context, req petstore.PlaceOrderRequest) (petstore.Order, error) {
	//TODO implement me
	panic("implement me")
}
