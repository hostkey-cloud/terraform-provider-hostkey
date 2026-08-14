package provider

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func validateServerImportID(id string) diag.Diagnostics {
	var diags diag.Diagnostics
	id = strings.TrimSpace(id)
	if id == "" {
		diags.AddError("Invalid import id", "Server id must not be empty.")
		return diags
	}
	if strings.HasPrefix(id, pendingIDPrefix) {
		diags.AddError("Invalid import id", "Cannot import pending:<invoice> placeholders; wait until deploy completes.")
		return diags
	}
	n, err := strconv.Atoi(id)
	if err != nil || n <= 0 {
		diags.AddError("Invalid import id", fmt.Sprintf("Import hostkey_server by numeric InvAPI id; got %q.", id))
		return diags
	}
	return diags
}

func validateServerIPImportID(id string) diag.Diagnostics {
	var diags diag.Diagnostics
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		diags.AddError("Invalid import id", "Use <server_id>/<ip>, e.g. 5860/1.2.3.4")
		return diags
	}
	serverID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || serverID <= 0 {
		diags.AddError("Invalid server_id", fmt.Sprintf("server_id must be a positive integer; got %q.", parts[0]))
		return diags
	}
	if err := validateIPv4(parts[1]); err != nil {
		diags.AddError("Invalid ip", err.Error())
	}
	return diags
}

func validateDNSDomainImportID(id string) diag.Diagnostics {
	var diags diag.Diagnostics
	id = strings.TrimSpace(id)
	if id == "" {
		diags.AddError("Invalid import id", "Use numeric domain id or zone FQDN.")
		return diags
	}
	if n, err := strconv.Atoi(id); err == nil {
		if n <= 0 {
			diags.AddError("Invalid import id", fmt.Sprintf("domain id must be a positive integer; got %q.", id))
		}
		return diags
	}
	if !dnsZoneRe.MatchString(id) {
		diags.AddError("Invalid import id", fmt.Sprintf("import by numeric id or zone FQDN; got %q.", id))
	}
	return diags
}

func validateDNSRecordImportID(id string) diag.Diagnostics {
	var diags diag.Diagnostics
	parts := strings.SplitN(id, "/", 4)
	if len(parts) != 4 || parts[0] == "" || parts[1] == "" || parts[2] == "" || parts[3] == "" {
		diags.AddError("Invalid import id", "Use zone/name/type/content (e.g. example.com/www/A/1.2.3.4).")
		return diags
	}
	if !dnsZoneRe.MatchString(parts[0]) {
		diags.AddError("Invalid zone", fmt.Sprintf("zone must be a valid FQDN; got %q.", parts[0]))
	}
	if !dnsLabelRe.MatchString(parts[1]) {
		diags.AddError("Invalid name", fmt.Sprintf("name must be @ or a DNS label; got %q.", parts[1]))
	}
	typ := strings.ToUpper(parts[2])
	found := false
	for _, allowed := range dnsRecordTypes {
		if typ == allowed {
			found = true
			break
		}
	}
	if !found {
		diags.AddError("Invalid type", fmt.Sprintf("type must be one of: %s; got %q.", strings.Join(dnsRecordTypes, ", "), parts[2]))
	}
	diags.Append(validateDNSRecordFields(dnsRecordModel{
		Zone:    types.StringValue(parts[0]),
		Name:    types.StringValue(parts[1]),
		Type:    types.StringValue(typ),
		Content: types.StringValue(parts[3]),
	})...)
	return diags
}

func validateSSHKeyImportID(id string) diag.Diagnostics {
	var diags diag.Diagnostics
	id = strings.TrimSpace(id)
	n, err := strconv.Atoi(id)
	if err != nil || n <= 0 {
		diags.AddError("Invalid import id", fmt.Sprintf("Import hostkey_ssh_key by numeric InvAPI id; got %q.", id))
	}
	return diags
}
