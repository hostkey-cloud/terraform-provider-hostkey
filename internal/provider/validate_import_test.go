package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestValidateServerImportID(t *testing.T) {
	if diags := validateServerImportID("123"); diags.HasError() {
		t.Fatalf("valid id: %v", diags.Errors())
	}
	for _, id := range []string{"", "pending:1", "abc"} {
		if diags := validateServerImportID(id); !diags.HasError() {
			t.Fatalf("expected error for %q", id)
		}
	}
}

func TestValidateServerIPImportID(t *testing.T) {
	if diags := validateServerIPImportID("100/1.2.3.4"); diags.HasError() {
		t.Fatalf("valid: %v", diags.Errors())
	}
	for _, id := range []string{"100", "100/not-ip"} {
		if diags := validateServerIPImportID(id); !diags.HasError() {
			t.Fatalf("expected error for %q", id)
		}
	}
}

func TestValidateDNSRecordImportID(t *testing.T) {
	id := "example.com/www/A/1.2.3.4"
	if diags := validateDNSRecordImportID(id); diags.HasError() {
		t.Fatalf("valid: %v", diags.Errors())
	}
	if diags := validateDNSRecordImportID("bad"); !diags.HasError() {
		t.Fatal("expected error for malformed id")
	}
}

func TestValidateDNSRecordFields(t *testing.T) {
	diags := validateDNSRecordFields(dnsRecordModel{
		Zone:    types.StringValue("example.com"),
		Name:    types.StringValue("www"),
		Type:    types.StringValue("A"),
		Content: types.StringValue("not-an-ip"),
	})
	if !diags.HasError() {
		t.Fatal("expected A record content error")
	}
}
