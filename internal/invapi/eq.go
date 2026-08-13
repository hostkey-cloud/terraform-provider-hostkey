package invapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

func (c *Client) EQShow(ctx context.Context, serverID int) (*ServerShowResponse, error) {
	params := url.Values{}
	params.Set("action", "show")
	params.Set("id", strconv.Itoa(serverID))

	body, err := c.PostForm(ctx, "eq", params)
	if err != nil {
		return nil, fmt.Errorf("eq/show: %w", err)
	}

	var resp ServerShowResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("eq/show decode: %w", err)
	}
	return &resp, nil
}

func (c *Client) EQList(ctx context.Context, filters url.Values) (*ServerListResponse, error) {
	params := url.Values{}
	params.Set("action", "list")
	for k, vv := range filters {
		for _, v := range vv {
			params.Add(k, v)
		}
	}

	body, err := c.PostForm(ctx, "eq", params)
	if err != nil {
		return nil, fmt.Errorf("eq/list: %w", err)
	}

	var resp ServerListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("eq/list decode: %w", err)
	}
	return &resp, nil
}

func (c *Client) EQOrderInstance(ctx context.Context, req OrderInstanceRequest) (*OrderInstanceResponse, error) {
	params := url.Values{}
	params.Set("action", "order_instance")
	params.Set("root_pass", req.RootPass)

	if req.ServerID > 0 {
		params.Set("id", strconv.Itoa(req.ServerID))
	} else if req.Preset != "" {
		params.Set("preset", req.Preset)
	}

	if req.LocationName != "" {
		params.Set("location_name", req.LocationName)
	}
	if req.OSID > 0 {
		params.Set("os_id", strconv.Itoa(req.OSID))
	}
	if req.SoftID > 0 {
		params.Set("soft_id", strconv.Itoa(req.SoftID))
	}
	if req.TrafficPlan > 0 {
		params.Set("traffic_plan", strconv.Itoa(req.TrafficPlan))
	}
	if req.Hostname != "" {
		params.Set("hostname", req.Hostname)
	}
	if req.SSHKey != "" {
		params.Set("ssh_key", req.SSHKey)
	}
	if req.PostInstallScript != "" {
		params.Set("post_install_script", req.PostInstallScript)
	}
	if req.DeployPeriod != "" {
		params.Set("deploy_period", req.DeployPeriod)
	}
	if req.DeployNotify != nil {
		if *req.DeployNotify {
			params.Set("deploy_notify", "true")
		} else {
			params.Set("deploy_notify", "false")
		}
	}
	if req.OwnOS {
		params.Set("own_os", "1")
	}
	if req.RootSize > 0 {
		params.Set("root_size", strconv.Itoa(req.RootSize))
	}
	if req.IPv4Amount > 0 {
		params.Set("ipv4_amount", strconv.Itoa(req.IPv4Amount))
	}
	if req.VLAN > 0 {
		params.Set("vlan", strconv.Itoa(req.VLAN))
	}
	if req.PrivateVLAN > 0 {
		params.Set("private_vlan", strconv.Itoa(req.PrivateVLAN))
	}
	if req.CustomDomain != "" {
		params.Set("custom_domain", req.CustomDomain)
	}
	if req.OSTemplate != "" {
		params.Set("os_template", req.OSTemplate)
	}
	if req.DeployOptions != "" {
		params.Set("deploy_options", req.DeployOptions)
	}
	for k, v := range req.Extra {
		if k == "" || k == "action" || k == "token" {
			continue
		}
		if params.Get(k) == "" {
			params.Set(k, v)
		}
	}

	body, err := c.PostForm(ctx, "eq", params)
	if err != nil {
		return nil, fmt.Errorf("eq/order_instance: %w", err)
	}

	var resp OrderInstanceResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("eq/order_instance decode: %w; body=%s", err, truncate(body, 512))
	}
	resp.RawBody = string(body)

	// Some InvAPI builds return id as string.
	if resp.ID == 0 {
		var alt struct {
			ID any `json:"id"`
		}
		if json.Unmarshal(body, &alt) == nil {
			switch v := alt.ID.(type) {
			case float64:
				resp.ID = int(v)
			case string:
				if n, err := strconv.Atoi(v); err == nil {
					resp.ID = n
				}
			}
		}
	}

	return &resp, nil
}

func (c *Client) EQRenameServer(ctx context.Context, serverID int, hostname string) error {
	params := url.Values{}
	params.Set("action", "rename_server")
	params.Set("id", strconv.Itoa(serverID))
	params.Set("hostname", hostname)

	_, err := c.PostForm(ctx, "eq", params)
	if err != nil {
		return fmt.Errorf("eq/rename_server: %w", err)
	}
	return nil
}

func (c *Client) WHMCSRequestCancellation(ctx context.Context, serverID int, reason string, cancellationType *int) error {
	params := url.Values{}
	params.Set("action", "request_cancellation")
	params.Set("id", strconv.Itoa(serverID))
	if reason != "" {
		params.Set("cancellation_reason", reason)
	}
	if cancellationType != nil {
		params.Set("cancellation_type", strconv.Itoa(*cancellationType))
	}

	_, err := c.PostForm(ctx, "whmcs", params)
	if err != nil {
		return fmt.Errorf("whmcs/request_cancellation: %w", err)
	}
	return nil
}

func (c *Client) EQUpdateServers(ctx context.Context) (*UpdateServersResponse, error) {
	params := url.Values{}
	params.Set("action", "update_servers")
	body, err := c.PostForm(ctx, "eq", params)
	if err != nil {
		return nil, fmt.Errorf("eq/update_servers: %w", err)
	}
	var resp UpdateServersResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("eq/update_servers decode: %w", err)
	}
	return &resp, nil
}

// WaitForNewServerID refreshes the session server list and waits until a new id appears.
func (c *Client) WaitForNewServerID(ctx context.Context, known map[int]struct{}, opts WaitOptions) (int, error) {
	interval := opts.PollInterval
	if interval <= 0 {
		interval = 15 * time.Second
	}
	timeout := opts.Timeout
	if timeout <= 0 {
		timeout = 90 * time.Minute
	}
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		upd, err := c.EQUpdateServers(ctx)
		if err != nil {
			return 0, err
		}

		// Prefer deploy_keys from update_servers (billing id → callback key).
		for _, key := range upd.DeployKeysMap() {
			if key == "" {
				continue
			}
			check, cbErr := c.CallbackCheck(ctx, key)
			if cbErr != nil {
				continue
			}
			if done, termErr := callbackTerminal(check); done && termErr == nil {
				if len(check.Context) > 0 {
					var cb CallbackContext
					if json.Unmarshal(check.Context, &cb) == nil && cb.ID != "" {
						if id, err := strconv.Atoi(cb.ID); err == nil {
							return id, nil
						}
					}
				}
			}
		}

		list, err := c.EQList(ctx, nil)
		if err != nil {
			return 0, err
		}
		ids, err := list.IDs()
		if err != nil {
			return 0, err
		}
		for _, id := range ids {
			if _, ok := known[id]; !ok {
				return id, nil
			}
		}
		if time.Now().After(deadline) {
			return 0, fmt.Errorf("timed out waiting for new server id after %s; known=%v current=%v", timeout, keysOf(known), ids)
		}
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		case <-ticker.C:
		}
	}
}

func keysOf(m map[int]struct{}) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func MainIPv4(show *ServerShowResponse) string {
	for _, ip := range show.IP {
		if ip.MainIP == 1 && ip.IP != "" {
			return ip.IP
		}
	}
	if len(show.IP) > 0 {
		return show.IP[0].IP
	}
	return ""
}
