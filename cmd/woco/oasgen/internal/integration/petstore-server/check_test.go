package petstore

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"reflect"
	"testing"
)

func TestGenerateAfter(t *testing.T) {
	t.Run("checkModel", func(t *testing.T) {
		t.Run("pet", func(t *testing.T) {
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
			assert.EqualValues(t, "labels,omitempty", tag.Field(1).Tag.Get("json"))
			assert.EqualValues(t, "Labels", tag.Field(1).Name)
			assert.EqualValues(t, "petstore.LabelSet", tag.Field(1).Type.String()) // map[string]string
		})
		t.Run("category", func(t *testing.T) {
			categoryS := Category{}
			categoryType := reflect.TypeOf(categoryS)
			assert.EqualValues(t, `id,omitempty`, categoryType.Field(0).Tag.Get("json"))
		})
		t.Run("order", func(t *testing.T) {
			order := Order{}
			orderType := reflect.TypeOf(order)
			orderValue := reflect.ValueOf(order)
			complete := orderType.Field(0)
			assert.EqualValues(t, `complete,omitempty`, complete.Tag.Get("json"))
			assert.EqualValues(t, `bool`, complete.Type.String())
			assert.EqualValues(t, false, orderValue.Field(0).Bool())

			oid, ok := orderType.FieldByName("ID")
			require.True(t, ok)
			assert.EqualValues(t, `id,omitempty`, oid.Tag.Get("json"))
			assert.EqualValues(t, `int64`, oid.Type.String())

			pid, ok := orderType.FieldByName("PetId")
			require.True(t, ok)
			assert.EqualValues(t, `petId,omitempty`, pid.Tag.Get("json"))
			assert.EqualValues(t, `int64`, pid.Type.String())

			quantity, ok := orderType.FieldByName("Quantity")
			require.True(t, ok)
			assert.EqualValues(t, `quantity,omitempty`, quantity.Tag.Get("json"))
			assert.EqualValues(t, `int32`, quantity.Type.String())

			shipDate, ok := orderType.FieldByName("ShipDate")
			require.True(t, ok)
			assert.EqualValues(t, `shipDate,omitempty`, shipDate.Tag.Get("json"))
			assert.EqualValues(t, `time.Time`, shipDate.Type.String())

			orderDate, ok := orderType.FieldByName("OrderDate")
			require.True(t, ok)
			assert.EqualValues(t, `omitempty,ltfield=ShipDate`, orderDate.Tag.Get("binding"))
			assert.EqualValues(t, `time.Time`, orderDate.Type.String())

			status, ok := orderType.FieldByName("Status")
			require.True(t, ok)
			assert.EqualValues(t, `status,omitempty`, status.Tag.Get("json"))
			assert.EqualValues(t, `petstore.OrderStatus`, status.Type.String())

			allof := reflect.TypeOf(NewPet{})
			assert.EqualValues(t, `,inline`, allof.Field(0).Tag.Get("json"))
			assert.EqualValues(t, `*petstore.Pet`, allof.Field(0).Type.String())

			usert := reflect.TypeOf(User{})
			// check binding
			assert.EqualValues(t, `omitempty,email`, usert.Field(0).Tag.Get("binding"))
		})
		t.Run("user", func(t *testing.T) {
			user := User{}
			userType := reflect.TypeOf(user)
			assert.EqualValues(t, `omitempty,email`, userType.Field(0).Tag.Get("binding"))
			assert.EqualValues(t, `email,omitempty`, userType.Field(0).Tag.Get("json"))
			assert.EqualValues(t, `string`, userType.Field(0).Type.String())
			assert.EqualValues(t, `firstName,omitempty`, userType.Field(1).Tag.Get("json"))
			assert.EqualValues(t, `string`, userType.Field(1).Type.String())
			assert.EqualValues(t, `id,omitempty`, userType.Field(2).Tag.Get("json"))
			assert.EqualValues(t, `int64`, userType.Field(2).Type.String())
			assert.EqualValues(t, `lastName,omitempty`, userType.Field(3).Tag.Get("json"))
			assert.EqualValues(t, `string`, userType.Field(3).Type.String())
			assert.EqualValues(t, `password,omitempty`, userType.Field(4).Tag.Get("json"))
			assert.EqualValues(t, `string`, userType.Field(4).Type.String())
			assert.EqualValues(t, `phone,omitempty`, userType.Field(5).Tag.Get("json"))
			assert.EqualValues(t, `string`, userType.Field(5).Type.String())
			assert.EqualValues(t, `user_status,omitempty`, userType.Field(6).Tag.Get("json"))
			assert.EqualValues(t, `int32`, userType.Field(6).Type.String())
			assert.EqualValues(t, `username,omitempty`, userType.Field(7).Tag.Get("json"))
			assert.EqualValues(t, `string`, userType.Field(7).Type.String())
		})
	})
	t.Run("checkModel-allof", func(t *testing.T) {
		newpet := reflect.TypeOf(NewPet{})
		assert.EqualValues(t, `,inline`, newpet.Field(0).Tag.Get("json"))
		assert.EqualValues(t, `*petstore.Pet`, newpet.Field(0).Type.String())
		assert.EqualValues(t, `Timestamp`, newpet.Field(2).Name)

		n1 := reflect.TypeOf(NewPets{&NewPet{}})
		assert.EqualValues(t, "*petstore.NewPet", n1.Elem().String())
	})
	t.Run("checkRequest", func(t *testing.T) {
		t.Run("tag", func(t *testing.T) {
			drHeader := reflect.TypeOf(DeletePetRequest{}.HeaderParams)
			assert.EqualValues(t, `api_key`, drHeader.Field(0).Tag.Get("header"))
			drUri := reflect.TypeOf(DeletePetRequest{}.PathParams)
			assert.EqualValues(t, `petId`, drUri.Field(0).Tag.Get("uri"))
			assert.EqualValues(t, `required`, drUri.Field(0).Tag.Get("binding"))
			urUri := reflect.TypeOf(UpdatePetWithFormRequest{}.PathParams)
			assert.EqualValues(t, `petId`, urUri.Field(0).Tag.Get("uri"))
			assert.EqualValues(t, `required`, urUri.Field(0).Tag.Get("binding"))
			urBody := reflect.TypeOf(UpdatePetWithFormRequest{}.Body)
			assert.EqualValues(t, `name`, urBody.Field(0).Tag.Get("form"))
			assert.EqualValues(t, `status`, urBody.Field(1).Tag.Get("form"))
			upBody := reflect.TypeOf(UpdateUserRequest{})
			assert.EqualValuesf(t, "", upBody.Field(1).Tag.Get("json"), "request body should not have json tag")
			assert.Equal(t, upBody.Field(1).Type.Kind(), reflect.Struct)

			usBody := reflect.TypeOf(LoginUserRequest{})
			assert.EqualValues(t, `true`, usBody.Field(1).Tag.Get("password"))

			arrayBody := reflect.TypeOf(CreateUsersWithArrayInputRequest{})
			assert.EqualValues(t, `UserArray`, arrayBody.Field(0).Name)
			assert.EqualValuesf(t, "", arrayBody.Field(0).Tag.Get("binding"), "array body should not have binding tag")
			assert.EqualValues(t, `[]*petstore.User`, arrayBody.Field(0).Type.String())

			prHeader := reflect.TypeOf(DeletePetRequestHeaderParams{})
			assert.EqualValues(t, `api_key`, prHeader.Field(0).Tag.Get("header"))
			assert.EqualValues(t, `*string`, prHeader.Field(0).Type.String(), "parameter not require should be pointer")

			upParams := reflect.TypeOf(UpdatePetWithFormRequestQueryParams{})
			assert.EqualValues(t, `timestamp`, upParams.Field(0).Tag.Get("form"))
			assert.EqualValues(t, `*int64`, upParams.Field(0).Type.String(), "parameter not require should be pointer")
		})
		t.Run("body", func(t *testing.T) {
			apr := reflect.TypeOf(AddPetRequest{})
			assert.EqualValues(t, `NewPet`, apr.Field(0).Name)
			assert.EqualValues(t, `petstore.NewPet`, apr.Field(0).Type.String())

			b1 := reflect.TypeOf(UpdateUserRequestBody{})
			assert.EqualValues(t, `petstore.User`, b1.Field(0).Type.String(), "RequestBody required")

			b2 := reflect.TypeOf(CreateUserProfileRequest{})
			assert.EqualValues(t, `petstore.JsonObject`, b2.Field(0).Type.String(), "JsonObject")
		})
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
