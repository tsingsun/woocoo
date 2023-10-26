package client

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/suite"
	"github.com/tsingsun/woocoo/cmd/woco/oasgen/internal/integration/petstore-server"
	"github.com/tsingsun/woocoo/pkg/conf"
	"github.com/tsingsun/woocoo/web/handler"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func init() {
	gin.SetMode(gin.ReleaseMode)
}

type clientTest struct {
	suite.Suite
	Router     *gin.Engine
	cfg        *Config
	client     *APIClient
	mockServer *httptest.Server
}

func (ct *clientTest) SetupSuite() {
	router := gin.Default()
	router.Use(handler.ErrorHandle().ApplyFunc(conf.New()))
	imp := &petstore.Service{}
	petstore.RegisterValidator()
	petstore.RegisterUserHandlers(&router.RouterGroup, imp)
	petstore.RegisterStoreHandlers(&router.RouterGroup, imp)
	petstore.RegisterPetHandlers(&router.RouterGroup, imp)
	ct.Router = router
	// Mock server which always responds 500.
	ct.mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		router.ServeHTTP(w, r)
	}))
	ct.cfg = &Config{
		BasePath:  ct.mockServer.URL,
		UserAgent: "oasgen",
		Headers: map[string]string{
			"X-Tenant-Id": "1",
		},
	}
	ct.client = NewAPIClient(ct.cfg)
}

func (ct *clientTest) TearDownSuite() {
	if ct.mockServer != nil {
		ct.mockServer.Close()
	}
}

func TestNewAPIClient(t *testing.T) {
	suite.Run(t, new(clientTest))
}

func (ct *clientTest) TestAddPet() {
	pet, res, err := ct.client.PetAPI.AddPet(context.Background(), &AddPetRequest{
		NewPet: NewPet{
			Pet: &Pet{
				ID:        1,
				Name:      "test",
				PhotoUrls: []string{"https://github.com"},
			},
			Timestamp: time.Now(),
		},
	})
	ct.Require().NoError(err)
	ct.EqualValues(1, pet.ID)
	ct.Equal(200, res.StatusCode)
}

func (ct *clientTest) TestFindPetsByTags() {
	pets, res, err := ct.client.PetAPI.FindPetsByTags(context.Background(), &FindPetsByTagsRequest{
		Tags: []string{"test"},
	})
	ct.Require().NoError(err)
	ct.Require().Equal(1, len(pets))
	ct.Require().Equal("dog", pets[0].Name)
	ct.Equal(200, res.StatusCode)
}

func (ct *clientTest) TestFindPetsByStatus() {
	pets, res, err := ct.client.PetAPI.FindPetsByStatus(context.Background(), &FindPetsByStatusRequest{
		Status: []string{"available"},
	})
	ct.Require().NoError(err)
	ct.Require().Equal(1, len(pets))
	ct.Require().Equal("dog", pets[0].Name)
	ct.Equal(200, res.StatusCode)
}

func (ct *clientTest) TestUpdatePetWithForm() {
	resp, err := ct.client.PetAPI.UpdatePetWithForm(context.Background(), &UpdatePetWithFormRequest{
		PathParams: UpdatePetWithFormRequestPathParams{
			PetId: 1,
		},
		Body: UpdatePetWithFormRequestBody{
			Name: "test",
		},
	})
	ct.Require().Error(err)
	ct.Equal(500, resp.StatusCode)
}

func (ct *clientTest) TestLoginUser() {
	token, resp, err := ct.client.UserAPI.LoginUser(context.Background(), &LoginUserRequest{
		Username: "admin",
		Password: "admin",
	},
	)
	ct.Require().NoError(err)
	ct.Require().Equal("ok", token)
	ct.Equal(200, resp.StatusCode)
}

func (ct *clientTest) TestGetInventory() {
	inv, resp, err := ct.client.StoreAPI.GetInventory(context.Background())
	ct.Require().NoError(err)
	ct.Require().EqualValues(1, inv["available"])
	ct.Equal(200, resp.StatusCode)
}
