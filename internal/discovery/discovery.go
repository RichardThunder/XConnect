package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"time"
)

// Device represents a peer on the tailnet.
type Device struct {
	HostName string   `json:"hostname,omitempty"`
	IP       string   `json:"ip,omitempty"`
	Addrs    []string `json:"addrs,omitempty"`
}

// Devices returns the list of tailnet devices. It tries tailscale status --json first
// (when Tailscale CLI is installed), then Tailscale API if apiToken is set.
func Devices(ctx context.Context, apiToken string) ([]Device, error) {
	if out, err := tailscaleStatus(ctx); err == nil {
		return parseStatusJSON(out)
	}
	if apiToken != "" {
		return tailscaleAPI(ctx, apiToken)
	}
	return nil, fmt.Errorf("device discovery: run 'tailscale status --json' or set TAILSCALE_API_TOKEN")
}

func tailscaleStatus(ctx context.Context) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "tailscale", "status", "--json")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return out, nil
}

// statusJSON matches the structure of tailscale status --json (Peer and Self).
type statusJSON struct {
	Self struct {
		HostName string `json:"HostName"`
	} `json:"Self"`
	Peer map[string]struct {
		HostName string   `json:"HostName"`
		TailscaleIPs []string `json:"TailscaleIPs"`
	} `json:"Peer"`
}

func parseStatusJSON(data []byte) ([]Device, error) {
	var s statusJSON
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	var list []Device
	// add self
	if s.Self.HostName != "" {
		list = append(list, Device{HostName: s.Self.HostName})
	}
	for _, p := range s.Peer {
		d := Device{HostName: p.HostName, Addrs: p.TailscaleIPs}
		if len(p.TailscaleIPs) > 0 {
			d.IP = p.TailscaleIPs[0]
		}
		list = append(list, d)
	}
	return list, nil
}

// Tailscale API: list devices (requires API token from admin console).
// See https://tailscale.com/kb/1101/api
const apiBase = "https://api.tailscale.com/api/v2"

type apiDevicesResponse struct {
	Devices []struct {
		Name      string   `json:"name"`
		Addresses []string `json:"addresses"`
	} `json:"devices"`
}

func tailscaleAPI(ctx context.Context, token string) ([]Device, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", apiBase+"/tailnet/-/devices", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tailscale API: %s", resp.Status)
	}
	var out apiDevicesResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	list := make([]Device, 0, len(out.Devices))
	for _, d := range out.Devices {
		ip := ""
		if len(d.Addresses) > 0 {
			ip = strings.TrimPrefix(d.Addresses[0], "fd7a:115c:a1e0::1/128")
			if ip == d.Addresses[0] {
				ip = d.Addresses[0]
			}
			// 100.x.x.x format
			for _, a := range d.Addresses {
				if strings.HasPrefix(a, "100.") {
					ip = strings.Split(a, "/")[0]
					break
				}
			}
		}
		list = append(list, Device{HostName: d.Name, IP: ip, Addrs: d.Addresses})
	}
	return list, nil
}

// Format base URL for a device (hostname or IP + port).
func BaseURL(d Device, port string) string {
	if port == "" {
		port = "8315"
	}
	if d.HostName != "" {
		return "http://" + d.HostName + ":" + port
	}
	if d.IP != "" {
		return "http://" + d.IP + ":" + port
	}
	if len(d.Addrs) > 0 {
		addr := d.Addrs[0]
		if i := strings.Index(addr, "/"); i >= 0 {
			addr = addr[:i]
		}
		return "http://" + addr + ":" + port
	}
	return ""
}
