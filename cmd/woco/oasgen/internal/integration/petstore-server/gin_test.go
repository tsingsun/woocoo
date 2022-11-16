package main

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"github.com/tsingsun/woocoo/cmd/woco/oasgen/internal/integration/petstore/server"
	"github.com/tsingsun/woocoo/web/handler"
	"net/http/httptest"
	"testing"
)

type ginTestSuite struct {
	suite.Suite
	Router *gin.Engine
}

func (s *ginTestSuite) SetupSuite() {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	router.Use(handler.ErrorHandle().ApplyFunc(nil))
	imp := &Server{}
	server.RegisterUserHandlers(router, imp)
	server.RegisterStoreHandlers(router, imp)
	server.RegisterPetHandlers(router, imp)
	s.Router = router
}

func (s *ginTestSuite) TestAddPet() {
	r := httptest.NewRequest("POST", "/pet", nil)
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, r)
}

func (s *ginTestSuite) TestDeletePet() {
	r := httptest.NewRequest("DELETE", "/pet/1", nil)
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, r)
}

func (s *ginTestSuite) TestGetPetById() {
	r := httptest.NewRequest("GET", "/pet/1", nil)
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, r)
}

func (s *ginTestSuite) TestUpdatePetWithForm() {
	r := httptest.NewRequest("POST", "/pet/1", nil)
	r.Form = map[string][]string{}
	r.Form.Add("name", "name")
	r.Form.Add("status", "status")
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, r)
	assert.Equal(s.T(), 500, w.Code)
	assert.JSONEq(s.T(), `{"errors":[{"message":"UpdatePetWithForm Error"}]}`, w.Body.String())
}

func (s *ginTestSuite) TestLoginUser() {
	r := httptest.NewRequest("GET", "/user/login?username=a&password=b", nil)
	r.Header.Set("accept", binding.MIMEXML)
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, r)
	assert.Equal(s.T(), 200, w.Code)
	assert.Equal(s.T(), `<string>ok</string>`, w.Body.String())
}

func TestGinTestSuite(t *testing.T) {
	suite.Run(t, new(ginTestSuite))
}
