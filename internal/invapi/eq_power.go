package invapi

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// EQPowerOn starts a powered-off server (eq/on).
func (c *Client) EQPowerOn(ctx context.Context, serverID int) error {
	return c.eqPowerAction(ctx, "on", serverID)
}

// EQPowerOff soft-stops a server (eq/off).
func (c *Client) EQPowerOff(ctx context.Context, serverID int) error {
	return c.eqPowerAction(ctx, "off", serverID)
}

// EQHardOff force-stops via IPMI/OpenStack (eq/hard_off).
func (c *Client) EQHardOff(ctx context.Context, serverID int) error {
	return c.eqPowerAction(ctx, "hard_off", serverID)
}

// EQReboot reboots a server (eq/reboot).
func (c *Client) EQReboot(ctx context.Context, serverID int) error {
	return c.eqPowerAction(ctx, "reboot", serverID)
}

func (c *Client) eqPowerAction(ctx context.Context, action string, serverID int) error {
	params := url.Values{}
	params.Set("action", action)
	params.Set("id", strconv.Itoa(serverID))
	_, err := c.PostForm(ctx, "eq", params)
	if err != nil {
		return fmt.Errorf("eq/%s: %w", action, err)
	}
	return nil
}

// PowerStateFromStatus maps InvAPI server status to Terraform power_state.
// Known off status: power_off. Everything else that is a live rental is treated as on.
func PowerStateFromStatus(status string) string {
	s := strings.ToLower(strings.TrimSpace(status))
	switch s {
	case "power_off", "poweroff", "off", "stopped":
		return "off"
	default:
		if s == "" {
			return ""
		}
		return "on"
	}
}
