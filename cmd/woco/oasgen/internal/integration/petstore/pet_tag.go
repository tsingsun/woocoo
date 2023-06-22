// Code generated by woco, DO NOT EDIT.

package petstore

type AddPetRequest struct {
	Body NewPet
}

type DeletePetRequest struct {
	UriParams    DeletePetRequestUriParams
	HeaderParams DeletePetRequestHeaderParams
}

type DeletePetRequestUriParams struct {
	PetId int64 `binding:"required" uri:"petId"`
}
type DeletePetRequestHeaderParams struct {
	APIKey string `header:"api_key"`
}

type FindPetsByStatusRequest struct {
	Body []string `binding:"required" form:"status"`
}

type FindPetsByTagsRequest struct {
	Body []string `binding:"required" form:"tags"`
}

type GetPetByIdRequest struct {
	UriParams GetPetByIdRequestUriParams
}

type GetPetByIdRequestUriParams struct {
	PetId int64 `binding:"required" uri:"petId"`
}

type UpdatePetRequest struct {
	Body Pet
}

type UpdatePetWithFormRequest struct {
	UriParams UpdatePetWithFormRequestUriParams
	Body      UpdatePetWithFormRequestBody
}

type UpdatePetWithFormRequestUriParams struct {
	PetId int64 `binding:"required" uri:"petId"`
}
type UpdatePetWithFormRequestBody struct {
	Name   string `form:"name"`
	Status string `form:"status"`
}

type UploadFileRequest struct {
	UriParams UploadFileRequestUriParams
	Body      UploadFileRequestBody
}

type UploadFileRequestUriParams struct {
	PetId int64 `binding:"required" uri:"petId"`
}
type UploadFileRequestBody struct {
	AdditionalMetadata string `form:"additionalMetadata"`
	File               []byte `form:"file"`
}
