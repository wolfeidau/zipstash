// Package api provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/oapi-codegen/oapi-codegen/v2 version v2.4.1 DO NOT EDIT.
package api

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/labstack/echo/v4"
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
	Part int32 `json:"part"`
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

// ServerInterface represents all server handlers.
type ServerInterface interface {
	// Creates a cache entry
	// (POST /v1/cache/{provider})
	CreateCacheEntry(ctx echo.Context, provider string) error
	// Updates a cache entry
	// (PUT /v1/cache/{provider})
	UpdateCacheEntry(ctx echo.Context, provider string) error
	// Get a cache entry by key
	// (GET /v1/cache/{provider}/{key})
	GetCacheEntryByKey(ctx echo.Context, provider string, key string) error
}

// ServerInterfaceWrapper converts echo contexts to parameters.
type ServerInterfaceWrapper struct {
	Handler ServerInterface
}

// CreateCacheEntry converts echo context to params.
func (w *ServerInterfaceWrapper) CreateCacheEntry(ctx echo.Context) error {
	var err error
	// ------------- Path parameter "provider" -------------
	var provider string

	err = runtime.BindStyledParameterWithOptions("simple", "provider", ctx.Param("provider"), &provider, runtime.BindStyledParameterOptions{ParamLocation: runtime.ParamLocationPath, Explode: false, Required: true})
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter provider: %s", err))
	}

	// Invoke the callback with all the unmarshaled arguments
	err = w.Handler.CreateCacheEntry(ctx, provider)
	return err
}

// UpdateCacheEntry converts echo context to params.
func (w *ServerInterfaceWrapper) UpdateCacheEntry(ctx echo.Context) error {
	var err error
	// ------------- Path parameter "provider" -------------
	var provider string

	err = runtime.BindStyledParameterWithOptions("simple", "provider", ctx.Param("provider"), &provider, runtime.BindStyledParameterOptions{ParamLocation: runtime.ParamLocationPath, Explode: false, Required: true})
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter provider: %s", err))
	}

	// Invoke the callback with all the unmarshaled arguments
	err = w.Handler.UpdateCacheEntry(ctx, provider)
	return err
}

// GetCacheEntryByKey converts echo context to params.
func (w *ServerInterfaceWrapper) GetCacheEntryByKey(ctx echo.Context) error {
	var err error
	// ------------- Path parameter "provider" -------------
	var provider string

	err = runtime.BindStyledParameterWithOptions("simple", "provider", ctx.Param("provider"), &provider, runtime.BindStyledParameterOptions{ParamLocation: runtime.ParamLocationPath, Explode: false, Required: true})
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter provider: %s", err))
	}

	// ------------- Path parameter "key" -------------
	var key string

	err = runtime.BindStyledParameterWithOptions("simple", "key", ctx.Param("key"), &key, runtime.BindStyledParameterOptions{ParamLocation: runtime.ParamLocationPath, Explode: false, Required: true})
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, fmt.Sprintf("Invalid format for parameter key: %s", err))
	}

	// Invoke the callback with all the unmarshaled arguments
	err = w.Handler.GetCacheEntryByKey(ctx, provider, key)
	return err
}

// This is a simple interface which specifies echo.Route addition functions which
// are present on both echo.Echo and echo.Group, since we want to allow using
// either of them for path registration
type EchoRouter interface {
	CONNECT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	DELETE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	GET(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	HEAD(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	OPTIONS(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	PATCH(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	POST(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	PUT(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
	TRACE(path string, h echo.HandlerFunc, m ...echo.MiddlewareFunc) *echo.Route
}

// RegisterHandlers adds each server route to the EchoRouter.
func RegisterHandlers(router EchoRouter, si ServerInterface) {
	RegisterHandlersWithBaseURL(router, si, "")
}

// Registers handlers, and prepends BaseURL to the paths, so that the paths
// can be served under a prefix.
func RegisterHandlersWithBaseURL(router EchoRouter, si ServerInterface, baseURL string) {

	wrapper := ServerInterfaceWrapper{
		Handler: si,
	}

	router.POST(baseURL+"/v1/cache/:provider", wrapper.CreateCacheEntry)
	router.PUT(baseURL+"/v1/cache/:provider", wrapper.UpdateCacheEntry)
	router.GET(baseURL+"/v1/cache/:provider/:key", wrapper.GetCacheEntryByKey)

}

// Base64 encoded, gzipped, json marshaled Swagger object
var swaggerSpec = []string{

	"H4sIAAAAAAAC/+RXW2/bNhT+KwS3RzWWncs2v61pkAYtlqBJnoogoKVjm41FsuRRGsXQfx9ISrYutB1v",
	"y259s0WK33c+fueiJU1kpqQAgYaOl9Qkc8iY+3nKkjm8k9/EQrL0QhjUeYJcCrumtFSgkYPbmQHOZWp/",
	"pWASzZXfRt/f3FyRajGiWCigY2pQczGjZUTldGoA7Ws/apjSMf1hsCYzqJgMLv2uMqK5XvQxbj997J9d",
	"RlTD15xrSOn4s3sxqlnelZEP7UygLvrBWAoajKkibaOdrhcJW8yk5jjPaEThiWVqYSnMnrmyYFx8BDHD",
	"OR0fBkKf8gXcG/4MfYhr/gxETgnOgSSWJwFLlHBBJgWCaaIN41FEp1JnDOmYcoEnRx6bZ3lGx8MVMhcI",
	"M9AW+gGKPugHKAKYrcCy4o1belMvNUIcBkJUDOemj3RlHxOUJFfWVwFIjpC5F3tHZlxc+MU1INOaFXbR",
	"zNno+MTYuHuSvv91dHxCkjkkDybPdkU6HB0eHZ/89PMvMZskKUz3/d/W5uQoIM4mK28VZqvgHcvbW27a",
	"LGr5uqlWfVPtvDjVwBA+wdccDAaSxG68hzqDtuVvI9fsFeYL5IppvDe5UlIjBOrGahNZb1oFPJFyAUz0",
	"Ig6dHLWIhiM0SgoD/RB5gNitv5qLd6FytiLQ8O6KbET9td7zdSH1MLXbd2rowZuFuOxmQUcSbhUI4TbJ",
	"tlU5B9wsyR++9bTqIn8i+lAjKvtVYOsldORphrOJ42alblW6LUP2ts/fVZfXeQLIZoEKfXZjH0d73M0V",
	"02jf2lGjQ+70darLKaz0Ppla7w6K3SeyAlyFEmhcGgkXKTwRJlLiNkUdJpZ9WNCX9Etv2k2otN3pD0c7",
	"On0nSHd85Bmuou2XlP/+bHemtdShsS4NDFtuM3FrAXn7w1MGxrDZxoPq5V2kK8B6u6V9uRKsYykREP9M",
	"pERJw+3fuj5UF7znNLjFdSLPJqCDtmNP1ZFxHMe7IAwGMa7t4/2jiHc63eNFTrkqwLvSbuJiKl1b4Ogq",
	"5+XkCyRIXCaQa9CPLtpH0H72p8OD+CB2ZlYgmOJ0TA/do8ZwO3gcDlz5HSyVlo88BV26S5S+KdirZDbA",
	"i9R+PriZo9EbHT+WAYI2dPy5q9FvLFt9C9THH9jSbBctBxpRwTIbTL1Mm2KgziGqvukCE3V55zeDwbcy",
	"LXyaCAThqDOlFjxx5AdfjK8N66Ne1vzbY2RZ+rvypdnpN4qHrwhbdQwH21bWd9OaCvnGcV4P3q3+716c",
	"snyBfxlNX6ACnHIBTwoShJRAtSeiJs8yZuetyjyGsNYoYHM4D1jNN83vyGrteSxotfgVYf9XVvNB9axW",
	"RsF6N1g+QOGq3gwCTjwHXMv1tvgA/7gXo2XoKD+K7unoV/dX84PshebqlId/i6nOAduGIpOCWNFtrv4e",
	"AAD//5lM/+CIFAAA",
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
