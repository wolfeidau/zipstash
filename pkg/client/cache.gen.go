// Package client provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/oapi-codegen/oapi-codegen/v2 version v2.4.1 DO NOT EDIT.
package client

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/oapi-codegen/runtime"
)

// CacheDownloadInstruction defines model for CacheDownloadInstruction.
type CacheDownloadInstruction struct {
	// Method HTTP method
	Method string  `json:"method"`
	Offset *Offset `json:"offset,omitempty"`

	// Url URL
	Url string `json:"url"`
}

// CacheEntry defines model for CacheEntry.
type CacheEntry struct {
	// Compression Compression algorithm
	Compression string `json:"compression"`

	// FileSize Size of the cache entry in bytes
	FileSize int64 `json:"file_size"`

	// Key Key of the cache entry
	Key string `json:"key"`

	// Paths Paths to upload the cache entry
	Paths []string `json:"paths"`

	// Sha256sum SHA256 checksum of the cache entry
	Sha256sum string `json:"sha256sum"`

	// Url URL to upload the cache entry
	Url *string `json:"url,omitempty"`
}

// CacheEntryCreateRequest defines model for CacheEntryCreateRequest.
type CacheEntryCreateRequest struct {
	CacheEntry CacheEntry `json:"cache_entry"`

	// MultipartSupported multipart supported
	MultipartSupported bool `json:"multipart_supported"`
}

// CacheEntryCreateResponse defines model for CacheEntryCreateResponse.
type CacheEntryCreateResponse struct {
	// Id Upload ID
	Id                 string                   `json:"id"`
	Multipart          bool                     `json:"multipart"`
	UploadInstructions []CacheUploadInstruction `json:"upload_instructions"`
}

// CacheEntryGetResponse defines model for CacheEntryGetResponse.
type CacheEntryGetResponse struct {
	CacheEntry           CacheEntry                 `json:"cache_entry"`
	DownloadInstructions []CacheDownloadInstruction `json:"download_instructions"`
	Multipart            bool                       `json:"multipart"`
}

// CacheEntryUpdateRequest defines model for CacheEntryUpdateRequest.
type CacheEntryUpdateRequest struct {
	// Id Upload ID
	Id string `json:"id"`

	// Key Key of the cache entry
	Key string `json:"key"`

	// MultipartEtags ETags
	MultipartEtags []CachePartETag `json:"multipart_etags"`
}

// CacheEntryUpdateResponse defines model for CacheEntryUpdateResponse.
type CacheEntryUpdateResponse struct {
	// Id Response ID
	Id string `json:"id"`
}

// CachePartETag Part index and ETag
type CachePartETag struct {
	// Etag ETag
	Etag string `json:"etag"`

	// Part Part index
	Part     int32 `json:"part"`
	PartSize int64 `json:"part_size"`
}

// CacheUploadInstruction defines model for CacheUploadInstruction.
type CacheUploadInstruction struct {
	// Method HTTP method
	Method string  `json:"method"`
	Offset *Offset `json:"offset,omitempty"`

	// Url URL
	Url string `json:"url"`
}

// Error defines model for Error.
type Error struct {
	// Code Error code
	Code int32 `json:"code"`

	// Message Error message
	Message string `json:"message"`
}

// Offset defines model for Offset.
type Offset struct {
	// End End position of the part
	End int64 `json:"end"`

	// Part Part number
	Part int32 `json:"part"`

	// Start Start position of the part
	Start int64 `json:"start"`
}

// CreateCacheEntryJSONRequestBody defines body for CreateCacheEntry for application/json ContentType.
type CreateCacheEntryJSONRequestBody = CacheEntryCreateRequest

// UpdateCacheEntryJSONRequestBody defines body for UpdateCacheEntry for application/json ContentType.
type UpdateCacheEntryJSONRequestBody = CacheEntryUpdateRequest

// RequestEditorFn  is the function signature for the RequestEditor callback function
type RequestEditorFn func(ctx context.Context, req *http.Request) error

// Doer performs HTTP requests.
//
// The standard http.Client implements this interface.
type HttpRequestDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Client which conforms to the OpenAPI3 specification for this service.
type Client struct {
	// The endpoint of the server conforming to this interface, with scheme,
	// https://api.deepmap.com for example. This can contain a path relative
	// to the server, such as https://api.deepmap.com/dev-test, and all the
	// paths in the swagger spec will be appended to the server.
	Server string

	// Doer for performing requests, typically a *http.Client with any
	// customized settings, such as certificate chains.
	Client HttpRequestDoer

	// A list of callbacks for modifying requests which are generated before sending over
	// the network.
	RequestEditors []RequestEditorFn
}

// ClientOption allows setting custom parameters during construction
type ClientOption func(*Client) error

// Creates a new Client, with reasonable defaults
func NewClient(server string, opts ...ClientOption) (*Client, error) {
	// create a client with sane default values
	client := Client{
		Server: server,
	}
	// mutate client and add all optional params
	for _, o := range opts {
		if err := o(&client); err != nil {
			return nil, err
		}
	}
	// ensure the server URL always has a trailing slash
	if !strings.HasSuffix(client.Server, "/") {
		client.Server += "/"
	}
	// create httpClient, if not already present
	if client.Client == nil {
		client.Client = &http.Client{}
	}
	return &client, nil
}

// WithHTTPClient allows overriding the default Doer, which is
// automatically created using http.Client. This is useful for tests.
func WithHTTPClient(doer HttpRequestDoer) ClientOption {
	return func(c *Client) error {
		c.Client = doer
		return nil
	}
}

// WithRequestEditorFn allows setting up a callback function, which will be
// called right before sending the request. This can be used to mutate the request.
func WithRequestEditorFn(fn RequestEditorFn) ClientOption {
	return func(c *Client) error {
		c.RequestEditors = append(c.RequestEditors, fn)
		return nil
	}
}

// The interface specification for the client above.
type ClientInterface interface {
	// CreateCacheEntryWithBody request with any body
	CreateCacheEntryWithBody(ctx context.Context, provider string, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*http.Response, error)

	CreateCacheEntry(ctx context.Context, provider string, body CreateCacheEntryJSONRequestBody, reqEditors ...RequestEditorFn) (*http.Response, error)

	// UpdateCacheEntryWithBody request with any body
	UpdateCacheEntryWithBody(ctx context.Context, provider string, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*http.Response, error)

	UpdateCacheEntry(ctx context.Context, provider string, body UpdateCacheEntryJSONRequestBody, reqEditors ...RequestEditorFn) (*http.Response, error)

	// GetCacheEntryByKey request
	GetCacheEntryByKey(ctx context.Context, provider string, key string, reqEditors ...RequestEditorFn) (*http.Response, error)
}

func (c *Client) CreateCacheEntryWithBody(ctx context.Context, provider string, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewCreateCacheEntryRequestWithBody(c.Server, provider, contentType, body)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) CreateCacheEntry(ctx context.Context, provider string, body CreateCacheEntryJSONRequestBody, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewCreateCacheEntryRequest(c.Server, provider, body)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) UpdateCacheEntryWithBody(ctx context.Context, provider string, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewUpdateCacheEntryRequestWithBody(c.Server, provider, contentType, body)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) UpdateCacheEntry(ctx context.Context, provider string, body UpdateCacheEntryJSONRequestBody, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewUpdateCacheEntryRequest(c.Server, provider, body)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

func (c *Client) GetCacheEntryByKey(ctx context.Context, provider string, key string, reqEditors ...RequestEditorFn) (*http.Response, error) {
	req, err := NewGetCacheEntryByKeyRequest(c.Server, provider, key)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	if err := c.applyEditors(ctx, req, reqEditors); err != nil {
		return nil, err
	}
	return c.Client.Do(req)
}

// NewCreateCacheEntryRequest calls the generic CreateCacheEntry builder with application/json body
func NewCreateCacheEntryRequest(server string, provider string, body CreateCacheEntryJSONRequestBody) (*http.Request, error) {
	var bodyReader io.Reader
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	bodyReader = bytes.NewReader(buf)
	return NewCreateCacheEntryRequestWithBody(server, provider, "application/json", bodyReader)
}

// NewCreateCacheEntryRequestWithBody generates requests for CreateCacheEntry with any type of body
func NewCreateCacheEntryRequestWithBody(server string, provider string, contentType string, body io.Reader) (*http.Request, error) {
	var err error

	var pathParam0 string

	pathParam0, err = runtime.StyleParamWithLocation("simple", false, "provider", runtime.ParamLocationPath, provider)
	if err != nil {
		return nil, err
	}

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/v1/cache/%s", pathParam0)
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", queryURL.String(), body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", contentType)

	return req, nil
}

// NewUpdateCacheEntryRequest calls the generic UpdateCacheEntry builder with application/json body
func NewUpdateCacheEntryRequest(server string, provider string, body UpdateCacheEntryJSONRequestBody) (*http.Request, error) {
	var bodyReader io.Reader
	buf, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	bodyReader = bytes.NewReader(buf)
	return NewUpdateCacheEntryRequestWithBody(server, provider, "application/json", bodyReader)
}

// NewUpdateCacheEntryRequestWithBody generates requests for UpdateCacheEntry with any type of body
func NewUpdateCacheEntryRequestWithBody(server string, provider string, contentType string, body io.Reader) (*http.Request, error) {
	var err error

	var pathParam0 string

	pathParam0, err = runtime.StyleParamWithLocation("simple", false, "provider", runtime.ParamLocationPath, provider)
	if err != nil {
		return nil, err
	}

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/v1/cache/%s", pathParam0)
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("PUT", queryURL.String(), body)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", contentType)

	return req, nil
}

// NewGetCacheEntryByKeyRequest generates requests for GetCacheEntryByKey
func NewGetCacheEntryByKeyRequest(server string, provider string, key string) (*http.Request, error) {
	var err error

	var pathParam0 string

	pathParam0, err = runtime.StyleParamWithLocation("simple", false, "provider", runtime.ParamLocationPath, provider)
	if err != nil {
		return nil, err
	}

	var pathParam1 string

	pathParam1, err = runtime.StyleParamWithLocation("simple", false, "key", runtime.ParamLocationPath, key)
	if err != nil {
		return nil, err
	}

	serverURL, err := url.Parse(server)
	if err != nil {
		return nil, err
	}

	operationPath := fmt.Sprintf("/v1/cache/%s/%s", pathParam0, pathParam1)
	if operationPath[0] == '/' {
		operationPath = "." + operationPath
	}

	queryURL, err := serverURL.Parse(operationPath)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", queryURL.String(), nil)
	if err != nil {
		return nil, err
	}

	return req, nil
}

func (c *Client) applyEditors(ctx context.Context, req *http.Request, additionalEditors []RequestEditorFn) error {
	for _, r := range c.RequestEditors {
		if err := r(ctx, req); err != nil {
			return err
		}
	}
	for _, r := range additionalEditors {
		if err := r(ctx, req); err != nil {
			return err
		}
	}
	return nil
}

// ClientWithResponses builds on ClientInterface to offer response payloads
type ClientWithResponses struct {
	ClientInterface
}

// NewClientWithResponses creates a new ClientWithResponses, which wraps
// Client with return type handling
func NewClientWithResponses(server string, opts ...ClientOption) (*ClientWithResponses, error) {
	client, err := NewClient(server, opts...)
	if err != nil {
		return nil, err
	}
	return &ClientWithResponses{client}, nil
}

// WithBaseURL overrides the baseURL.
func WithBaseURL(baseURL string) ClientOption {
	return func(c *Client) error {
		newBaseURL, err := url.Parse(baseURL)
		if err != nil {
			return err
		}
		c.Server = newBaseURL.String()
		return nil
	}
}

// ClientWithResponsesInterface is the interface specification for the client with responses above.
type ClientWithResponsesInterface interface {
	// CreateCacheEntryWithBodyWithResponse request with any body
	CreateCacheEntryWithBodyWithResponse(ctx context.Context, provider string, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*CreateCacheEntryResponse, error)

	CreateCacheEntryWithResponse(ctx context.Context, provider string, body CreateCacheEntryJSONRequestBody, reqEditors ...RequestEditorFn) (*CreateCacheEntryResponse, error)

	// UpdateCacheEntryWithBodyWithResponse request with any body
	UpdateCacheEntryWithBodyWithResponse(ctx context.Context, provider string, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*UpdateCacheEntryResponse, error)

	UpdateCacheEntryWithResponse(ctx context.Context, provider string, body UpdateCacheEntryJSONRequestBody, reqEditors ...RequestEditorFn) (*UpdateCacheEntryResponse, error)

	// GetCacheEntryByKeyWithResponse request
	GetCacheEntryByKeyWithResponse(ctx context.Context, provider string, key string, reqEditors ...RequestEditorFn) (*GetCacheEntryByKeyResponse, error)
}

type CreateCacheEntryResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON201      *CacheEntryCreateResponse
	JSONDefault  *Error
}

// Status returns HTTPResponse.Status
func (r CreateCacheEntryResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r CreateCacheEntryResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

type UpdateCacheEntryResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *CacheEntryUpdateResponse
	JSONDefault  *Error
}

// Status returns HTTPResponse.Status
func (r UpdateCacheEntryResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r UpdateCacheEntryResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

type GetCacheEntryByKeyResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *CacheEntryGetResponse
	JSONDefault  *Error
}

// Status returns HTTPResponse.Status
func (r GetCacheEntryByKeyResponse) Status() string {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.Status
	}
	return http.StatusText(0)
}

// StatusCode returns HTTPResponse.StatusCode
func (r GetCacheEntryByKeyResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

// CreateCacheEntryWithBodyWithResponse request with arbitrary body returning *CreateCacheEntryResponse
func (c *ClientWithResponses) CreateCacheEntryWithBodyWithResponse(ctx context.Context, provider string, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*CreateCacheEntryResponse, error) {
	rsp, err := c.CreateCacheEntryWithBody(ctx, provider, contentType, body, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseCreateCacheEntryResponse(rsp)
}

func (c *ClientWithResponses) CreateCacheEntryWithResponse(ctx context.Context, provider string, body CreateCacheEntryJSONRequestBody, reqEditors ...RequestEditorFn) (*CreateCacheEntryResponse, error) {
	rsp, err := c.CreateCacheEntry(ctx, provider, body, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseCreateCacheEntryResponse(rsp)
}

// UpdateCacheEntryWithBodyWithResponse request with arbitrary body returning *UpdateCacheEntryResponse
func (c *ClientWithResponses) UpdateCacheEntryWithBodyWithResponse(ctx context.Context, provider string, contentType string, body io.Reader, reqEditors ...RequestEditorFn) (*UpdateCacheEntryResponse, error) {
	rsp, err := c.UpdateCacheEntryWithBody(ctx, provider, contentType, body, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseUpdateCacheEntryResponse(rsp)
}

func (c *ClientWithResponses) UpdateCacheEntryWithResponse(ctx context.Context, provider string, body UpdateCacheEntryJSONRequestBody, reqEditors ...RequestEditorFn) (*UpdateCacheEntryResponse, error) {
	rsp, err := c.UpdateCacheEntry(ctx, provider, body, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseUpdateCacheEntryResponse(rsp)
}

// GetCacheEntryByKeyWithResponse request returning *GetCacheEntryByKeyResponse
func (c *ClientWithResponses) GetCacheEntryByKeyWithResponse(ctx context.Context, provider string, key string, reqEditors ...RequestEditorFn) (*GetCacheEntryByKeyResponse, error) {
	rsp, err := c.GetCacheEntryByKey(ctx, provider, key, reqEditors...)
	if err != nil {
		return nil, err
	}
	return ParseGetCacheEntryByKeyResponse(rsp)
}

// ParseCreateCacheEntryResponse parses an HTTP response from a CreateCacheEntryWithResponse call
func ParseCreateCacheEntryResponse(rsp *http.Response) (*CreateCacheEntryResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &CreateCacheEntryResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 201:
		var dest CacheEntryCreateResponse
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON201 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && true:
		var dest Error
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSONDefault = &dest

	}

	return response, nil
}

// ParseUpdateCacheEntryResponse parses an HTTP response from a UpdateCacheEntryWithResponse call
func ParseUpdateCacheEntryResponse(rsp *http.Response) (*UpdateCacheEntryResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &UpdateCacheEntryResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 200:
		var dest CacheEntryUpdateResponse
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON200 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && true:
		var dest Error
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSONDefault = &dest

	}

	return response, nil
}

// ParseGetCacheEntryByKeyResponse parses an HTTP response from a GetCacheEntryByKeyWithResponse call
func ParseGetCacheEntryByKeyResponse(rsp *http.Response) (*GetCacheEntryByKeyResponse, error) {
	bodyBytes, err := io.ReadAll(rsp.Body)
	defer func() { _ = rsp.Body.Close() }()
	if err != nil {
		return nil, err
	}

	response := &GetCacheEntryByKeyResponse{
		Body:         bodyBytes,
		HTTPResponse: rsp,
	}

	switch {
	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && rsp.StatusCode == 200:
		var dest CacheEntryGetResponse
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSON200 = &dest

	case strings.Contains(rsp.Header.Get("Content-Type"), "json") && true:
		var dest Error
		if err := json.Unmarshal(bodyBytes, &dest); err != nil {
			return nil, err
		}
		response.JSONDefault = &dest

	}

	return response, nil
}

// Base64 encoded, gzipped, json marshaled Swagger object
var swaggerSpec = []string{

	"H4sIAAAAAAAC/+RX23LbNhD9FQzaRyai5EtbvTWOx/Ek03gS+ynj8cDkSkQtAgiwdEx7+O8dACTFCyRZ",
	"bd3rm0Qscc6ePVgsn2gicyUFCDR0/kRNkkHO3M8TlmTwVn4TK8nSc2FQFwlyKeya0lKBRg4uMgfMZGp/",
	"pWASzZUPo+8uLy9IvRhRLBXQOTWouVjSKqJysTCA9rXvNSzonH43WZOZ1EwmH31UFdFCr8YYV58+jPeu",
	"Iqrha8E1pHT+xb0YNSyvq8indipQl+NkLAUNxtSZ9tFO1ouErZZSc8xyGlF4YLlaWQrLR64sGBcfQCwx",
	"o/ODQOoLvoIbwx9hDPGZPwKRC4IZkMTyJGCJEi7IbYlgumjTeBbRhdQ5QzqnXODxocfmeZHT+bRF5gJh",
	"CdpC30E5Bn0PZQCzl1hevnJLr5qlTorTQIqKYWbGSBf2MUFJCmV9FYDkCLl7cbRlzsW5X1wDMq1ZaRdN",
	"xmZHx8bmPZL03c+zo2OSZJDcmSLflel0dnB4dPzDjz/F7DZJYbHv/742x4cBcTZZeaswWwUfWN5WuWuz",
	"qOfrrlpNpfrn4kQDQ/gEXwswGDgkNvAGmhO07fx2zpotYbFCrpjGG1MoJTVCoG+0QWQd1CZ8K+UKmBhl",
	"HNo56hENZ2iUFAbGKfIAsStfmvO3oXbWEuh4tyUbUV/WG75upB6mcftODT14txFXw1MwkIRbBUK4XbJ9",
	"Vc4AN0vyu6ue1rfIH8g+dBFV4y6wtQgDebrpbOK4WakrlW47IXvb56/qy+tzAsiWgQ59emkfR3vU5oJp",
	"tG/t6NEhd/o+NeQUVnqfk9pEB8UeE2kB21QCF5dGwkUKD4SJlLigaMDEsg8L+pz70pt2Eyrt3/QHs503",
	"vW+G9ZAxmhKG8QNRHJ3IZ9TdqlVq3I7+/XPhqdZSh0bCNDCouWDi1gKlGZcjB2PYcuNGzfIu0jVgE25p",
	"f2wFG9hRBMQ/FSlR0nD7t+ktdbH3nCS3OFYU+S3ooGXZQ71lHMfxLgiDQYzP9vH+WcQ7Xe/xIqdcneB1",
	"ZYO4WEh3pXB0XfeRK2KQmYwY0Pcu1XvQ/qOBTl/Hr2PnZAWCKU7n9MA96kzFk/vpxPXtyZPS8p6noCtX",
	"QelvE1tHZrM7T+13hxtWOpeqI8dyQNCGzr8MBfqF5e1HRLP9a9vT7aLlQCMqWG4zaZZpVwnUBUT1x2Bg",
	"FK+ufTAYfCPT0p8RgSAcdabUiieO/ORX4xvDeqvnTQ39+bOqfKF8T3f6zeLpC8LWV42D7Svrr+GGCvnG",
	"MWsm9t7g4F5csGKFfxpN350CnAoBDwoShJRAHRNRU+Q5s4NabR5DWG+GsAe4CFjN37b/I6v1B7mg1eIX",
	"hP1PWc0nNbJaFQX73eTpDkrX9ZYQcOIZ4FquN+V7+Nu9GD2FtvIz7J6OfnF/db/knmmuQXv4p5jqDLBv",
	"KHJbEiu6Pau/BQAA///PFBDzwRQAAA==",
}

// GetSwagger returns the content of the embedded swagger specification file
// or error if failed to decode
func decodeSpec() ([]byte, error) {
	zipped, err := base64.StdEncoding.DecodeString(strings.Join(swaggerSpec, ""))
	if err != nil {
		return nil, fmt.Errorf("error base64 decoding spec: %w", err)
	}
	zr, err := gzip.NewReader(bytes.NewReader(zipped))
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %w", err)
	}
	var buf bytes.Buffer
	_, err = buf.ReadFrom(zr)
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %w", err)
	}

	return buf.Bytes(), nil
}

var rawSpec = decodeSpecCached()

// a naive cached of a decoded swagger spec
func decodeSpecCached() func() ([]byte, error) {
	data, err := decodeSpec()
	return func() ([]byte, error) {
		return data, err
	}
}

// Constructs a synthetic filesystem for resolving external references when loading openapi specifications.
func PathToRawSpec(pathToFile string) map[string]func() ([]byte, error) {
	res := make(map[string]func() ([]byte, error))
	if len(pathToFile) > 0 {
		res[pathToFile] = rawSpec
	}

	return res
}

// GetSwagger returns the Swagger specification corresponding to the generated code
// in this file. The external references of Swagger specification are resolved.
// The logic of resolving external references is tightly connected to "import-mapping" feature.
// Externally referenced files must be embedded in the corresponding golang packages.
// Urls can be supported but this task was out of the scope.
func GetSwagger() (swagger *openapi3.T, err error) {
	resolvePath := PathToRawSpec("")

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	loader.ReadFromURIFunc = func(loader *openapi3.Loader, url *url.URL) ([]byte, error) {
		pathToFile := url.String()
		pathToFile = path.Clean(pathToFile)
		getSpec, ok := resolvePath[pathToFile]
		if !ok {
			err1 := fmt.Errorf("path not found: %s", pathToFile)
			return nil, err1
		}
		return getSpec()
	}
	var specData []byte
	specData, err = rawSpec()
	if err != nil {
		return
	}
	swagger, err = loader.LoadFromData(specData)
	if err != nil {
		return
	}
	return
}
