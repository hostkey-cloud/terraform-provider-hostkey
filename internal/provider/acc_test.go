package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccSSHKey_basic(t *testing.T) {
	testAccPreCheck(t)

	name := fmt.Sprintf("tf-acc-%s", acctest.RandString(8))
	// Fixed ed25519 public key material (not a private secret).
	pub := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIKaccsshkeyfortestonlydonotuseanywhereelse tf-acc@hostkey"

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccSSHKeyConfig(name, pub),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hostkey_ssh_key.test", "name", name),
					resource.TestCheckResourceAttrSet("hostkey_ssh_key.test", "id"),
				),
			},
			{
				ResourceName:            "hostkey_ssh_key.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"key"},
			},
		},
	})
}

func testAccSSHKeyConfig(name, pub string) string {
	return fmt.Sprintf(`
provider "hostkey" {
  region = "RU"
}

resource "hostkey_ssh_key" "test" {
  name    = %q
  key     = %q
  default = false
}
`, name, pub)
}

func TestAccServer_basic(t *testing.T) {
	testAccPreCheck(t)

	location := testAccLocation(t)
	preset := testAccPreset(t)
	rootPass := testAccRootPass(t)

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: testAccServerConfig(location, preset, rootPass),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("hostkey_server.test", "id"),
					resource.TestMatchResourceAttr("hostkey_server.test", "id", regexp.MustCompile(`^\d+$`)),
					resource.TestCheckResourceAttrSet("hostkey_server.test", "main_ipv4"),
				),
			},
			{
				ResourceName:            "hostkey_server.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"root_pass", "cancellation_type", "cancellation_reason", "timeouts"},
			},
		},
	})
}

func testAccServerConfig(location, preset, rootPass string) string {
	return fmt.Sprintf(`
provider "hostkey" {
  region = "RU"
}

resource "hostkey_server" "test" {
  preset_name       = %q
  location_name     = %q
  os_name           = "Ubuntu 22.04"
  traffic_plan_name = "3 TB / 1 Gbps VM"
  deploy_period     = "monthly"
  root_pass         = %q
  power_state       = "on"
  cancellation_type = 1
  cancellation_reason = "terraform acceptance test"

  tags = {
    managed = "tf-acc"
  }

  timeouts {
    create = "90m"
    delete = "30m"
  }
}
`, preset, location, rootPass)
}

func TestAccDNSDomain_basic(t *testing.T) {
	testAccPreCheck(t)

	domain := os.Getenv("HOSTKEY_ACC_DNS_DOMAIN")
	if domain == "" {
		t.Skip("set HOSTKEY_ACC_DNS_DOMAIN to a disposable zone name you can create/delete in pdns")
	}

	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories(),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
provider "hostkey" {
  region = "RU"
}

resource "hostkey_dns_domain" "test" {
  name = %q
}
`, domain),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("hostkey_dns_domain.test", "name", domain),
					resource.TestCheckResourceAttrSet("hostkey_dns_domain.test", "id"),
				),
			},
			{
				ResourceName:      "hostkey_dns_domain.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
