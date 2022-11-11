package petstore

import (
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestGenerateAfter(t *testing.T) {
	t.Run("checkModel", func(t *testing.T) {
		apiRes := reflect.TypeOf(ApiResponse{})
		// check struct tag
		assert.EqualValues(t, "Code", apiRes.Field(0).Name)
		assert.EqualValues(t, `code,omitempty`, apiRes.Field(0).Tag.Get("json"))

		pet := Pet{}
		petType := reflect.TypeOf(pet)
		petValue := reflect.ValueOf(pet)
		assert.EqualValues(t, `category,omitempty`, petType.Field(0).Tag.Get("json"))
		assert.EqualValues(t, `*petstore.Category`, petType.Field(0).Type.String())
		assert.True(t, petValue.Field(0).IsNil())
		assert.EqualValues(t, "ID", petType.Field(1).Name)
		assert.EqualValues(t, `id,omitempty`, petType.Field(1).Tag.Get("json"))
		assert.EqualValues(t, `int64`, petType.Field(1).Type.String())
		assert.EqualValues(t, "Name", petType.Field(2).Name)
		assert.EqualValues(t, `name`, petType.Field(2).Tag.Get("json"))
		assert.EqualValues(t, `string`, petType.Field(2).Type.String())
		assert.EqualValues(t, "PhotoUrls", petType.Field(3).Name)
		assert.EqualValues(t, `photoUrls`, petType.Field(3).Tag.Get("json"))
		assert.EqualValues(t, `[]string`, petType.Field(3).Type.String())
		assert.EqualValues(t, "Tags", petType.Field(5).Name)
		assert.EqualValues(t, `tags,omitempty`, petType.Field(5).Tag.Get("json"))
		assert.EqualValues(t, `[]petstore.Tag`, petType.Field(5).Type.String())

		order := Order{}
		orderType := reflect.TypeOf(order)
		orderValue := reflect.ValueOf(order)
		assert.EqualValues(t, `complete,omitempty`, orderType.Field(0).Tag.Get("json"))
		assert.EqualValues(t, `bool`, orderType.Field(0).Type.String())
		assert.EqualValues(t, false, orderValue.Field(0).Bool())
		assert.EqualValues(t, `id,omitempty`, orderType.Field(1).Tag.Get("json"))
		assert.EqualValues(t, `int64`, orderType.Field(1).Type.String())
		assert.EqualValues(t, `petId,omitempty`, orderType.Field(2).Tag.Get("json"))
		assert.EqualValues(t, `int64`, orderType.Field(2).Type.String())
		assert.EqualValues(t, `quantity,omitempty`, orderType.Field(3).Tag.Get("json"))
		assert.EqualValues(t, `int32`, orderType.Field(3).Type.String())
		assert.EqualValues(t, `shipDate,omitempty`, orderType.Field(4).Tag.Get("json"))
		assert.EqualValues(t, `time.Time`, orderType.Field(4).Type.String())
		assert.EqualValues(t, `status,omitempty`, orderType.Field(5).Tag.Get("json"))
		assert.EqualValues(t, `string`, orderType.Field(5).Type.String())
	})
	t.Run("checkRequest", func(t *testing.T) {
		dr := reflect.TypeOf(DeletePetRequest{})
		assert.EqualValues(t, `api_key`, dr.Field(0).Tag.Get("header"))
		assert.EqualValues(t, `petId`, dr.Field(1).Tag.Get("uri"))
		assert.EqualValues(t, `required`, dr.Field(1).Tag.Get("binding"))
		ur := reflect.TypeOf(UpdatePetWithFormRequest{})
		assert.EqualValues(t, `petId`, ur.Field(0).Tag.Get("uri"))
		assert.EqualValues(t, `required`, ur.Field(0).Tag.Get("binding"))
		assert.EqualValues(t, `name`, ur.Field(1).Tag.Get("form"))
		assert.EqualValues(t, `status`, ur.Field(2).Tag.Get("form"))

	})
}
