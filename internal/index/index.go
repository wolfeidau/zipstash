package index

import (
	"fmt"
	"slices"
	"time"

	"github.com/wolfeidau/zipstash/internal/ciauth"
)

type CacheRecord struct {
	UpdatedAt         time.Time `json:"updated_at"`
	MultipartUploadId *string   `json:"multipart_upload_id"`
	Identity          *Identity `json:"identity"`
	Owner             string    `json:"owner"`
	Paths             string    `json:"path"`
	Provider          string    `json:"provider"`
	Key               string    `json:"id"`
	Name              string    `json:"name"`
	Branch            string    `json:"branch"`
	Checksum          string    `json:"checksum"`
	ChecksumAlgorithm string    `json:"checksum_algorithm"`
	Compression       string    `json:"compression"`
	Architecture      string    `json:"architecture"`
	OperatingSystem   string    `json:"operating_system"`
	FileSize          int64     `json:"file_size"`
	CpuCount          int32     `json:"cpu_count"`
}

type TenantRecord struct {
	ID           string `json:"id"`
	ProviderType string `json:"provider_type"`
	Owner        string `json:"owner"`
}

func (r *TenantRecord) Validate() error {
	if !slices.Contains(ciauth.DefaultProviderNames, r.ProviderType) {
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
