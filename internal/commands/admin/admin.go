package admin

import (
	"github.com/wolfeidau/zipstash/api/gen/proto/go/provision/v1/provisionv1connect"
)

type Globals struct {
	Debug   bool
	Version string
	Client  provisionv1connect.ProvisionServiceClient
}
