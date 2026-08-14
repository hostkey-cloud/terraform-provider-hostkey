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

	_, err = matchNamedID("Debian", items)
	if err == nil {
		t.Fatal("expected not found for substring Debian (exact match required)")
	}

	_, err = matchNamedID("Windows", items)
	if err == nil {
		t.Fatal("expected not found")
	}
}

func TestMatchTrafficPlan(t *testing.T) {
	items := []trafficNamedID{
		{ID: 12, Name: "1Gbps 50TB", Price: 0},
		{ID: 33, Name: "1Gbps 50TB", Price: 0},
		{ID: 14, Name: "1Gbps unmetered", Price: 100},
		{ID: 35, Name: "1Gbps unmetered", Price: 10000},
	}

	id, err := matchTrafficPlan("1Gbps 50TB - FREE", items)
	if err == nil {
		t.Fatal("expected ambiguous FREE when two rows share name+price 0")
	}

	id, err = matchTrafficPlan("1Gbps 50TB - FREE", []trafficNamedID{
		{ID: 12, Name: "1Gbps 50TB", Price: 0},
		{ID: 35, Name: "1Gbps unmetered", Price: 10000},
	})
	if err != nil || id != 12 {
		t.Fatalf("unique free: id=%d err=%v", id, err)
	}

	id, err = matchTrafficPlan("1Gbps unmetered (10000 P)", items)
	if err != nil || id != 35 {
		t.Fatalf("10000P: id=%d err=%v", id, err)
	}

	id, err = matchTrafficPlan("1Gbps unmetered (100 P)", items)
	if err != nil || id != 14 {
		t.Fatalf("100P: id=%d err=%v", id, err)
	}

	_, err = matchTrafficPlan("1Gbps unmetered", items)
	if err == nil {
		t.Fatal("expected ambiguous unmetered without price hint")
	}

	_, err = matchTrafficPlan("1Gbps 50TB", items)
	if err == nil {
		t.Fatal("expected ambiguous duplicate same-price name without traffic_plan_id")
	}
}
