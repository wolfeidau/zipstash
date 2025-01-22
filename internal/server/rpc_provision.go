package server

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/rs/zerolog/log"
	v1 "github.com/wolfeidau/zipstash/api/gen/proto/go/provision/v1"
	"github.com/wolfeidau/zipstash/internal/index"
	"go.opentelemetry.io/otel/trace"
)

type ProvisionServiceHandler struct {
	store *index.Store
}

func NewProvisionServiceHandler(store *index.Store) *ProvisionServiceHandler {
	return &ProvisionServiceHandler{
		store: store,
	}
}

func (ps *ProvisionServiceHandler) CreateTenant(ctx context.Context, req *connect.Request[v1.CreateTenantRequest]) (*connect.Response[v1.CreateTenantResponse], error) {
	span := trace.SpanFromContext(ctx)
	span.SetName("Provision.CreateTenant")
	defer span.End()

	err := ps.store.PutTenant(ctx, req.Msg.Id, index.TenantRecord{
		ID:           req.Msg.Id,
		ProviderType: fromProviderV1(req.Msg.ProviderType),
		Owner:        req.Msg.Slug,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to create tenant")
		span.RecordError(err)
		if errors.Is(err, index.ErrAlreadyExists) {
			return nil, connect.NewError(connect.CodeAlreadyExists, errors.New("cache.v1.ProvisionService.CreateTenant already exists"))
		}
		return nil, connect.NewError(connect.CodeInternal, errors.New("cache.v1.ProvisionService.CreateTenant internal error"))
	}

	return connect.NewResponse(&v1.CreateTenantResponse{
		Id: req.Msg.Id,
	}), nil
}

func (ps *ProvisionServiceHandler) GetTenant(ctx context.Context, getTenantReq *connect.Request[v1.GetTenantRequest]) (*connect.Response[v1.GetTenantResponse], error) {
	span := trace.SpanFromContext(ctx)
	span.SetName("Provision.GetTenant")
	defer span.End()

	tenant, err := ps.store.GetTenant(ctx, getTenantReq.Msg.Id)
	if err != nil {
		log.Error().Err(err).Msg("failed to get tenant")
		span.RecordError(err)
		if errors.Is(err, index.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, errors.New("cache.v1.ProvisionService.GetTenant tenant not found"))
		}
		return nil, connect.NewError(connect.CodeInternal, errors.New("cache.v1.ProvisionService.GetTenant internal error"))
	}

	return connect.NewResponse(&v1.GetTenantResponse{
		Id:   tenant.ID,
		Slug: tenant.Owner,
	}), nil
}
