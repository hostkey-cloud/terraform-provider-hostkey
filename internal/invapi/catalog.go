package invapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type OSListResponse struct {
	Result     string          `json:"result"`
	OSList     []OSEntry       `json:"os_list"`
	OSExcluded json.RawMessage `json:"os_excluded"`
}

type OSEntry struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Active int    `json:"active"`
}

type TrafficPlansListResponse struct {
	Result       string        `json:"result"`
	TrafficPlans []TrafficPlan `json:"traffic_plans"`
}

type TrafficPlan struct {
	ID        int     `json:"id"`
	Name      string  `json:"name"`
	Active    int     `json:"active"`
	Location  string  `json:"location"`
	Locations string  `json:"locations"`
	Price     float64 `json:"price"`
	MainPlan  int     `json:"main_plan"`
}

type PresetsListFilter struct {
	Location string
}

type OSListFilter struct {
	Location   string
	ServerID   int
	InstanceID int // preset id
	BillPeriod string
}

type TrafficPlansListFilter struct {
	Location   string
	ServerID   int
	InstanceID int // preset id
}

type SoftwareListFilter struct {
	Location   string
	ServerID   int
	InstanceID int
	BillPeriod string
}

type SoftwareListResponse struct {
	Result   string          `json:"result"`
	Software []SoftwareEntry `json:"software"`
}

type SoftwareEntry struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Active int    `json:"active"`
}

func (c *Client) PresetsList(ctx context.Context, filter PresetsListFilter) (*PresetListResponse, error) {
	params := url.Values{}
	params.Set("action", "list")
	if filter.Location != "" {
		params.Set("location", filter.Location)
	}
	body, err := c.PostForm(ctx, "presets", params)
	if err != nil {
		return nil, fmt.Errorf("presets/list: %w", err)
	}
	var resp PresetListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("presets/list decode: %w", err)
	}
	return &resp, nil
}

func (c *Client) OSList(ctx context.Context, filter OSListFilter) (*OSListResponse, error) {
	params := url.Values{}
	params.Set("action", "list")
	if filter.Location != "" {
		params.Set("location", filter.Location)
	}
	if filter.ServerID > 0 {
		params.Set("id", strconv.Itoa(filter.ServerID))
	}
	if filter.InstanceID > 0 {
		params.Set("instance_id", strconv.Itoa(filter.InstanceID))
	}
	if filter.BillPeriod != "" {
		params.Set("bill_period", filter.BillPeriod)
	}
	body, err := c.PostForm(ctx, "os", params)
	if err != nil {
		return nil, fmt.Errorf("os/list: %w", err)
	}
	var resp OSListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("os/list decode: %w", err)
	}
	return &resp, nil
}

func (c *Client) TrafficPlansList(ctx context.Context, filter TrafficPlansListFilter) (*TrafficPlansListResponse, error) {
	params := url.Values{}
	params.Set("action", "list")
	if filter.Location != "" {
		params.Set("location", filter.Location)
	}
	if filter.ServerID > 0 {
		params.Set("id", strconv.Itoa(filter.ServerID))
	}
	if filter.InstanceID > 0 {
		params.Set("instance", strconv.Itoa(filter.InstanceID))
	}
	// InvAPI quirk: traffic_plans/list works in public mode (no token).
	// Sending a Customer session token often returns "invalid request".
	// Docs: https://hostkey.com/documentation/apidocs/traffic_plans/#traffic_planslist (RU: https://hostkey.ru/documentation/apidocs/traffic_plans/#traffic_planslist)
	body, err := c.PostFormWithoutAuth(ctx, "traffic_plans", params)
	if err != nil {
		return nil, fmt.Errorf("traffic_plans/list: %w", err)
	}
	var resp TrafficPlansListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("traffic_plans/list decode: %w", err)
	}
	if !strings.EqualFold(resp.Result, "OK") && resp.Result != "" {
		return nil, fmt.Errorf("traffic_plans/list: result=%s", resp.Result)
	}
	return &resp, nil
}

func (c *Client) SoftwareList(ctx context.Context, filter SoftwareListFilter) (*SoftwareListResponse, error) {
	params := url.Values{}
	params.Set("action", "list")
	if filter.Location != "" {
		params.Set("location", filter.Location)
	}
	if filter.ServerID > 0 {
		params.Set("id", strconv.Itoa(filter.ServerID))
	}
	if filter.InstanceID > 0 {
		params.Set("instance_id", strconv.Itoa(filter.InstanceID))
	}
	if filter.BillPeriod != "" {
		params.Set("bill_period", filter.BillPeriod)
	}
	body, err := c.PostForm(ctx, "software", params)
	if err != nil {
		return nil, fmt.Errorf("software/list: %w", err)
	}
	var resp SoftwareListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("software/list decode: %w", err)
	}
	return &resp, nil
}
