// Code generated by woco, DO NOT EDIT.

package petstore

// AddPetRequest is the request object for (POST /pet)
type AddPetRequest struct {
	NewPet `json:",inline"`
}

// DeletePetRequest is the request object for (DELETE /pet/{petId})
type DeletePetRequest struct {
	PathParams   DeletePetRequestPathParams
	HeaderParams DeletePetRequestHeaderParams
}

type DeletePetRequestPathParams struct {
	// PetId Pet id to delete
	PetId int64 `binding:"required" uri:"petId"`
}

type DeletePetRequestHeaderParams struct {
	APIKey *string `header:"api_key"`
}

// FindPetsByStatusRequest is the request object for (GET /pet/findByStatus)
type FindPetsByStatusRequest struct {
	// Status Status values that need to be considered for filter
	Status []string `binding:"required,dive,oneof=available pending sold" form:"status"`
}

// FindPetsByTagsRequest is the request object for (GET /pet/findByTags)
type FindPetsByTagsRequest struct {
	// Tags Tags to filter by
	Tags []string `binding:"required" form:"tags"`
}

// GetPetByIdRequest is the request object for (GET /pet/{petId})
type GetPetByIdRequest struct {
	// PetId ID of pet to return
	PetId int64 `binding:"required" uri:"petId"`
}

// UpdatePetRequest is the request object for (PUT /pet)
type UpdatePetRequest struct {
	// Pet A pet for sale in the pet store
	Pet `json:",inline"`
}

// UpdatePetWithFormRequest is the request object for (POST /pet/{petId})
type UpdatePetWithFormRequest struct {
	PathParams  UpdatePetWithFormRequestPathParams
	QueryParams UpdatePetWithFormRequestQueryParams
	Body        UpdatePetWithFormRequestBody
}

type UpdatePetWithFormRequestPathParams struct {
	// PetId ID of pet that needs to be updated
	PetId int64 `binding:"required" uri:"petId"`
}

type UpdatePetWithFormRequestQueryParams struct {
	// Timestamp Timestamp of the update
	Timestamp *int64 `form:"timestamp"`
}

type UpdatePetWithFormRequestBody struct {
	// Name Updated name of the pet
	Name string `form:"name"`
	// Status Updated status of the pet
	Status string `form:"status"`
}

// UploadFileRequest is the request object for (POST /pet/{petId}/uploadImage)
type UploadFileRequest struct {
	PathParams UploadFileRequestPathParams
	Body       UploadFileRequestBody
}

type UploadFileRequestPathParams struct {
	// PetId ID of pet to update
	PetId int64 `binding:"required" uri:"petId"`
}

type UploadFileRequestBody struct {
	// AdditionalMetadata Additional data to pass to server
	AdditionalMetadata string `form:"additionalMetadata"`
	// File file to upload
	File []byte `form:"file"`
	Md5  string `form:"md5"`
}
