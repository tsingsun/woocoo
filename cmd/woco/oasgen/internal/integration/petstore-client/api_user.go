// Code generated by woco, DO NOT EDIT.

package client

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
)

type UserAPI api

// (POST /user)
func (a *UserAPI) CreateUser(ctx context.Context, req *CreateUserRequest) (ret *CreateUserResponse, resp *http.Response, err error) {
	var (
		contentType string
		body        any
	)
	path := "/user"
	contentType = selectHeaderContentType([]string{"application/json"})
	body = req

	request, err := a.client.prepareRequest("POST", a.client.cfg.BasePath+path, contentType, body)
	if err != nil {
		return
	}
	accept := selectHeaderAccept([]string{"application/json"})
	request.Header.Set("Accept", accept)
	resp, err = a.client.Do(request)
	if err != nil {
		return
	}
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	if resp.StatusCode == http.StatusOK {
		ret = new(CreateUserResponse)
		err = a.client.decode(respBody, ret, resp.Header.Get("Content-Type"))
		if err == nil {
			return
		}
	} else if resp.StatusCode >= 300 {
		err = errors.New(string(respBody))
	}

	return
}

// (POST /user/profile)
func (a *UserAPI) CreateUserProfile(ctx context.Context, req *CreateUserProfileRequest) (ret json.RawMessage, resp *http.Response, err error) {
	var (
		contentType string
		body        any
	)
	path := "/user/profile"
	contentType = selectHeaderContentType([]string{"application/json"})
	body = req

	request, err := a.client.prepareRequest("POST", a.client.cfg.BasePath+path, contentType, body)
	if err != nil {
		return
	}
	accept := selectHeaderAccept([]string{"application/json"})
	request.Header.Set("Accept", accept)
	resp, err = a.client.Do(request)
	if err != nil {
		return
	}
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	if resp.StatusCode == http.StatusOK {
		err = a.client.decode(respBody, &ret, resp.Header.Get("Content-Type"))
		if err == nil {
			return
		}
	} else if resp.StatusCode >= 300 {
		err = errors.New(string(respBody))
	}

	return
}

// (POST /user/createWithArray)
func (a *UserAPI) CreateUsersWithArrayInput(ctx context.Context, req *CreateUsersWithArrayInputRequest) (resp *http.Response, err error) {
	var (
		contentType string
		body        any
	)
	path := "/user/createWithArray"
	contentType = selectHeaderContentType([]string{"application/json"})
	body = req

	request, err := a.client.prepareRequest("POST", a.client.cfg.BasePath+path, contentType, body)
	if err != nil {
		return
	}
	resp, err = a.client.Do(request)
	if err != nil {
		return
	}
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	if resp.StatusCode == http.StatusOK {
		return
	} else if resp.StatusCode >= 300 {
		err = errors.New(string(respBody))
	}

	return
}

// (POST /user/createWithList)
func (a *UserAPI) CreateUsersWithListInput(ctx context.Context, req *CreateUsersWithListInputRequest) (resp *http.Response, err error) {
	var (
		contentType string
		body        any
	)
	path := "/user/createWithList"
	contentType = selectHeaderContentType([]string{"application/json"})
	body = req

	request, err := a.client.prepareRequest("POST", a.client.cfg.BasePath+path, contentType, body)
	if err != nil {
		return
	}
	resp, err = a.client.Do(request)
	if err != nil {
		return
	}
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	if resp.StatusCode == http.StatusOK {
		return
	} else if resp.StatusCode >= 300 {
		err = errors.New(string(respBody))
	}

	return
}

// (DELETE /user/{username})
func (a *UserAPI) DeleteUser(ctx context.Context, req *DeleteUserRequest) (resp *http.Response, err error) {
	var (
		contentType string
		body        any
	)
	path := "/user/{username}"
	path = path[:6] + req.Username + path[6+10:]

	request, err := a.client.prepareRequest("DELETE", a.client.cfg.BasePath+path, contentType, body)
	if err != nil {
		return
	}
	resp, err = a.client.Do(request)
	if err != nil {
		return
	}
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	if resp.StatusCode == http.StatusOK {
		return
	} else if resp.StatusCode >= 300 {
		err = errors.New(string(respBody))
	}

	return
}

// (GET /user/{username})
func (a *UserAPI) GetUserByName(ctx context.Context, req *GetUserByNameRequest) (ret *User, resp *http.Response, err error) {
	var (
		contentType string
		body        any
	)
	path := "/user/{username}"
	path = path[:6] + req.Username + path[6+10:]

	request, err := a.client.prepareRequest("GET", a.client.cfg.BasePath+path, contentType, body)
	if err != nil {
		return
	}
	accept := selectHeaderAccept([]string{"application/json", "application/xml"})
	request.Header.Set("Accept", accept)
	resp, err = a.client.Do(request)
	if err != nil {
		return
	}
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	if resp.StatusCode == http.StatusOK {
		ret = new(User)
		err = a.client.decode(respBody, ret, resp.Header.Get("Content-Type"))
		if err == nil {
			return
		}
	} else if resp.StatusCode >= 300 {
		err = errors.New(string(respBody))
	}

	return
}

// (GET /user/login)
func (a *UserAPI) LoginUser(ctx context.Context, req *LoginUserRequest) (ret string, resp *http.Response, err error) {
	var (
		contentType string
		body        any
	)
	path := "/user/login"
	queryParams := url.Values{}
	queryParams.Add("username", req.Username)
	queryParams.Add("password", req.Password)

	request, err := a.client.prepareRequest("GET", a.client.cfg.BasePath+path, contentType, body)
	if err != nil {
		return
	}
	request.URL.RawQuery = queryParams.Encode()
	accept := selectHeaderAccept([]string{"application/json", "application/xml"})
	request.Header.Set("Accept", accept)
	resp, err = a.client.Do(request)
	if err != nil {
		return
	}
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	if resp.StatusCode == http.StatusOK {
		err = a.client.decode(respBody, &ret, resp.Header.Get("Content-Type"))
		if err == nil {
			return
		}
	} else if resp.StatusCode >= 300 {
		err = errors.New(string(respBody))
	}

	return
}

// (GET /user/logout)
func (a *UserAPI) LogoutUser(ctx context.Context) (resp *http.Response, err error) {
	var (
		contentType string
		body        any
	)
	path := "/user/logout"

	request, err := a.client.prepareRequest("GET", a.client.cfg.BasePath+path, contentType, body)
	if err != nil {
		return
	}
	resp, err = a.client.Do(request)
	if err != nil {
		return
	}
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	if resp.StatusCode == http.StatusOK {
		return
	} else if resp.StatusCode >= 300 {
		err = errors.New(string(respBody))
	}

	return
}

// (PUT /user/{username})
func (a *UserAPI) UpdateUser(ctx context.Context, req *UpdateUserRequest) (resp *http.Response, err error) {
	var (
		contentType string
		body        any
	)
	path := "/user/{username}"
	path = path[:6] + req.PathParams.Username + path[6+10:]
	contentType = selectHeaderContentType([]string{"application/json"})
	body = req.Body

	request, err := a.client.prepareRequest("PUT", a.client.cfg.BasePath+path, contentType, body)
	if err != nil {
		return
	}
	resp, err = a.client.Do(request)
	if err != nil {
		return
	}
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	if resp.StatusCode == http.StatusOK {
		return
	} else if resp.StatusCode >= 300 {
		err = errors.New(string(respBody))
	}

	return
}
