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

type GitHubActions struct {
	TenantID string `help:"tenant id to create" required:""`
	Owner    string `help:"owner of the tenant" required:""`
}

func (c *GitHubActions) Run(ctx context.Context, globals *Globals) error {
	ctx, span := trace.Start(ctx, "GitHubActions.Run")
	defer span.End()

	res, err := globals.Client.CreateTenant(ctx, &connect.Request[provisionv1.CreateTenantRequest]{
		Msg: &provisionv1.CreateTenantRequest{
			Id:           c.TenantID,
			ProviderType: providerv1.Provider_PROVIDER_GITHUB_ACTIONS,
			Provider: &provisionv1.CreateTenantRequest_GithubActions{
				GithubActions: &providerv1.GitHubActions{
					Owner: c.Owner,
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create tenant: %w", err)
	}

	log.Info().Msgf("created tenant: %s", res.Msg.Id)

	return nil
}

type GitLab struct {
	TenantID string `help:"tenant id to create" required:""`
	Owner    string `help:"owner of the tenant" required:""`
}

func (c *GitLab) Run(ctx context.Context, globals *Globals) error {
	ctx, span := trace.Start(ctx, "GitLab.Run")
	defer span.End()

	res, err := globals.Client.CreateTenant(ctx, &connect.Request[provisionv1.CreateTenantRequest]{
		Msg: &provisionv1.CreateTenantRequest{
			Id:           c.TenantID,
			ProviderType: providerv1.Provider_PROVIDER_GITLAB,
			Provider: &provisionv1.CreateTenantRequest_Gitlab{
				Gitlab: &providerv1.GitLab{
					Owner: c.Owner,
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create tenant: %w", err)
	}

	log.Info().Str("id", res.Msg.Id).Msg("created buildkite tenant")

	return nil
}

type Buildkite struct {
	TenantID         string `help:"tenant id to create" required:""`
	OrganizationSlug string `help:"org of the tenant" required:""`
}

func (c *Buildkite) Run(ctx context.Context, globals *Globals) error {
	ctx, span := trace.Start(ctx, "Buildkite.Run")
	defer span.End()

	res, err := globals.Client.CreateTenant(ctx, &connect.Request[provisionv1.CreateTenantRequest]{
		Msg: &provisionv1.CreateTenantRequest{
			Id:           c.TenantID,
			ProviderType: providerv1.Provider_PROVIDER_BUILDKITE,
			Provider: &provisionv1.CreateTenantRequest_Buildkite{
				Buildkite: &providerv1.Buildkite{
					OrganizationSlug: c.OrganizationSlug,
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create tenant: %w", err)
	}

	log.Info().Str("id", res.Msg.Id).Msg("created buildkite tenant")

	return nil
}

type CreateTenantCmd struct {
	GitHubActions GitHubActions `cmd:"" help:"create a github actions tenant"`
	GitLab        GitLab        `cmd:"" help:"create a gitlab ci tenant"`
	Buildkite     Buildkite     `cmd:"" help:"create a buildkite tenant"`
}
