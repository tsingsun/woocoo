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
	petstore.UnimplementedStoreServer
	petstore.UnimplementedPetServer
	petstore.UnimplementedUserServer
}

func (s Server) FindPetsByTags(c *gin.Context, req *petstore.FindPetsByTagsRequest) ([]petstore.Pet, error) {
	return []petstore.Pet{
		{ID: 1, Name: "dog"},
	}, nil
}

func (s Server) UpdatePetWithForm(c *gin.Context, req *petstore.UpdatePetWithFormRequest) (err error) {
	return nil
}
