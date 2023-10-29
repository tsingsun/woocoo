// Code generated by woco, DO NOT EDIT.

package petstore

// CreateUserRequest Created user object
type CreateUserRequest struct {
	// CreateUserRequestBody A User who is purchasing from the pet store
	User `json:",inline"`
}

type CreateUserResponse struct {
	UserID string `json:"userID,omitempty"`
}

// CreateUserProfileRequest Created user object
type CreateUserProfileRequest struct {
	// CreateUserProfileRequestBody A JSON object
	JsonObject `json:",inline"`
}

// CreateUsersWithArrayInputRequest List of user object
type CreateUsersWithArrayInputRequest struct {
	UserArray []*User
}

// CreateUsersWithListInputRequest List of user object
type CreateUsersWithListInputRequest struct {
	UserArray []*User
}

type DeleteUserRequest struct {
	// Username The name that needs to be deleted
	Username string `binding:"required" uri:"username"`
}

type GetUserByNameRequest struct {
	// Username The name that needs to be fetched. Use user1 for testing.
	Username string `binding:"required" uri:"username"`
}

type LoginUserRequest struct {
	// Username The user name for login
	Username string `binding:"regex=oas_pattern_0,required" form:"username"`
	// Password The password for login in clear text
	Password string `binding:"required" form:"password" password:"true"`
}

// UpdateUserRequest Updated user object
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
