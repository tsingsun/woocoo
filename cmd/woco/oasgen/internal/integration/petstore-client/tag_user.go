// Code generated by woco, DO NOT EDIT.

package client

// CreateUserRequest is the request object for (POST /user)
type CreateUserRequest struct {
	// CreateUserRequestBody A User who is purchasing from the pet store
	User `json:",inline"`
}

// CreateUserResponse Create user response
type CreateUserResponse struct {
	UserID string `json:"userID,omitempty"`
}

// CreateUserProfileRequest is the request object for (POST /user/profile)
type CreateUserProfileRequest struct {
	// CreateUserProfileRequestBody A JSON object
	JsonObject `json:",inline"`
}

// CreateUsersWithArrayInputRequest is the request object for (POST /user/createWithArray)
type CreateUsersWithArrayInputRequest struct {
	UserArray []*User
}

// CreateUsersWithListInputRequest is the request object for (POST /user/createWithList)
type CreateUsersWithListInputRequest struct {
	UserArray []*User
}

// DeleteUserRequest is the request object for (DELETE /user/{username})
type DeleteUserRequest struct {
	// Username The name that needs to be deleted
	Username string `binding:"required" uri:"username"`
}

// GetUserByNameRequest is the request object for (GET /user/{username})
type GetUserByNameRequest struct {
	// Username The name that needs to be fetched. Use user1 for testing.
	Username string `binding:"required" uri:"username"`
}

// LoginUserRequest is the request object for (GET /user/login)
type LoginUserRequest struct {
	// Username The user name for login
	Username string `binding:"required,regex=oas_pattern_0" form:"username"`
	// Password The password for login in clear text
	Password string `binding:"required" form:"password" password:"true"`
}

// UpdateUserRequest is the request object for (PUT /user/{username})
type UpdateUserRequest struct {
	PathParams UpdateUserRequestPathParams
	Body       UpdateUserRequestBody
}

type UpdateUserRequestPathParams struct {
	// Username name that need to be deleted
	Username string `binding:"required" uri:"username"`
}

type UpdateUserRequestBody struct {
	// UpdateUserRequestBody A User who is purchasing from the pet store
	User `json:",inline"`
}
