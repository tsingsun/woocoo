package main

import (
	"bytes"
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
	server.RegisterValidator()
	server.RegisterUserHandlers(&router.RouterGroup, imp)
	server.RegisterStoreHandlers(&router.RouterGroup, imp)
	server.RegisterPetHandlers(&router.RouterGroup, imp)
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
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, r)
	assert.Equal(s.T(), 400, w.Code)

	r = httptest.NewRequest("GET", "/user/login?username=abc&password=b", nil)
	r.Header.Set("accept", binding.MIMEXML)
	w = httptest.NewRecorder()
	s.Router.ServeHTTP(w, r)
	assert.Equal(s.T(), 200, w.Code)
	assert.Equal(s.T(), `<string>ok</string>`, w.Body.String())
}

func (s *ginTestSuite) TestGetOrderById() {
	r := httptest.NewRequest("GET", "/store/order/1", nil)
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, r)
	assert.Equal(s.T(), 200, w.Code)

	r = httptest.NewRequest("GET", "/store/order/6", nil)
	w = httptest.NewRecorder()
	s.Router.ServeHTTP(w, r)
	assert.Equal(s.T(), 400, w.Code)

}

func (s *ginTestSuite) TestUpdateUser() {
	t := s.T()
	t.Run("email validator", func(t *testing.T) {
		bf := bytes.NewBufferString(`{"id":1,"email":"email"}`)
		r := httptest.NewRequest("PUT", "/user/1", bf)
		w := httptest.NewRecorder()
		s.Router.ServeHTTP(w, r)
		assert.Equal(t, 400, w.Code)
	})
	t.Run("email omitempty", func(t *testing.T) {
		bf := bytes.NewBufferString(`{"id":1}`)
		r := httptest.NewRequest("PUT", "/user/1", bf)
		w := httptest.NewRecorder()
		s.Router.ServeHTTP(w, r)
		assert.Equal(s.T(), 400, w.Code)
	})
	t.Run("email ok", func(t *testing.T) {
		bf := bytes.NewBufferString(`{"id":1,"email":"test@woocoo.net"}`)
		r := httptest.NewRequest("PUT", "/user/1", bf)
		w := httptest.NewRecorder()
		s.Router.ServeHTTP(w, r)
		assert.Equal(s.T(), 400, w.Code)
	})
}

func TestGinTestSuite(t *testing.T) {
	suite.Run(t, new(ginTestSuite))
}
