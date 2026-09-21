package temporal

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"os/exec"

	"github.com/ahmedhesham301/hetzner-control-plane/api-server/data"
	"github.com/ahmedhesham301/hetzner-control-plane/api-server/utils"

	"github.com/ahmedhesham301/hetzner-control-plane/modules/hetzner"
	"github.com/ahmedhesham301/hetzner-control-plane/modules/services"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"go.temporal.io/sdk/activity"
)

func checkImageExist(ctx context.Context, serviceJSON json.RawMessage) (*int64, error) {
	service, _ := data.ParseService(serviceJSON)
	images, err := hetzner.HClient.Image.AllWithOpts(ctx, hcloud.ImageListOpts{
		LabelSelector: services.ConvertToHetznerLabels(
			service.GetConfigMap(),
		),
	})

	if err != nil {
		return nil, err
	}
	if len(images) == 0 {
		return nil, nil
	}
	return &images[0].ID, nil

}

type buildImageParams struct {
	ServiceJSON   json.RawMessage
	Env           string
	TemplatesPath string
	NetworkID     int64
}

func buildImage(ctx context.Context, params buildImageParams) (*int64, error) {
	service, err := data.ParseService(params.ServiceJSON)
	if err != nil {
		return nil, err
	}
	// Template files belong to the worker; retries may carry an outdated path.
	templatesPath := os.Getenv("PACKER_TEMPLATES_PATH")
	if templatesPath == "" {
		templatesPath = params.TemplatesPath
	}
	args := service.GetPackerBuildArgs(templatesPath, params.Env, params.NetworkID)

	logger := activity.GetLogger(ctx)
	cmd := exec.CommandContext(ctx, "packer", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		logger.Error("packer build command failed", "err", err, "output", string(output))
		return nil, err
	}
	logger.Info("packer build command output", "output", string(output))

	id, err := utils.GetSnapshotID(output)
	return &id, err
}

type deployServiceParams struct {
	ServiceJSON        json.RawMessage
	ImageID            int64
	Env                string
	AllowAllFirewallID *int64
	NetworkID          int64
}

func deployService(ctx context.Context, params deployServiceParams) error {
	service, _ := data.ParseService(params.ServiceJSON)

	err, server := service.Provision(ctx, params.ImageID, params.Env, params.AllowAllFirewallID, params.NetworkID)
	if err != nil {
		return err
	}
	return service.SaveToDB(ctx, *server)

}

func GetOrCreateAllowAllFirewall(ctx context.Context) (*int64, error) {
	firewall, _, err := hetzner.HClient.Firewall.GetByName(ctx, "allow_all")
	if err != nil {
		return nil, err
	}
	if firewall == nil {
		allIPv4 := net.IPNet{
			IP:   net.IPv4zero,
			Mask: net.CIDRMask(0, 32),
		}
		opts := hcloud.FirewallCreateOpts{
			Name: "allow_all",
			Rules: []hcloud.FirewallRule{
				{
					Direction: "in",
					SourceIPs: []net.IPNet{allIPv4},
					Port:      new("any"),
					Protocol:  hcloud.FirewallRuleProtocolTCP,
				},
				{
					Direction:      "out",
					DestinationIPs: []net.IPNet{allIPv4},
					Port:           new("any"),
					Protocol:       hcloud.FirewallRuleProtocolTCP,
				},
			},
		}
		firewallResult, _, err := hetzner.HClient.Firewall.Create(ctx, opts)
		if err != nil {
			return nil, err
		}
		firewall = firewallResult.Firewall
	}
	return &firewall.ID, nil
}
