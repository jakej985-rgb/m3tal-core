package api

import (
	"encoding/json"
	"net/http"

	"github.com/jakej985-rgb/m3tal-core/core/networking/vpn"
)

// VPNHandlers provides HTTP handlers for Gluetun VPN Manager operations.
type VPNHandlers struct{}

// NewVPNHandlers creates a new instance of VPNHandlers.
func NewVPNHandlers() *VPNHandlers {
	return &VPNHandlers{}
}

// GetStatus returns the connection status and settings of the VPN.
// GET /api/v2/vpn/status
func (h *VPNHandlers) GetStatus(w http.ResponseWriter, r *http.Request) {
	mgr, err := vpn.NewManager()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "VPN_INIT_FAILED", "failed to initialize VPN manager: "+err.Error(), nil)
		return
	}

	status, err := mgr.GetStatus()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "VPN_STATUS_FAILED", err.Error(), nil)
		return
	}

	sendSuccess(w, http.StatusOK, status, nil)
}

// ControlVPN handles starting, stopping, or restarting the VPN container.
// POST /api/v2/vpn/control
func (h *VPNHandlers) ControlVPN(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Action string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "INVALID_JSON", "invalid JSON body", nil)
		return
	}

	mgr, err := vpn.NewManager()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "VPN_INIT_FAILED", "failed to initialize VPN manager: "+err.Error(), nil)
		return
	}

	var opErr error
	switch req.Action {
	case "start":
		opErr = mgr.Start()
	case "stop":
		opErr = mgr.Stop()
	case "restart":
		opErr = mgr.Restart()
	default:
		sendError(w, http.StatusBadRequest, "VALIDATION_FAILED", "invalid action: must be 'start', 'stop', or 'restart'", nil)
		return
	}

	if opErr != nil {
		sendError(w, http.StatusInternalServerError, "VPN_ACTION_FAILED", opErr.Error(), nil)
		return
	}

	sendSuccess(w, http.StatusOK, map[string]string{"status": "success", "action": req.Action}, nil)
}

// SwitchRegion handles switching the VPN region.
// POST /api/v2/vpn/region
func (h *VPNHandlers) SwitchRegion(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Region string `json:"region"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "INVALID_JSON", "invalid JSON body", nil)
		return
	}

	if req.Region == "" {
		sendError(w, http.StatusBadRequest, "VALIDATION_FAILED", "region is required", nil)
		return
	}

	mgr, err := vpn.NewManager()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "VPN_INIT_FAILED", "failed to initialize VPN manager: "+err.Error(), nil)
		return
	}

	if err := mgr.SwitchRegion(req.Region); err != nil {
		sendError(w, http.StatusInternalServerError, "VPN_REGION_FAILED", err.Error(), nil)
		return
	}

	sendSuccess(w, http.StatusOK, map[string]string{"status": "success", "region": req.Region}, nil)
}

// SyncPort handles manual port sync.
// POST /api/v2/vpn/sync-port
func (h *VPNHandlers) SyncPort(w http.ResponseWriter, r *http.Request) {
	mgr, err := vpn.NewManager()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "VPN_INIT_FAILED", "failed to initialize VPN manager: "+err.Error(), nil)
		return
	}

	port, err := mgr.SyncForwardedPort()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "VPN_SYNC_FAILED", err.Error(), nil)
		return
	}

	sendSuccess(w, http.StatusOK, map[string]any{"status": "success", "forwarded_port": port}, nil)
}

// CheckLeak handles leak check.
// GET /api/v2/vpn/check-leak
func (h *VPNHandlers) CheckLeak(w http.ResponseWriter, r *http.Request) {
	mgr, err := vpn.NewManager()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "VPN_INIT_FAILED", "failed to initialize VPN manager: "+err.Error(), nil)
		return
	}

	isLeak, hostIP, vpnIP, err := mgr.CheckLeak()
	if err != nil {
		sendError(w, http.StatusInternalServerError, "VPN_LEAK_CHECK_FAILED", err.Error(), nil)
		return
	}

	// Automatic kill-switch trigger on leak detection
	var stoppedContainers []string
	if isLeak {
		stoppedContainers, _ = mgr.StopDependentContainers()
	}

	sendSuccess(w, http.StatusOK, map[string]any{
		"leak":               isLeak,
		"host_ip":            hostIP,
		"vpn_ip":             vpnIP,
		"stopped_containers": stoppedContainers,
	}, nil)
}
