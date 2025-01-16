package index

import "time"

type CacheRecord struct {
	ID                string    `json:"id"`
	Paths             string    `json:"path"`
	Name              string    `json:"name"`
	Branch            string    `json:"branch"`
	Sha256            string    `json:"sha256"`
	Compression       string    `json:"compression"`
	FileSize          int64     `json:"file_size"`
	MultipartUploadId *string   `json:"multipart_upload_id"`
	Identity          *Identity `json:"identity"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type Identity struct {
	Subject  string   `json:"subject"`
	Issuer   string   `json:"issuer"`
	Audience []string `json:"audience"`
}
