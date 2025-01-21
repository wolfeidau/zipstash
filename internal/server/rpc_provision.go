package server

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/rs/zerolog/log"
	v1 "github.com/wolfeidau/zipstash/api/gen/proto/go/provision/v1"
	"github.com/wolfeidau/zipstash/internal/index"
	"github.com/wolfeidau/zipstash/internal/provider"
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

	var err error
	switch prov := req.Msg.Provider.(type) {
	case *v1.CreateTenantRequest_Buildkite:
		err = ps.store.PutTenant(ctx, req.Msg.Id, index.TenantRecord{
			ID:           req.Msg.Id,
			ProviderType: provider.Buildkite,
			Provider: index.Buildkite{
				OrganizationSlug: prov.Buildkite.OrganizationSlug,
			},
		})
	case *v1.CreateTenantRequest_GithubActions:
		err = ps.store.PutTenant(ctx, req.Msg.Id, index.TenantRecord{
			ID:           req.Msg.Id,
			ProviderType: provider.GitHubActions,
			Provider: index.GitHubActions{
				Owner: prov.GithubActions.Owner,
			},
		})
	case *v1.CreateTenantRequest_Gitlab:
		err = ps.store.PutTenant(ctx, req.Msg.Id, index.TenantRecord{
			ID:           req.Msg.Id,
			ProviderType: provider.Buildkite,
			Provider: index.GitLab{
				Owner: prov.Gitlab.Owner,
			},
		})
	default:
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("not implemented"))
	}

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
