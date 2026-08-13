package invapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

type NetIPv4 struct {
	IP   string `json:"ip"`
	VLAN int    `json:"vlan"`
}

type NetAddIPv4Response struct {
	Result string          `json:"result"`
	Action string          `json:"action"`
	ID     int             `json:"id"`
	IPs    json.RawMessage `json:"ips"`
	Keys   json.RawMessage `json:"keys"`
}

func (r *NetAddIPv4Response) ParsedIPs() []NetIPv4 {
	if len(r.IPs) == 0 || string(r.IPs) == "null" {
		return nil
	}
	var asObjs []NetIPv4
	if err := json.Unmarshal(r.IPs, &asObjs); err == nil && len(asObjs) > 0 {
		// Accept only if at least one entry has an IP (string arrays partially decode into empty structs).
		ok := false
		for _, x := range asObjs {
			if x.IP != "" {
				ok = true
				break
			}
		}
		if ok {
			out := make([]NetIPv4, 0, len(asObjs))
			for _, x := range asObjs {
				if x.IP != "" {
					out = append(out, x)
				}
			}
			return out
		}
	}
	var asStrs []string
	if err := json.Unmarshal(r.IPs, &asStrs); err == nil {
		out := make([]NetIPv4, 0, len(asStrs))
		for _, s := range asStrs {
			if s != "" {
				out = append(out, NetIPv4{IP: s})
			}
		}
		return out
	}
	return nil
}

type NetAddIPv4Request struct {
	ServerID int
	Port     string
	IP       string // optional specific address
	Amount   int    // used when IP empty; default 1
}

func (c *Client) NetAddIPv4(ctx context.Context, req NetAddIPv4Request) (*NetAddIPv4Response, error) {
	params := url.Values{}
	params.Set("action", "add_ipv4")
	params.Set("id", strconv.Itoa(req.ServerID))
	if req.Port != "" {
		params.Set("port", req.Port)
	}
	if req.IP != "" {
		params.Add("ips[]", req.IP)
	} else {
		amount := req.Amount
		if amount <= 0 {
			amount = 1
		}
		params.Set("amount", strconv.Itoa(amount))
	}

	body, err := c.PostForm(ctx, "net", params)
	if err != nil {
		return nil, fmt.Errorf("net/add_ipv4: %w", err)
	}
	var resp NetAddIPv4Response
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("net/add_ipv4 decode: %w; body=%s", err, truncate(body, 512))
	}
	return &resp, nil
}

func (c *Client) NetRemoveIPv4(ctx context.Context, serverID int, ip string) error {
	params := url.Values{}
	params.Set("action", "remove_ipv4")
	params.Set("id", strconv.Itoa(serverID))
	params.Set("ip", ip)
	_, err := c.PostForm(ctx, "net", params)
	if err != nil {
		return fmt.Errorf("net/remove_ipv4: %w", err)
	}
	return nil
}

// ServerHasIPv4 reports whether eq/show lists the given IPv4 on the server.
func (c *Client) ServerHasIPv4(ctx context.Context, serverID int, ip string) (bool, error) {
	show, err := c.EQShow(ctx, serverID)
	if err != nil {
		return false, err
	}
	for _, entry := range show.IP {
		if entry.IP == ip {
			return true, nil
		}
	}
	return false, nil
}
