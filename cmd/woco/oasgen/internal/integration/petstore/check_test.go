package petstore

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
)

func TestGenerateAfter(t *testing.T) {
	t.Run("checkModel", func(t *testing.T) {
		pet := Pet{}
		petType := reflect.TypeOf(pet)
		petValue := reflect.ValueOf(pet)

		category := petType.Field(0)
		assert.EqualValues(t, `category,omitempty`, category.Tag.Get("json"))
		assert.EqualValues(t, `Category`, category.Tag.Get("xml"))
		assert.EqualValues(t, `*petstore.Category`, category.Type.String())
		assert.True(t, petValue.Field(0).IsNil())

		id := petType.Field(1)
		assert.EqualValues(t, "ID", id.Name)
		assert.EqualValues(t, `id,omitempty`, id.Tag.Get("json"))
		assert.EqualValues(t, `int64`, id.Type.String())

		name := petType.Field(2)
		assert.EqualValues(t, "Name", name.Name)
		assert.EqualValues(t, `name`, name.Tag.Get("json"))
		assert.EqualValues(t, `string`, name.Type.String())

		photoUrls := petType.Field(3)
		assert.EqualValues(t, "PhotoUrls", photoUrls.Name)
		assert.EqualValues(t, `photoUrls`, photoUrls.Tag.Get("json"))
		assert.EqualValues(t, `photoUrl`, photoUrls.Tag.Get("xml"))
		assert.EqualValues(t, `[]string`, photoUrls.Type.String())

		tags := petType.Field(5)
		assert.EqualValues(t, "Tags", tags.Name)
		assert.EqualValues(t, `tags,omitempty`, tags.Tag.Get("json"))
		assert.EqualValues(t, `tag`, tags.Tag.Get("xml"))
		assert.EqualValues(t, `[]*petstore.Tag`, tags.Type.String())

		tag := reflect.TypeOf(Tag{})
		assert.EqualValues(t, "id,omitempty", tag.Field(0).Tag.Get("json"))
		assert.EqualValues(t, "labelSet,omitempty", tag.Field(1).Tag.Get("json"))
		assert.EqualValues(t, "Labels", tag.Field(1).Name)                     // map[string]string
		assert.EqualValues(t, "petstore.LabelSet", tag.Field(1).Type.String()) // map[string]string

		categoryS := Category{}
		categoryType := reflect.TypeOf(categoryS)
		assert.EqualValues(t, `id,omitempty`, categoryType.Field(0).Tag.Get("json"))

		order := Order{}
		orderType := reflect.TypeOf(order)
		orderValue := reflect.ValueOf(order)
		complete := orderType.Field(0)
		assert.EqualValues(t, `complete,omitempty`, complete.Tag.Get("json"))
		assert.EqualValues(t, `bool`, complete.Type.String())
		assert.EqualValues(t, false, orderValue.Field(0).Bool())

		oid := orderType.Field(1)
		assert.EqualValues(t, `id,omitempty`, oid.Tag.Get("json"))
		assert.EqualValues(t, `int64`, oid.Type.String())

		pid := orderType.Field(2)
		assert.EqualValues(t, `petId,omitempty`, pid.Tag.Get("json"))
		assert.EqualValues(t, `int64`, pid.Type.String())

		quantity := orderType.Field(3)
		assert.EqualValues(t, `quantity,omitempty`, quantity.Tag.Get("json"))
		assert.EqualValues(t, `int32`, quantity.Type.String())

		shipDate := orderType.Field(4)
		assert.EqualValues(t, `shipDate,omitempty`, shipDate.Tag.Get("json"))
		assert.EqualValues(t, `time.Time`, shipDate.Type.String())

		status := orderType.Field(5)
		assert.EqualValues(t, `status,omitempty`, status.Tag.Get("json"))
		assert.EqualValues(t, `string`, status.Type.String())

		allof := reflect.TypeOf(NewPet{})
		assert.EqualValues(t, `,inline`, allof.Field(0).Tag.Get("json"))
		assert.EqualValues(t, `*petstore.Pet`, allof.Field(0).Type.String())
	})
	t.Run("checkRequest", func(t *testing.T) {
		drHeader := reflect.TypeOf(DeletePetRequest{}.HeaderParams)
		assert.EqualValues(t, `api_key`, drHeader.Field(0).Tag.Get("header"))
		drUri := reflect.TypeOf(DeletePetRequest{}.UriParams)
		assert.EqualValues(t, `petId`, drUri.Field(0).Tag.Get("uri"))
		assert.EqualValues(t, `required`, drUri.Field(0).Tag.Get("binding"))
		urUri := reflect.TypeOf(UpdatePetWithFormRequest{}.UriParams)
		assert.EqualValues(t, `petId`, urUri.Field(0).Tag.Get("uri"))
		assert.EqualValues(t, `required`, urUri.Field(0).Tag.Get("binding"))
		urBody := reflect.TypeOf(UpdatePetWithFormRequest{}.Body)
		assert.EqualValues(t, `name`, urBody.Field(0).Tag.Get("form"))
		assert.EqualValues(t, `status`, urBody.Field(1).Tag.Get("form"))
		upBOdy := reflect.TypeOf(UpdateUserRequest{}.Body)
		assert.EqualValues(t, `user`, upBOdy.Field(0).Tag.Get("json"))
		assert.NotEqual(t, upBOdy.Field(0).Type.Kind(), reflect.Ptr)

		usBody := reflect.TypeOf(LoginUserRequest{}.Body)
		assert.EqualValues(t, `true`, usBody.Field(1).Tag.Get("password"))

		arrayBody := reflect.TypeOf(CreateUsersWithArrayInputRequest{}.Body)
		assert.EqualValues(t, `required`, arrayBody.Field(0).Tag.Get("binding"))
		assert.EqualValues(t, `[]*petstore.User`, arrayBody.Field(0).Type.String())
	})
	t.Run("checkResponse", func(t *testing.T) {
		res := reflect.TypeOf(UnimplementedPetServer{})
		// response from reference
		md, ok := res.MethodByName("FindPetsByTags")
		require.True(t, ok)
		/// get md return type,check if it is a special type
		assert.EqualValues(t, `petstore.Pets`, md.Type.Out(0).String())

		md, ok = res.MethodByName("FindPetsByStatus")
		require.True(t, ok)
		assert.EqualValuesf(t, `[]*petstore.Pet`, md.Type.Out(0).String(), "array response type should be petstore.Pets slice")
	})
}
