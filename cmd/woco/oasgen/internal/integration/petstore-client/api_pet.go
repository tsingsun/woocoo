// Code generated by woco, DO NOT EDIT.

package client

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/tsingsun/woocoo/cmd/woco/oasgen/internal/integration/extra"
)

type PetAPI api

// (POST /pet)
func (a *PetAPI) AddPet(ctx context.Context, req *AddPetRequest) (ret *Pet, resp *http.Response, err error) {
	var (
		contentType string
		body        any
	)
	path := "/pet"
	contentType = selectHeaderContentType([]string{"application/json", "application/xml"})
	body = req

	request, err := a.client.prepareRequest("POST", a.client.cfg.BasePath+path, contentType, body)
	if err != nil {
		return
	}
	accept := selectHeaderAccept([]string{"application/json", "application/xml"})
	request.Header.Set("Accept", accept)
	resp, err = a.client.Do(ctx, request)
	if err != nil {
		return
	}
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	if resp.StatusCode == http.StatusOK {
		ret = new(Pet)
		err = a.client.decode(respBody, ret, resp.Header.Get("Content-Type"))
		if err == nil {
			return
		}
	} else if resp.StatusCode >= 300 {
		err = errors.New(string(respBody))
	}

	return
}

// (DELETE /pet/{petId})
func (a *PetAPI) DeletePet(ctx context.Context, req *DeletePetRequest) (resp *http.Response, err error) {
	var (
		contentType string
		body        any
	)
	path := "/pet/{petId}"
	path = path[:5] + strconv.FormatInt(int64(req.PathParams.PetId), 10) + path[5+7:]

	request, err := a.client.prepareRequest("DELETE", a.client.cfg.BasePath+path, contentType, body)
	if err != nil {
		return
	}
	if req.HeaderParams.APIKey != nil {
		request.Header.Set("api_key", *req.HeaderParams.APIKey)
	}
	resp, err = a.client.Do(ctx, request)
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

// (GET /pet/findByStatus)
func (a *PetAPI) FindPetsByStatus(ctx context.Context, req *FindPetsByStatusRequest) (ret []*Pet, resp *http.Response, err error) {
	var (
		contentType string
		body        any
	)
	path := "/pet/findByStatus"
	queryParams := url.Values{}
	for _, v := range req.Status {
		queryParams.Add("status", fmt.Sprintf("%v", v))
	}

	request, err := a.client.prepareRequest("GET", a.client.cfg.BasePath+path, contentType, body)
	if err != nil {
		return
	}
	request.URL.RawQuery = queryParams.Encode()
	accept := selectHeaderAccept([]string{"application/json", "application/xml"})
	request.Header.Set("Accept", accept)
	resp, err = a.client.Do(ctx, request)
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

// (GET /pet/findByTags)
func (a *PetAPI) FindPetsByTags(ctx context.Context, req *FindPetsByTagsRequest) (ret Pets, resp *http.Response, err error) {
	var (
		contentType string
		body        any
	)
	path := "/pet/findByTags"
	queryParams := url.Values{}
	for _, v := range req.Tags {
		queryParams.Add("tags", v)
	}

	request, err := a.client.prepareRequest("GET", a.client.cfg.BasePath+path, contentType, body)
	if err != nil {
		return
	}
	request.URL.RawQuery = queryParams.Encode()
	accept := selectHeaderAccept([]string{"application/json", "application/xml"})
	request.Header.Set("Accept", accept)
	resp, err = a.client.Do(ctx, request)
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

// (GET /pet/{petId})
func (a *PetAPI) GetPetById(ctx context.Context, req *GetPetByIdRequest) (ret *Pet, resp *http.Response, err error) {
	var (
		contentType string
		body        any
	)
	path := "/pet/{petId}"
	path = path[:5] + strconv.FormatInt(int64(req.PetId), 10) + path[5+7:]

	request, err := a.client.prepareRequest("GET", a.client.cfg.BasePath+path, contentType, body)
	if err != nil {
		return
	}
	accept := selectHeaderAccept([]string{"application/json", "application/xml"})
	request.Header.Set("Accept", accept)
	resp, err = a.client.Do(ctx, request)
	if err != nil {
		return
	}
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	if resp.StatusCode == http.StatusOK {
		ret = new(Pet)
		err = a.client.decode(respBody, ret, resp.Header.Get("Content-Type"))
		if err == nil {
			return
		}
	} else if resp.StatusCode >= 300 {
		err = errors.New(string(respBody))
	}

	return
}

// (PUT /pet)
func (a *PetAPI) UpdatePet(ctx context.Context, req *UpdatePetRequest) (ret *Pet, resp *http.Response, err error) {
	var (
		contentType string
		body        any
	)
	path := "/pet"
	contentType = selectHeaderContentType([]string{"application/json", "application/xml"})
	body = req

	request, err := a.client.prepareRequest("PUT", a.client.cfg.BasePath+path, contentType, body)
	if err != nil {
		return
	}
	accept := selectHeaderAccept([]string{"application/json", "application/xml"})
	request.Header.Set("Accept", accept)
	resp, err = a.client.Do(ctx, request)
	if err != nil {
		return
	}
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	if resp.StatusCode == http.StatusOK {
		ret = new(Pet)
		err = a.client.decode(respBody, ret, resp.Header.Get("Content-Type"))
		if err == nil {
			return
		}
	} else if resp.StatusCode >= 300 {
		err = errors.New(string(respBody))
	}

	return
}

// (POST /pet/{petId})
func (a *PetAPI) UpdatePetWithForm(ctx context.Context, req *UpdatePetWithFormRequest) (resp *http.Response, err error) {
	var (
		contentType string
		body        any
	)
	path := "/pet/{petId}"
	path = path[:5] + strconv.FormatInt(int64(req.PathParams.PetId), 10) + path[5+7:]
	queryParams := url.Values{}
	if req.QueryParams.Timestamp != nil {
		queryParams.Add("timestamp", strconv.FormatInt(int64(*req.QueryParams.Timestamp), 10))
	}
	contentType = selectHeaderContentType([]string{"application/x-www-form-urlencoded"})
	forms := url.Values{}
	forms.Add("name", req.Body.Name)
	forms.Add("status", req.Body.Status)
	body = strings.NewReader(forms.Encode())

	request, err := a.client.prepareRequest("POST", a.client.cfg.BasePath+path, contentType, body)
	if err != nil {
		return
	}
	request.URL.RawQuery = queryParams.Encode()
	resp, err = a.client.Do(ctx, request)
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

// (POST /pet/{petId}/uploadImage)
func (a *PetAPI) UploadFile(ctx context.Context, req *UploadFileRequest) (ret *extra.ApiResponse, resp *http.Response, err error) {
	var (
		contentType string
		body        any
	)
	path := "/pet/{petId}/uploadImage"
	path = path[:5] + strconv.FormatInt(int64(req.PathParams.PetId), 10) + path[5+7:]
	contentType = selectHeaderContentType([]string{"multipart/form-data"})
	forms := url.Values{}
	forms.Add("additionalMetadata", req.Body.AdditionalMetadata)
	forms.Add("file", string(req.Body.File))
	forms.Add("md5", req.Body.Md5)
	body = forms

	request, err := a.client.prepareRequest("POST", a.client.cfg.BasePath+path, contentType, body)
	if err != nil {
		return
	}
	accept := selectHeaderAccept([]string{"application/json"})
	request.Header.Set("Accept", accept)
	resp, err = a.client.Do(ctx, request)
	if err != nil {
		return
	}
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}
	if resp.StatusCode == http.StatusOK {
		ret = new(extra.ApiResponse)
		err = a.client.decode(respBody, ret, resp.Header.Get("Content-Type"))
		if err == nil {
			return
		}
	} else if resp.StatusCode >= 300 {
		err = errors.New(string(respBody))
	}

	return
}
