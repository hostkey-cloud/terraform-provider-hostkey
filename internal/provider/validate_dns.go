package provider

import (
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
)

func validateDNSRecordFields(plan dnsRecordModel) diag.Diagnostics {
	var diags diag.Diagnostics

	zone := strings.TrimSpace(plan.Zone.ValueString())
	if zone != "" && !dnsZoneRe.MatchString(zone) {
		diags.AddAttributeError(path.Root("zone"), "Invalid zone", "zone must be a valid FQDN (e.g. example.com).")
	}

	typ := strings.ToUpper(strings.TrimSpace(plan.Type.ValueString()))
	content := strings.TrimSpace(plan.Content.ValueString())
	if content == "" {
		diags.AddAttributeError(path.Root("content"), "Invalid content", "content must not be empty.")
		return diags
	}

	switch typ {
	case "A":
		if err := validateIPv4(content); err != nil {
			diags.AddAttributeError(path.Root("content"), "Invalid A record", err.Error())
		}
	case "AAAA":
		if err := validateIPv6(content); err != nil {
			diags.AddAttributeError(path.Root("content"), "Invalid AAAA record", err.Error())
		}
	case "CNAME", "MX", "NS", "PTR":
		if !hostnameRe.MatchString(content) && !dnsZoneRe.MatchString(content) {
			diags.AddAttributeError(path.Root("content"), "Invalid "+typ+" record",
				fmt.Sprintf("%s content must be a hostname or FQDN; got %q.", typ, content))
		}
	}

	if !plan.Priority.IsNull() {
		p := plan.Priority.ValueInt64()
		if p < 0 || p > maxDNSPriority {
			diags.AddAttributeError(path.Root("priority"), "Invalid priority",
				fmt.Sprintf("priority must be between 0 and %d; got %d.", maxDNSPriority, p))
		}
	}

	if !plan.TTL.IsNull() {
		ttl := plan.TTL.ValueInt64()
		if ttl < minDNSTTL || ttl > maxDNSTTL {
			diags.AddAttributeError(path.Root("ttl"), "Invalid ttl",
				fmt.Sprintf("ttl must be between %d and %d; got %d.", minDNSTTL, maxDNSTTL, ttl))
		}
	}

	return diags
}
