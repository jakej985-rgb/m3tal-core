package models

// VPNStatus represents the current state of Gluetun VPN connection.
type VPNStatus struct {
	Connected     bool   `json:"connected"`
	Provider      string `json:"provider"`
	ExternalIP    string `json:"external_ip"`
	Region        string `json:"region"`
	ForwardedPort int    `json:"forwarded_port,omitempty"`
	StatusText    string `json:"status_text"`
}

// VPNLeakReport represents the check result of VPN leak check.
type VPNLeakReport struct {
	Leak              bool     `json:"leak"`
	HostIP            string   `json:"host_ip"`
	VPNIP             string   `json:"vpn_ip"`
	StoppedContainers []string `json:"stopped_containers,omitempty"`
}
