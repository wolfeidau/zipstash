package index

import (
	"fmt"
	"slices"
	"time"

	"github.com/wolfeidau/zipstash/internal/provider"
)

type CacheRecord struct {
	ID                string    `json:"id"`
	Paths             string    `json:"path"`
	Provider          string    `json:"provider"`
	Owner             string    `json:"owner"`
	Name              string    `json:"name"`
	Branch            string    `json:"branch"`
	Sha256            string    `json:"sha256"`
	Compression       string    `json:"compression"`
	FileSize          int64     `json:"file_size"`
	MultipartUploadId *string   `json:"multipart_upload_id"`
	Identity          *Identity `json:"identity"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type TenantRecord struct {
	ID           string   `json:"id"`
	ProviderType string   `json:"provider_type"`
	Provider     Provider `json:"provider"`
}

func (r *TenantRecord) Validate() error {
	if !slices.Contains(provider.DefaultProviders, r.ProviderType) {
		return fmt.Errorf("invalid provider_type: `%s`", r.ProviderType)
	}

	if r.Provider == nil {
		return fmt.Errorf("provider is required")
	}

	return nil
}

type Provider interface {
	Key() string
}

type GitHubActions struct {
	Owner string `json:"owner"`
}

func (gh GitHubActions) Key() string {
	return fmt.Sprintf("%s#%s", provider.GitHubActions, gh.Owner)
}

type GitLab struct {
	Owner string `json:"owner"`
}

func (gl GitLab) Key() string {
	return fmt.Sprintf("%s#%s", provider.GitLab, gl.Owner)
}

type Buildkite struct {
	OrganizationSlug string `json:"organization_slug"`
}

func (bk Buildkite) Key() string {
	return fmt.Sprintf("%s#%s", provider.Buildkite, bk.OrganizationSlug)
}

type Identity struct {
	Subject  string   `json:"subject"`
	Issuer   string   `json:"issuer"`
	Audience []string `json:"audience"`
}

func BuildkiteProviderKey(org, pipeline string) string {
	return fmt.Sprintf("%s#%s", provider.Buildkite, org)
}

func GitHubActionsProviderKey(owner, repository string) string {
	return fmt.Sprintf("%s#%s", provider.GitHubActions, owner)
}

func GitLabProviderKey(owner, project string) string {
	return fmt.Sprintf("%s#%s", provider.GitLab, owner)
}

func providerKey(provider, owner string) string {
	return fmt.Sprintf("%s#%s", provider, owner)
}
