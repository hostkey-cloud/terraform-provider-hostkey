package invapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

type SSHKey struct {
	ID         int    `json:"id"`
	CustomerID int    `json:"customer_id"`
	Name       string `json:"name"`
	Key        string `json:"key"`
	Default    int    `json:"default"`
	Created    string `json:"created"`
}

type SSHKeysListResponse struct {
	Result  string   `json:"result"`
	Module  string   `json:"module"`
	Action  string   `json:"action"`
	SSHKeys []SSHKey `json:"sshkeys"`
}

type SSHKeyMutateResponse struct {
	Result string `json:"result"`
	Module string `json:"module"`
	Action string `json:"action"`
	SSHKey SSHKey `json:"sshkey"`
}

func (c *Client) SSHKeysList(ctx context.Context) ([]SSHKey, error) {
	params := url.Values{}
	params.Set("action", "list")
	body, err := c.PostForm(ctx, "ssh_keys", params)
	if err != nil {
		return nil, fmt.Errorf("ssh_keys/list: %w", err)
	}
	var resp SSHKeysListResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("ssh_keys/list decode: %w", err)
	}
	return resp.SSHKeys, nil
}

func (c *Client) SSHKeyGet(ctx context.Context, id int) (*SSHKey, error) {
	keys, err := c.SSHKeysList(ctx)
	if err != nil {
		return nil, err
	}
	for i := range keys {
		if keys[i].ID == id {
			return &keys[i], nil
		}
	}
	return nil, fmt.Errorf("ssh_keys: key id %d not found", id)
}

// SSHKeyAdd stores a public key in InvAPI SSH key storage.
// InvAPI expects nested form fields params[name] and params[key].
func (c *Client) SSHKeyAdd(ctx context.Context, name, publicKey string, isDefault bool) (*SSHKey, error) {
	params := url.Values{}
	params.Set("action", "add")
	params.Set("params[name]", name)
	params.Set("params[key]", publicKey)
	if isDefault {
		params.Set("params[default]", "1")
	} else {
		params.Set("params[default]", "0")
	}

	body, err := c.PostForm(ctx, "ssh_keys", params)
	if err != nil {
		return nil, fmt.Errorf("ssh_keys/add: %w", err)
	}
	var resp SSHKeyMutateResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("ssh_keys/add decode: %w; body=%s", err, truncate(body, 512))
	}
	if resp.SSHKey.ID == 0 {
		return nil, fmt.Errorf("ssh_keys/add: empty id in response: %s", truncate(body, 512))
	}
	return &resp.SSHKey, nil
}

func (c *Client) SSHKeyDelete(ctx context.Context, id int) error {
	params := url.Values{}
	params.Set("action", "delete")
	params.Set("id", strconv.Itoa(id))
	_, err := c.PostForm(ctx, "ssh_keys", params)
	if err != nil {
		return fmt.Errorf("ssh_keys/delete: %w", err)
	}
	return nil
}
