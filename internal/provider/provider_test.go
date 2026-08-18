//go:build acceptance

package provider

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
)

func testAccPreCheck(t *testing.T) {
	t.Helper()
	if os.Getenv("TF_ACC") == "" {
		t.Skip("set TF_ACC=1 to run acceptance tests")
	}
	if os.Getenv("HOSTKEY_API_KEY") == "" && os.Getenv("HOSTKEY_API_TOKEN") == "" {
		t.Fatal("HOSTKEY_API_KEY or HOSTKEY_API_TOKEN must be set for acceptance tests")
	}
}

func testAccProtoV6ProviderFactories() map[string]func() (tfprotov6.ProviderServer, error) {
	return map[string]func() (tfprotov6.ProviderServer, error){
		"hostkey": providerserver.NewProtocol6WithError(New("test")()),
	}
}

func testAccLocation(t *testing.T) string {
	t.Helper()
	if v := os.Getenv("HOSTKEY_ACC_LOCATION"); v != "" {
		return v
	}
	return "NL"
}

func testAccPreset(t *testing.T) string {
	t.Helper()
	if v := os.Getenv("HOSTKEY_ACC_PRESET"); v != "" {
		return v
	}
	return "vm.pico"
}

func testAccRootPass(t *testing.T) string {
	t.Helper()
	if v := os.Getenv("HOSTKEY_ACC_ROOT_PASS"); v != "" {
		return v
	}
	return "TfAccPass1%"
}
