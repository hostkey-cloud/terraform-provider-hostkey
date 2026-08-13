package invapi

import (
	"encoding/json"
	"testing"
)

func TestNetAddIPv4Response_ParsedIPs(t *testing.T) {
	var r NetAddIPv4Response
	_ = json.Unmarshal([]byte(`{"ips":[{"ip":"1.2.3.4","vlan":100}]}`), &r)
	ips := r.ParsedIPs()
	if len(ips) != 1 || ips[0].IP != "1.2.3.4" || ips[0].VLAN != 100 {
		t.Fatalf("objects: %+v", ips)
	}

	_ = json.Unmarshal([]byte(`{"ips":["10.0.0.1"]}`), &r)
	ips = r.ParsedIPs()
	if len(ips) != 1 || ips[0].IP != "10.0.0.1" {
		t.Fatalf("strings: %+v", ips)
	}
}
