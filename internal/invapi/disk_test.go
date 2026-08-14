package invapi

import "testing"

func TestDiskCount(t *testing.T) {
	cases := []struct {
		hdd, desc string
		want      int
	}{
		{"2x960", "BM E3-12xx/32/2x960GB SSD", 2},
		{"1000", "BM EPYC 3151/32/1TB NVMe", 1},
		{"2x1000", "BM E2288G/128/2x960GB SSD", 2},
		{"2000", "BM EPYC 9354/192/2x1TB NVMe", 2},
		{"8000", "BM EPYC 9354/384/2x3.84TB NVMe", 2},
		{"4x1920", "BM 2xEPYC 7451/384/4x1.92TB SSD", 4},
		{"2000", "BM 2xE5-2680v4/128/2x960", 2},
		{"1000", "BM i9-9900K/64/1TB NVMe + RTX A5000", 1},
		{"", "", 0},
	}
	for _, tc := range cases {
		got := DiskCount(tc.hdd, tc.desc)
		if got != tc.want {
			t.Errorf("DiskCount(%q, %q)=%d want %d", tc.hdd, tc.desc, got, tc.want)
		}
	}
}

func TestValidateDiskMirror(t *testing.T) {
	if err := ValidateDiskMirror("hba", 1, true); err == nil {
		t.Fatal("1-disk hba must error")
	}
	if err := ValidateDiskMirror("raid1", 1, true); err == nil {
		t.Fatal("1-disk raid1 must error")
	}
	if err := ValidateDiskMirror("", 1, true); err != nil {
		t.Fatalf("omit on 1 disk: %v", err)
	}
	if err := ValidateDiskMirror("hba", 2, true); err != nil {
		t.Fatalf("2-disk hba: %v", err)
	}
	if err := ValidateDiskMirror("raid1", 2, true); err != nil {
		t.Fatalf("2-disk raid1: %v", err)
	}
	if err := ValidateDiskMirror("raid10", 2, true); err == nil {
		t.Fatal("raid10 on 2 disks must error")
	}
	if err := ValidateDiskMirror("raid10", 4, true); err != nil {
		t.Fatalf("4-disk raid10: %v", err)
	}
	if err := ValidateDiskMirror("hba", 2, false); err == nil {
		t.Fatal("disk_mirror on VM must error")
	}
}
