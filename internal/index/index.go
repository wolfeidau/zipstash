package index

import (
	"fmt"
	"slices"
	"time"

	"github.com/wolfeidau/zipstash/internal/provider"
)

type CacheRecord struct {
	UpdatedAt         time.Time `json:"updated_at"`
	MultipartUploadId *string   `json:"multipart_upload_id"`
	Identity          *Identity `json:"identity"`
	ID                string    `json:"id"`
	Paths             string    `json:"path"`
	Provider          string    `json:"provider"`
	Owner             string    `json:"owner"`
	Name              string    `json:"name"`
	Branch            string    `json:"branch"`
	Sha256            string    `json:"sha256"`
	Compression       string    `json:"compression"`
	FileSize          int64     `json:"file_size"`
}

type TenantRecord struct {
	ID           string `json:"id"`
	ProviderType string `json:"provider_type"`
	Owner        string `json:"owner"`
}

func (r *TenantRecord) Validate() error {
	if !slices.Contains(provider.DefaultProviders, r.ProviderType) {
		return fmt.Errorf("invalid provider_type: `%s`", r.ProviderType)
	}

	if r.Owner == "" {
		return fmt.Errorf("provider is required")
	}

	return nil
}

type Identity struct {
	Subject  string   `json:"subject"`
	Issuer   string   `json:"issuer"`
	Audience []string `json:"audience"`
}

func TenantKey(provider, owner string) string {
	return fmt.Sprintf("%s#%s", provider, owner)
}
