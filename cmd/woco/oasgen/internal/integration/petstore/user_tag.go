// Code generated by woco, DO NOT EDIT.

package petstore

type CreateUserRequest struct {
	Body CreateUserRequestBody
}
type CreateUserRequestBody struct {
	User User `binding:"required" json:"user"`
}

type DeleteUserRequest struct {
	UriParams DeleteUserRequestUriParams
}
type DeleteUserRequestUriParams struct {
	Username string `binding:"required" uri:"username"`
}

type GetUserByNameRequest struct {
	UriParams GetUserByNameRequestUriParams
}
type GetUserByNameRequestUriParams struct {
	Username string `binding:"required" uri:"username"`
}

type LoginUserRequest struct {
	Body LoginUserRequestBody
}
type LoginUserRequestBody struct {
	Username string `binding:"regex=oas_pattern_0,required" form:"username"`
	Password string `binding:"required" form:"password" password:"true"`
}

type UpdateUserRequest struct {
	UriParams UpdateUserRequestUriParams
	Body      UpdateUserRequestBody
}
type UpdateUserRequestUriParams struct {
	Username string `binding:"required" uri:"username"`
}
type UpdateUserRequestBody struct {
	User User `binding:"required" json:"user"`
}