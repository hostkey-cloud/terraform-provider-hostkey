package invapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type DNSDomain struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type DNSRecord struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Content  string `json:"content"`
	TTL      int    `json:"ttl"`
	Priority int    `json:"priority"`
}

type pdnsEnvelope struct {
	Result string          `json:"result"`
	Module string          `json:"module"`
	Action string          `json:"action"`
	Data   json.RawMessage `json:"data"`
	Error  string          `json:"error"`
}

func (c *Client) PDNSListDomains(ctx context.Context) ([]DNSDomain, error) {
	params := url.Values{}
	params.Set("action", "list_domains")
	body, err := c.PostForm(ctx, "pdns", params)
	if err != nil {
		return nil, fmt.Errorf("pdns/list_domains: %w", err)
	}
	var env pdnsEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("pdns/list_domains decode: %w", err)
	}
	var domains []DNSDomain
	if len(env.Data) > 0 && string(env.Data) != "null" {
		if err := json.Unmarshal(env.Data, &domains); err != nil {
			return nil, fmt.Errorf("pdns/list_domains data: %w; body=%s", err, truncate(body, 512))
		}
	}
	return domains, nil
}

func (c *Client) PDNSAddDomain(ctx context.Context, name string, serverID int) (*DNSDomain, error) {
	params := url.Values{}
	params.Set("action", "add_domain")
	params.Set("params[name]", name)
	if serverID > 0 {
		params.Set("params[server_id]", strconv.Itoa(serverID))
	}
	body, err := c.PostForm(ctx, "pdns", params)
	if err != nil {
		return nil, fmt.Errorf("pdns/add_domain: %w", err)
	}
	var env pdnsEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("pdns/add_domain decode: %w", err)
	}
	var data struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return nil, fmt.Errorf("pdns/add_domain data: %w; body=%s", err, truncate(body, 512))
	}
	if data.ID == 0 {
		return nil, fmt.Errorf("pdns/add_domain: empty id; body=%s", truncate(body, 512))
	}
	return &DNSDomain{ID: data.ID, Name: name}, nil
}

func (c *Client) PDNSDeleteDomain(ctx context.Context, id int, zone string) error {
	params := url.Values{}
	params.Set("action", "delete_domain")
	params.Set("params[id]", strconv.Itoa(id))
	params.Set("params[zone]", zone)
	_, err := c.PostForm(ctx, "pdns", params)
	if err != nil {
		return fmt.Errorf("pdns/delete_domain: %w", err)
	}
	return nil
}

func (c *Client) PDNSViewZone(ctx context.Context, zone string) ([]DNSRecord, int, error) {
	params := url.Values{}
	params.Set("action", "view_zone")
	params.Set("params[zone]", zone)
	body, err := c.PostForm(ctx, "pdns", params)
	if err != nil {
		return nil, 0, fmt.Errorf("pdns/view_zone: %w", err)
	}
	var env pdnsEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, 0, fmt.Errorf("pdns/view_zone decode: %w", err)
	}
	var data struct {
		ID      int         `json:"id"`
		Name    string      `json:"name"`
		Records []DNSRecord `json:"records"`
	}
	if err := json.Unmarshal(env.Data, &data); err != nil {
		return nil, 0, fmt.Errorf("pdns/view_zone data: %w; body=%s", err, truncate(body, 512))
	}
	return data.Records, data.ID, nil
}

type PDNSAddDNSRequest struct {
	Zone     string
	Name     string
	Type     string
	Content  []string
	TTL      int
	Priority int
}

func (c *Client) PDNSAddDNS(ctx context.Context, req PDNSAddDNSRequest) error {
	params := url.Values{}
	params.Set("action", "add_dns")
	params.Set("params[zone]", req.Zone)
	params.Set("params[name]", req.Name)
	params.Set("params[type]", req.Type)
	for _, ctn := range req.Content {
		params.Add("params[content][]", ctn)
	}
	if req.TTL > 0 {
		params.Set("params[ttl]", strconv.Itoa(req.TTL))
	}
	if req.Priority > 0 {
		params.Set("params[priority]", strconv.Itoa(req.Priority))
	}
	_, err := c.PostForm(ctx, "pdns", params)
	if err != nil {
		return fmt.Errorf("pdns/add_dns: %w", err)
	}
	return nil
}

type PDNSDeleteDNSRequest struct {
	Zone     string
	Name     string
	Type     string
	Content  string
	Priority int
}

func (c *Client) PDNSDeleteDNS(ctx context.Context, req PDNSDeleteDNSRequest) error {
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return fmt.Errorf("pdns/delete_dns: content is required (type-only delete would remove all %s records on the name)", req.Type)
	}
	params := url.Values{}
	params.Set("action", "delete_dns")
	params.Set("params[zone]", req.Zone)
	if req.Name != "" {
		params.Set("params[name]", req.Name)
	}
	params.Set("params[type]", req.Type)
	params.Add("params[content][]", content)
	if req.Priority > 0 {
		params.Set("params[priority]", strconv.Itoa(req.Priority))
	}
	_, err := c.PostForm(ctx, "pdns", params)
	if err != nil {
		return fmt.Errorf("pdns/delete_dns: %w", err)
	}
	return nil
}
