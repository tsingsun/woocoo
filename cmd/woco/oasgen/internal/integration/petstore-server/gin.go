package petstore

import (
	"encoding/json"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/tsingsun/woocoo/pkg/gds"
	"time"
)

type Service struct {
	UnimplementedStoreServer
	UnimplementedPetServer
	UnimplementedUserServer
}

func (s Service) AddPet(c *gin.Context, req *AddPetRequest) (_ *Pet, err error) {
	return req.Pet, nil
}

func (s Service) FindPetsByTags(ctx *gin.Context, req *FindPetsByTagsRequest) (Pets, error) {
	return []*Pet{
		{ID: 1, Name: "dog"},
	}, nil
}

func (s Service) FindPetsByStatus(c *gin.Context, req *FindPetsByStatusRequest) ([]*Pet, error) {
	st := "unknown"
	if len(req.Status) > 0 {
		st = req.Status[0]
	}
	return []*Pet{
		{ID: 1, Name: "dog", Status: PetStatus(st)},
	}, nil
}

func (s Service) UpdatePetWithForm(ctx *gin.Context, req *UpdatePetWithFormRequest) (err error) {
	return errors.New("UpdatePetWithForm Error")
}

func (s Service) LoginUser(ctx *gin.Context, req *LoginUserRequest) (_ string, err error) {
	return "ok", nil
}

func (s Service) GetOrderById(ctx *gin.Context, req *GetOrderByIdRequest) (*Order, error) {
	return &Order{
		ID: 1, PetId: 1, Quantity: 1, ShipDate: gds.Ptr(time.Now()), Status: "placed", Complete: true,
	}, nil
}

func (s Service) GetPetById(ctx *gin.Context, req *GetPetByIdRequest) (_ *Pet, err error) {
	return &Pet{
		ID: 1, Name: "dog", PhotoUrls: []string{"http://github.com"},
		Tags: []*Tag{{ID: 1, Name: "blue"}},
	}, nil
}

func (Service) UpdateUser(ctx *gin.Context, req *UpdateUserRequest) (err error) {
	return nil
}

func (Service) PlaceOrder(ctx *gin.Context, req *PlaceOrderRequest) (res *Order, err error) {
	return nil, nil
}

func (Service) GetInventory(ctx *gin.Context) (res map[string]int32, err error) {
	return map[string]int32{
		"available": 1,
	}, nil
}

func (Service) DeletePet(ctx *gin.Context, req *DeletePetRequest) (err error) {
	if req.HeaderParams.APIKey != nil && *req.HeaderParams.APIKey == "wrong" {
		ctx.AbortWithStatus(401)
		return nil
	}
	if ctx.Request.Header.Get("X-Interceptor") == "true" {
		if ctx.Request.Header.Get("X-Interceptor-Value") != "1" {
			ctx.AbortWithStatus(401)
			return errors.New("X-Interceptor not found")
		}
	}
	return nil
}

func (s Service) CreateUserProfile(c *gin.Context, req *CreateUserProfileRequest) (json.RawMessage, error) {
	return json.RawMessage(req.JsonObject), nil
}

func (s Service) Token(c *gin.Context, req *TokenRequest) (*TokenResponse, error) {
	if req.GrantType != GrantTypeClientCredentials {
		return nil, errors.New("grant type not supported")
	}
	return &TokenResponse{
		AccessToken: "Test",
		ExpiresIn:   1000,
	}, nil
}
