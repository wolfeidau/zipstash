// Code generated by protoc-gen-connect-go. DO NOT EDIT.
//
// Source: cache/v1/cache.proto

package cachev1connect

import (
	connect "connectrpc.com/connect"
	context "context"
	errors "errors"
	v1 "github.com/wolfeidau/zipstash/api/gen/proto/go/cache/v1"
	http "net/http"
	strings "strings"
)

// This is a compile-time assertion to ensure that this generated file and the connect package are
// compatible. If you get a compiler error that this constant is not defined, this code was
// generated with a version of connect newer than the one compiled into your binary. You can fix the
// problem by either regenerating this code with an older version of connect or updating the connect
// version compiled into your binary.
const _ = connect.IsAtLeastVersion1_13_0

const (
	// CacheServiceName is the fully-qualified name of the CacheService service.
	CacheServiceName = "cache.v1.CacheService"
)

// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as Spec.Procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.
const (
	// CacheServiceCreateEntryProcedure is the fully-qualified name of the CacheService's CreateEntry
	// RPC.
	CacheServiceCreateEntryProcedure = "/cache.v1.CacheService/CreateEntry"
	// CacheServiceUpdateEntryProcedure is the fully-qualified name of the CacheService's UpdateEntry
	// RPC.
	CacheServiceUpdateEntryProcedure = "/cache.v1.CacheService/UpdateEntry"
	// CacheServiceGetEntryProcedure is the fully-qualified name of the CacheService's GetEntry RPC.
	CacheServiceGetEntryProcedure = "/cache.v1.CacheService/GetEntry"
)

// CacheServiceClient is a client for the cache.v1.CacheService service.
type CacheServiceClient interface {
	// CreateEntry creates a new cache entry
	CreateEntry(context.Context, *connect.Request[v1.CreateEntryRequest]) (*connect.Response[v1.CreateEntryResponse], error)
	// UpdateEntry updates an existing cache entry
	UpdateEntry(context.Context, *connect.Request[v1.UpdateEntryRequest]) (*connect.Response[v1.UpdateEntryResponse], error)
	// GetEntry retrieves a cache entry by key
	GetEntry(context.Context, *connect.Request[v1.GetEntryRequest]) (*connect.Response[v1.GetEntryResponse], error)
}

// NewCacheServiceClient constructs a client for the cache.v1.CacheService service. By default, it
// uses the Connect protocol with the binary Protobuf Codec, asks for gzipped responses, and sends
// uncompressed requests. To use the gRPC or gRPC-Web protocols, supply the connect.WithGRPC() or
// connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Connect or gRPC server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewCacheServiceClient(httpClient connect.HTTPClient, baseURL string, opts ...connect.ClientOption) CacheServiceClient {
	baseURL = strings.TrimRight(baseURL, "/")
	cacheServiceMethods := v1.File_cache_v1_cache_proto.Services().ByName("CacheService").Methods()
	return &cacheServiceClient{
		createEntry: connect.NewClient[v1.CreateEntryRequest, v1.CreateEntryResponse](
			httpClient,
			baseURL+CacheServiceCreateEntryProcedure,
			connect.WithSchema(cacheServiceMethods.ByName("CreateEntry")),
			connect.WithClientOptions(opts...),
		),
		updateEntry: connect.NewClient[v1.UpdateEntryRequest, v1.UpdateEntryResponse](
			httpClient,
			baseURL+CacheServiceUpdateEntryProcedure,
			connect.WithSchema(cacheServiceMethods.ByName("UpdateEntry")),
			connect.WithClientOptions(opts...),
		),
		getEntry: connect.NewClient[v1.GetEntryRequest, v1.GetEntryResponse](
			httpClient,
			baseURL+CacheServiceGetEntryProcedure,
			connect.WithSchema(cacheServiceMethods.ByName("GetEntry")),
			connect.WithClientOptions(opts...),
		),
	}
}

// cacheServiceClient implements CacheServiceClient.
type cacheServiceClient struct {
	createEntry *connect.Client[v1.CreateEntryRequest, v1.CreateEntryResponse]
	updateEntry *connect.Client[v1.UpdateEntryRequest, v1.UpdateEntryResponse]
	getEntry    *connect.Client[v1.GetEntryRequest, v1.GetEntryResponse]
}

// CreateEntry calls cache.v1.CacheService.CreateEntry.
func (c *cacheServiceClient) CreateEntry(ctx context.Context, req *connect.Request[v1.CreateEntryRequest]) (*connect.Response[v1.CreateEntryResponse], error) {
	return c.createEntry.CallUnary(ctx, req)
}

// UpdateEntry calls cache.v1.CacheService.UpdateEntry.
func (c *cacheServiceClient) UpdateEntry(ctx context.Context, req *connect.Request[v1.UpdateEntryRequest]) (*connect.Response[v1.UpdateEntryResponse], error) {
	return c.updateEntry.CallUnary(ctx, req)
}

// GetEntry calls cache.v1.CacheService.GetEntry.
func (c *cacheServiceClient) GetEntry(ctx context.Context, req *connect.Request[v1.GetEntryRequest]) (*connect.Response[v1.GetEntryResponse], error) {
	return c.getEntry.CallUnary(ctx, req)
}

// CacheServiceHandler is an implementation of the cache.v1.CacheService service.
type CacheServiceHandler interface {
	// CreateEntry creates a new cache entry
	CreateEntry(context.Context, *connect.Request[v1.CreateEntryRequest]) (*connect.Response[v1.CreateEntryResponse], error)
	// UpdateEntry updates an existing cache entry
	UpdateEntry(context.Context, *connect.Request[v1.UpdateEntryRequest]) (*connect.Response[v1.UpdateEntryResponse], error)
	// GetEntry retrieves a cache entry by key
	GetEntry(context.Context, *connect.Request[v1.GetEntryRequest]) (*connect.Response[v1.GetEntryResponse], error)
}

// NewCacheServiceHandler builds an HTTP handler from the service implementation. It returns the
// path on which to mount the handler and the handler itself.
//
// By default, handlers support the Connect, gRPC, and gRPC-Web protocols with the binary Protobuf
// and JSON codecs. They also support gzip compression.
func NewCacheServiceHandler(svc CacheServiceHandler, opts ...connect.HandlerOption) (string, http.Handler) {
	cacheServiceMethods := v1.File_cache_v1_cache_proto.Services().ByName("CacheService").Methods()
	cacheServiceCreateEntryHandler := connect.NewUnaryHandler(
		CacheServiceCreateEntryProcedure,
		svc.CreateEntry,
		connect.WithSchema(cacheServiceMethods.ByName("CreateEntry")),
		connect.WithHandlerOptions(opts...),
	)
	cacheServiceUpdateEntryHandler := connect.NewUnaryHandler(
		CacheServiceUpdateEntryProcedure,
		svc.UpdateEntry,
		connect.WithSchema(cacheServiceMethods.ByName("UpdateEntry")),
		connect.WithHandlerOptions(opts...),
	)
	cacheServiceGetEntryHandler := connect.NewUnaryHandler(
		CacheServiceGetEntryProcedure,
		svc.GetEntry,
		connect.WithSchema(cacheServiceMethods.ByName("GetEntry")),
		connect.WithHandlerOptions(opts...),
	)
	return "/cache.v1.CacheService/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case CacheServiceCreateEntryProcedure:
			cacheServiceCreateEntryHandler.ServeHTTP(w, r)
		case CacheServiceUpdateEntryProcedure:
			cacheServiceUpdateEntryHandler.ServeHTTP(w, r)
		case CacheServiceGetEntryProcedure:
			cacheServiceGetEntryHandler.ServeHTTP(w, r)
		default:
			http.NotFound(w, r)
		}
	})
}

// UnimplementedCacheServiceHandler returns CodeUnimplemented from all methods.
type UnimplementedCacheServiceHandler struct{}

func (UnimplementedCacheServiceHandler) CreateEntry(context.Context, *connect.Request[v1.CreateEntryRequest]) (*connect.Response[v1.CreateEntryResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("cache.v1.CacheService.CreateEntry is not implemented"))
}

func (UnimplementedCacheServiceHandler) UpdateEntry(context.Context, *connect.Request[v1.UpdateEntryRequest]) (*connect.Response[v1.UpdateEntryResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("cache.v1.CacheService.UpdateEntry is not implemented"))
}

func (UnimplementedCacheServiceHandler) GetEntry(context.Context, *connect.Request[v1.GetEntryRequest]) (*connect.Response[v1.GetEntryResponse], error) {
	return nil, connect.NewError(connect.CodeUnimplemented, errors.New("cache.v1.CacheService.GetEntry is not implemented"))
}