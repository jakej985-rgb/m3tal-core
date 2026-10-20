package vpn

// Supported VPN Provider names as expected by Gluetun
const (
	ProviderPIA     = "private internet access"
	ProviderMullvad = "mullvad"
	ProviderNordVPN = "nordvpn"
	ProviderCustom  = "custom"
)

// VPNStatus represents the current status of the VPN connection.
type VPNStatus struct {
	Connected     bool   `json:"connected"`
	Provider      string `json:"provider"`
	ExternalIP    string `json:"external_ip"`
	Region        string `json:"region"`
	ForwardedPort int    `json:"forwarded_port,omitempty"`
	StatusText    string `json:"status_text"`
}

// VPNConfig represents the configuration variables for the VPN connection.
type VPNConfig struct {
	Provider      string `json:"provider"`
	User          string `json:"user,omitempty"`
	Password      string `json:"password,omitempty"`
	Regions       string `json:"regions,omitempty"`
	PortForwarded bool   `json:"port_forwarded,omitempty"`
}
