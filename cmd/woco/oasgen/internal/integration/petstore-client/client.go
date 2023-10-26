// Code generated by woco, DO NOT EDIT.

package client

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	BasePath   string            `json:"basePath,omitempty" yaml:"basePath,omitempty"`
	Host       string            `json:"host,omitempty" yaml:"host,omitempty"`
	Headers    map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
	UserAgent  string            `json:"userAgent,omitempty" yaml:"userAgent,omitempty"`
	HTTPClient *http.Client
}

func NewConfig() *Config {
	return &Config{
		BasePath:  "http://petstore.swagger.io/v2",
		UserAgent: "oasgen/1.0.0/go",
	}
}

type APIClient struct {
	cfg      *Config
	common   api // Reuse a single struct instead of allocating one for each service on the heap.
	PetAPI   *PetAPI
	StoreAPI *StoreAPI
	UserAPI  *UserAPI
}

type api struct {
	client *APIClient
}

func NewAPIClient(cfg *Config) *APIClient {
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = http.DefaultClient
	}
	c := &APIClient{
		cfg: cfg,
	}
	c.common.client = c

	c.PetAPI = (*PetAPI)(&c.common)
	c.StoreAPI = (*StoreAPI)(&c.common)
	c.UserAPI = (*UserAPI)(&c.common)

	return c
}

// parameterToString convert any parameters to string, using a delimiter if format is provided.
func parameterToString(obj any, collectionFormat string, isSlice bool) string {
	delimiter := ","

	switch collectionFormat {
	case "pipeDelimited":
		delimiter = "|"
	case "spaceDelimited":
		delimiter = " "
	}

	if isSlice {
		return strings.Trim(strings.Replace(fmt.Sprint(obj), " ", delimiter, -1), "[]")
	}

	return fmt.Sprintf("%v", obj)
}

// selectHeaderContentType select a content type from the available list.
func selectHeaderContentType(contentTypes []string) string {
	if len(contentTypes) == 0 {
		return ""
	}
	if contains(contentTypes, "application/json") {
		return "application/json"
	}
	return contentTypes[0] // use the first content type specified in 'consumes'
}

// selectHeaderAccept join all accept types and return
func selectHeaderAccept(accepts []string) string {
	if len(accepts) == 0 {
		return ""
	}
	if contains(accepts, "application/json") {
		return "application/json"
	}
	return strings.Join(accepts, ",")
}

// contains is a case insenstive match, finding needle in a haystack
func contains(haystack []string, needle string) bool {
	for _, a := range haystack {
		if strings.ToLower(a) == strings.ToLower(needle) {
			return true
		}
	}
	return false
}

func (c *APIClient) prepareRequest(
	method string, path string,
	contentType string,
	body any,
) (req *http.Request, err error) {
	var (
		payload io.Reader
	)
	if body != nil {
		payload, err = parseRequestBody(body, contentType)
		if err != nil {
			return
		}
	}
	if payload != nil {
		req, err = http.NewRequest(method, path, payload)
	} else {
		req, err = http.NewRequest(method, path, nil)
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	req.Header.Set("Content-Type", contentType)
	for k, v := range c.cfg.Headers {
		req.Header.Set(k, v)
	}
	if c.cfg.Host != "" {
		req.Header.Set("Host", c.cfg.Host)
	}
	return
}

// Do sends an HTTP request and returns an HTTP response.
func (c *APIClient) Do(req *http.Request) (res *http.Response, err error) {
	return c.cfg.HTTPClient.Do(req)
}

func (c *APIClient) decode(b []byte, v interface{}, contentType string) error {
	if strings.Contains(contentType, "application/json") {
		return json.Unmarshal(b, v)
	}
	if strings.Contains(contentType, "application/xml") {
		return xml.Unmarshal(b, v)
	}

	return errors.New("undefined response type")
}

// Set request body from an interface{}
func parseRequestBody(body interface{}, contentType string) (io.Reader, error) {
	switch data := body.(type) {
	case io.Reader:
		return data, nil
	case []byte:
		return bytes.NewBuffer(data), nil
	case string:
		return strings.NewReader(data), nil
	case *string:
		return strings.NewReader(*data), nil
	}

	var (
		bodyBuf = bytes.Buffer{}
		err     error
	)
	switch contentType {
	case "application/json":
		err = json.NewEncoder(&bodyBuf).Encode(body)
	case "application/xml":
		err = xml.NewEncoder(&bodyBuf).Encode(body)
	case "multipart/form-data":
		w := multipart.NewWriter(&bodyBuf)
		formParams, ok := body.(url.Values)
		if !ok {
			return nil, fmt.Errorf("Invalid body type %s\n", contentType)
		}
		for k, v := range formParams {
			for _, iv := range v {
				if strings.HasPrefix(k, "@") { // file
					err = addFile(w, k[1:], iv)
					if err != nil {
						return nil, err
					}
				} else { // form value
					w.WriteField(k, iv)
				}
			}
		}
		w.Close()
	default:
		err = fmt.Errorf("Invalid body type %s\n", contentType)
	}
	if err != nil {
		return nil, err
	}

	return &bodyBuf, nil
}

// Add a file to the multipart request
func addFile(w *multipart.Writer, fieldName, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	part, err := w.CreateFormFile(fieldName, filepath.Base(path))
	if err != nil {
		return err
	}
	_, err = io.Copy(part, file)

	return err
}

func isCertificateError(err error) bool {
	if err != nil && strings.Contains(err.Error(), "x509: certificate signed by unknown authority") {
		return true
	}
	return false
}
