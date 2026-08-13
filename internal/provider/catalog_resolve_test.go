package provider

import "testing"

// Fixture only — exercises matchNamedID (exact / partial / ambiguous / missing).
// Real OS/preset/soft/traffic catalogs come from InvAPI and change over time;
// list them with data.hostkey_oses / presets / software / traffic_plans.
func TestMatchNamedID(t *testing.T) {
	items := []namedID{
		{ID: 187, Name: "Ubuntu 22.04"},
		{ID: 237, Name: "Ubuntu 24.04"},
		{ID: 180, Name: "Debian 11"},
	}

	id, err := matchNamedID("Ubuntu 22.04", items)
	if err != nil || id != 187 {
		t.Fatalf("exact: id=%d err=%v", id, err)
	}

	id, err = matchNamedID("Debian 11", items)
	if err != nil || id != 180 {
		t.Fatalf("exact debian: id=%d err=%v", id, err)
	}

	_, err = matchNamedID("Ubuntu", items)
	if err == nil {
		t.Fatal("expected ambiguous Ubuntu")
	}

	id, err = matchNamedID("Debian", items)
	if err != nil || id != 180 {
		t.Fatalf("partial debian: id=%d err=%v", id, err)
	}

	_, err = matchNamedID("Windows", items)
	if err == nil {
		t.Fatal("expected not found")
	}
}
