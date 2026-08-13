package invapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

type Tag struct {
	ID          int    `json:"id"`
	Tag         string `json:"tag"`
	Value       string `json:"value"`
	Extra       string `json:"extra"`
	Internal    int    `json:"internal"`
	Component   string `json:"component"`
	ComponentID int    `json:"component_id"`
}

type TagsListResponse struct {
	Result string `json:"result"`
	Tags   []Tag  `json:"tags"`
}

func (c *Client) TagsList(ctx context.Context, serverID int) (*TagsListResponse, error) {
	params := url.Values{}
	params.Set("action", "list")
	params.Set("id", strconv.Itoa(serverID))
	params.Set("component", "eq")
	body, err := c.PostForm(ctx, "tags", params)
	if err != nil {
		return nil, fmt.Errorf("tags/list: %w", err)
	}
	var resp TagsListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("tags/list decode: %w", err)
	}
	return &resp, nil
}

func (c *Client) TagsAdd(ctx context.Context, serverID int, tag, value string) error {
	params := url.Values{}
	params.Set("action", "add")
	params.Set("id", strconv.Itoa(serverID))
	params.Set("tag", tag)
	if value != "" {
		params.Set("value", value)
	}
	_, err := c.PostForm(ctx, "tags", params)
	if err != nil {
		return fmt.Errorf("tags/add: %w", err)
	}
	return nil
}

func (c *Client) TagsRemove(ctx context.Context, serverID int, tag string) error {
	params := url.Values{}
	params.Set("action", "remove")
	params.Set("tag", tag)
	params.Set("component", "eq")
	params.Set("component_id", strconv.Itoa(serverID))
	params.Set("id", strconv.Itoa(serverID))
	_, err := c.PostForm(ctx, "tags", params)
	if err != nil {
		return fmt.Errorf("tags/remove: %w", err)
	}
	return nil
}
