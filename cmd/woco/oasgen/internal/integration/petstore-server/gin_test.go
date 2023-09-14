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
	"strings"
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
	imp := &Service{}
	server.RegisterValidator()
	server.RegisterUserHandlers(&router.RouterGroup, imp)
	server.RegisterStoreHandlers(&router.RouterGroup, imp)
	server.RegisterPetHandlers(&router.RouterGroup, imp)
	s.Router = router
}

func (s *ginTestSuite) TestAddPet() {
	sr := strings.NewReader(`{"id":1,"name":"name","photoUrls":["localhost"],"owner":{"email":"owner@example.com"},"timestamp":"2023-01-01T00:00:00Z"}`)
	r := httptest.NewRequest("POST", "/pet", sr)
	r.Header.Add("Content-Type", binding.MIMEJSON)
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, r)
	assert.Equal(s.T(), 200, w.Code)
	assert.Contains(s.T(), w.Body.String(), `"id":1,"name":"name"`)
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
	assert.JSONEq(s.T(), `{"errors":[{"code":500,"message":"UpdatePetWithForm Error"}]}`, w.Body.String())
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
	s.Run("email format error", func() {
		bf := bytes.NewBufferString(`{"id":1,"email":"email"}`)
		r := httptest.NewRequest("PUT", "/user/1", bf)
		r.Header.Add("Content-Type", binding.MIMEJSON)
		w := httptest.NewRecorder()
		s.Router.ServeHTTP(w, r)
		s.Equal(400, w.Code)
	})
	s.Run("email omitempty", func() {
		bf := bytes.NewBufferString(`{"id":1}`)
		r := httptest.NewRequest("PUT", "/user/1", bf)
		r.Header.Add("Content-Type", binding.MIMEJSON)
		w := httptest.NewRecorder()
		s.Router.ServeHTTP(w, r)
		s.Equal(200, w.Code)
	})
	s.Run("email ok", func() {
		bf := bytes.NewBufferString(`{"id":1,"email":"test@woocoo.net"}`)
		r := httptest.NewRequest("PUT", "/user/1", bf)
		r.Header.Add("Content-Type", binding.MIMEJSON)
		w := httptest.NewRecorder()
		s.Router.ServeHTTP(w, r)
		s.Equal(200, w.Code)
	})
}

func (s *ginTestSuite) TestFindPetsByStatusRequest() {
	s.Run("empty status", func() {
		r := httptest.NewRequest("GET", "/pet/findByStatus?status=available", nil)
		w := httptest.NewRecorder()
		s.Router.ServeHTTP(w, r)
		s.Contains(w.Body.String(), "available")
		s.Equal(200, w.Code)
	})
}

func (s *ginTestSuite) TestPostOrder() {
	t := s.T()
	t.Run("empty ltfield validate", func(t *testing.T) {
		bf := bytes.NewBufferString(`{"id":1,"status":"placed"}`)
		r := httptest.NewRequest("POST", "/store/order", bf)
		r.Header.Set("Content-Type", binding.MIMEJSON)
		w := httptest.NewRecorder()
		s.Router.ServeHTTP(w, r)
		assert.Equal(t, 200, w.Code)
	})
	t.Run("wrong time", func(t *testing.T) {
		bf := bytes.NewBufferString(`{"id":1,"status":"placed","shipDate":"2006-01-02"}`)
		r := httptest.NewRequest("POST", "/store/order", bf)
		r.Header.Set("Content-Type", binding.MIMEJSON)
		w := httptest.NewRecorder()
		s.Router.ServeHTTP(w, r)
		assert.Equal(t, 400, w.Code)
		assert.Contains(t, w.Header().Get("Content-Type"), binding.MIMEJSON)
	})
	t.Run("with time", func(t *testing.T) {
		bf := bytes.NewBufferString(`{"id":1,"status":"placed","shipDate":"2006-01-02T15:04:05Z","orderDate":"2005-01-02T15:04:05Z"}`)
		r := httptest.NewRequest("POST", "/store/order", bf)
		r.Header.Set("Content-Type", binding.MIMEJSON)
		w := httptest.NewRecorder()
		s.Router.ServeHTTP(w, r)
		assert.Equal(t, 200, w.Code)
	})
	t.Run("with time validate failure", func(t *testing.T) {
		bf := bytes.NewBufferString(`{"id":1,"status":"placed","shipDate":"2006-01-02T15:04:05Z","orderDate":"2007-01-02T15:04:05Z"}`)
		r := httptest.NewRequest("POST", "/store/order", bf)
		r.Header.Set("Content-Type", binding.MIMEJSON)
		w := httptest.NewRecorder()
		s.Router.ServeHTTP(w, r)
		assert.Equal(t, 400, w.Code)
	})
}

func TestGinTestSuite(t *testing.T) {
	suite.Run(t, new(ginTestSuite))
}
