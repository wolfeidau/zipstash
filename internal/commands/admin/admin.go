package admin

import (
	"github.com/wolfeidau/zipstash/api/gen/proto/go/provision/v1/provisionv1connect"
)

type Globals struct {
	Client  provisionv1connect.ProvisionServiceClient
	Version string
	Debug   bool
}
