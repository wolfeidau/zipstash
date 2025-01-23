package admin

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/rs/zerolog/log"
	"github.com/wolfeidau/zipstash/pkg/trace"

	providerv1 "github.com/wolfeidau/zipstash/api/gen/proto/go/provider/v1"
	provisionv1 "github.com/wolfeidau/zipstash/api/gen/proto/go/provision/v1"
)

type CreateTenantCmd struct {
	Provider string `help:"provider type" default:"github" enum:"github,gitlab,buildkite"`
	TenantID string `help:"tenant id to create" required:""`
	Slug     string `help:"slug of the tenant" required:""`
}

func (c *CreateTenantCmd) Run(ctx context.Context, globals *Globals) error {
	ctx, span := trace.Start(ctx, "CreateTenantCmd.Run")
	defer span.End()

	var prov providerv1.Provider
	switch c.Provider {
	case "github":
		prov = providerv1.Provider_PROVIDER_GITHUB_ACTIONS
	case "gitlab":
		prov = providerv1.Provider_PROVIDER_GITLAB
	case "buildkite":
		prov = providerv1.Provider_PROVIDER_BUILDKITE
	default:
		return fmt.Errorf("invalid provider type: %s", c.Provider)
	}

	res, err := globals.Client.CreateTenant(ctx, &connect.Request[provisionv1.CreateTenantRequest]{
		Msg: &provisionv1.CreateTenantRequest{
			Id:           c.TenantID,
			ProviderType: prov,
			Slug:         c.Slug,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create tenant: %w", err)
	}

	log.Info().Str("id", res.Msg.Id).Str("provider", c.Provider).Msg("created tenant")

	return nil
}
