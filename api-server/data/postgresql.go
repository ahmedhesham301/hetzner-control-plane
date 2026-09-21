package data

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/ahmedhesham301/hetzner-control-plane/api-server/utils"
	"github.com/ahmedhesham301/hetzner-control-plane/modules/database"
	"github.com/ahmedhesham301/hetzner-control-plane/modules/hetzner"
	"github.com/ahmedhesham301/hetzner-control-plane/modules/random"
	"github.com/ahmedhesham301/hetzner-control-plane/modules/services"
	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

var templates = map[string]map[string]bool{
	"database": {
		"postgresql": true,
	},
}

type ServiceInfo struct {
	AppName     string         `json:"app_name"`
	AppVersions []string       `json:"app_versions"`
	Vars        map[string]any `json:"vars"`
}

type MonitoringConfig struct {
	NodeExporter     bool `json:"node_exporter"`
	Service_exporter bool `json:"service_exporter"`
}

type NetworkConfig struct {
	PublicIPv4     bool `json:"public_ipv4"`
	PublicIPv6     bool `json:"public_ipv6"`
	PrivateNetwork bool `json:"private_network"`
}

type postgresql struct {
	ServiceType string   `json:"type"`
	Engine      string   `json:"engine"`
	Version     string   `json:"version"`
	ServerType  string   `json:"server_type"`
	Location    string   `json:"location"`
	FirewallIDs *[]int64 `json:"firewalls_ids"`

	Network NetworkConfig `json:"network"`

	Monitoring MonitoringConfig `json:"monitoring"`

	Config *map[string]any `json:"config"`

	ExtraLabels *map[string]string `json:"extra_labels"`

	RecordID *int
}

// Provision implements [Service].
func (p *postgresql) Provision(ctx context.Context, imageID int64, env string, allowAllFirewallID *int64, networkID int64) (error, *hcloud.Server) {
	serverOpts := hcloud.ServerCreateOpts{
		Name:       random.AddRandomLetters("postgresql" + "-" + p.Version),
		ServerType: &hcloud.ServerType{Name: p.ServerType},
		Image:      &hcloud.Image{ID: imageID},
		Location:   &hcloud.Location{Name: p.Location},
		PublicNet: &hcloud.ServerCreatePublicNet{
			EnableIPv4: p.Network.PublicIPv4,
			EnableIPv6: p.Network.PublicIPv6,
		},
		Labels: services.AppendManagedLabel(GetConfigMapString(p)),
	}

	var firewalls []*hcloud.ServerCreateFirewall
	if p.FirewallIDs != nil {
		for _, id := range *p.FirewallIDs {
			firewalls = append(firewalls, &hcloud.ServerCreateFirewall{
				Firewall: hcloud.Firewall{
					ID: id,
				},
			})
		}
	}

	if env == "dev" {
		serverOpts.PublicNet.EnableIPv4 = true
		serverOpts.PublicNet.EnableIPv6 = true
		firewalls = append(firewalls, &hcloud.ServerCreateFirewall{
			Firewall: hcloud.Firewall{
				ID: *allowAllFirewallID,
			},
		})
	}
	serverOpts.Firewalls = firewalls
	if p.Network.PrivateNetwork {
		serverOpts.Networks = []*hcloud.Network{
			{

				ID: networkID,
			},
		}
	}
	createResult, _, err := hetzner.HClient.Server.Create(ctx, serverOpts)
	return err, createResult.Server
}

func (p *postgresql) Validate() error {
	// handle if type does not exist
	_, ok := templates[p.ServiceType]
	if !ok {
		return errors.New("unsupported type")
	}

	// handle if engine does not exist
	_, ok = templates[p.ServiceType][p.Engine]
	if !ok {
		return errors.New("unsupported engine")
	}
	// handle if version does not exist

	// one of publicIPv4 or privateNetwork has to be true
	if !(p.Network.PrivateNetwork || p.Network.PublicIPv4) {
		return errors.New("one of publicIPv4 or privateNetwork must to be true")
	}
	return nil
}

func (p *postgresql) SaveToDB(ctx context.Context, server hcloud.Server) error {
	env := os.Getenv("ENV")

	var ip string
	if env == "dev" {
		ip = server.PublicNet.IPv4.IP.String()
	} else {
		ip = server.PrivateNet[0].IP.String()
	}

	query := `UPDATE services SET server_id=$1, server_name=$2, location=$3, firewalls_ids = $4,private_ip=$5, node_exporter=$6, service_exporter=$7
	WHERE id = $8;`
	_, err := database.Pool.Exec(
		ctx, query,
		server.ID, server.Name, server.Location.Name, p.FirewallIDs, ip, p.Monitoring.NodeExporter, p.Monitoring.Service_exporter, p.RecordID,
	)
	return err
}

func (p *postgresql) CreateRecord(ctx context.Context) (error, *int) {
	query := `INSERT INTO services (type, engine)
	VALUES ($1, $2)
	RETURNING id;`
	row := database.Pool.QueryRow(
		ctx, query,
		p.ServiceType,
		p.Engine,
	)
	err := row.Scan(&p.RecordID)
	return err, p.RecordID
}

func (p *postgresql) GetConfigMap() map[string]any {
	return map[string]any{
		"type":             p.ServiceType,
		"engine":           p.Engine,
		"version":          p.Version,
		"node_exporter":    p.Monitoring.NodeExporter,
		"service_exporter": p.Monitoring.Service_exporter,
	}
}

func (p *postgresql) GetPackerBuildArgs(templatesPath string, env string, networkID int64) []string {
	return []string{
		"build", "-machine-readable",
		"-var", fmt.Sprintf("config=%v", utils.ConvertMapToJsonString(p.GetConfigMap())),
		"-var", fmt.Sprintf("env=%v", env),
		"-var", fmt.Sprintf("networkID=%v", networkID),
		templatesPath + "/database/postgresql/main.pkr.hcl",
	}
}
