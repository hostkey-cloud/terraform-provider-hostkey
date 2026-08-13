package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/hostkey-cloud/terraform-provider-hostkey/internal/invapi"
)

const (
	pendingIDPrefix = "pending:"
	privateKnownKey = "known_server_ids"
)

type privateData interface {
	SetKey(context.Context, string, []byte) diag.Diagnostics
	GetKey(context.Context, string) ([]byte, diag.Diagnostics)
}

func pendingID(invoice int) string {
	return fmt.Sprintf("%s%d", pendingIDPrefix, invoice)
}

func parsePendingInvoice(id string) (int, bool) {
	if !strings.HasPrefix(id, pendingIDPrefix) {
		return 0, false
	}
	raw := strings.TrimPrefix(id, pendingIDPrefix)
	if strings.HasPrefix(raw, "billing-") {
		return 0, false
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		return 0, false
	}
	return n, true
}

func listKnownIDs(list *invapi.ServerListResponse) map[int]struct{} {
	known := map[int]struct{}{}
	if list == nil {
		return known
	}
	ids, err := list.IDs()
	if err != nil {
		return known
	}
	for _, id := range ids {
		known[id] = struct{}{}
	}
	return known
}

func setPrivateKnownIDs(ctx context.Context, priv privateData, known map[int]struct{}) error {
	if priv == nil {
		return nil
	}
	ids := keysInt(known)
	b, err := json.Marshal(ids)
	if err != nil {
		return err
	}
	diags := priv.SetKey(ctx, privateKnownKey, b)
	if diags.HasError() {
		return fmt.Errorf("set private known ids: %s", diags[0].Detail())
	}
	return nil
}

func getPrivateKnownIDs(ctx context.Context, priv privateData) (map[int]struct{}, diag.Diagnostics) {
	out := map[int]struct{}{}
	var diags diag.Diagnostics
	if priv == nil {
		return out, diags
	}
	val, d := priv.GetKey(ctx, privateKnownKey)
	diags.Append(d...)
	if diags.HasError() || len(val) == 0 {
		return out, diags
	}
	var ids []int
	if err := json.Unmarshal(val, &ids); err != nil {
		diags.AddWarning("private state", "could not decode known_server_ids")
		return out, diags
	}
	for _, id := range ids {
		out[id] = struct{}{}
	}
	return out, diags
}
