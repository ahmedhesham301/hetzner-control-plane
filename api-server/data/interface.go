package data

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

type Service interface {
	Validate() error
	Provision(ctx context.Context, imageID int64, env string, allowAllFirewallID *int64, networkID int64) (error, *hcloud.Server)
	GetConfigMap() map[string]any
	GetPackerBuildArgs(templatesPath string, env string, networkID int64) []string
	SaveToDB(ctx context.Context, server hcloud.Server) error
	CreateRecord(ctx context.Context) (error, *int)
}

func GetConfigMapString(p Service) map[string]string {
	result := make(map[string]string)
	for k, v := range p.GetConfigMap() {
		result[k] = fmt.Sprintf("%v", v)
	}
	return result
}

func ParseService(body []byte) (Service, error) {
	var kind struct {
		Type   string `json:"type"`
		Engine string `json:"engine"`
	}
	if err := json.Unmarshal(body, &kind); err != nil {
		return nil, err
	}

	var service Service
	switch {
	case kind.Type == "database" && kind.Engine == "postgresql":
		service = &postgresql{}
	default:
		return nil, fmt.Errorf(
			"unsupported service: %s/%s", kind.Type, kind.Engine,
		)
	}

	if err := json.Unmarshal(body, service); err != nil {
		return nil, err
	}
	if err := service.Validate(); err != nil {
		return nil, err
	}
	return service, nil
}
