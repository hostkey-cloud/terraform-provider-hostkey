package invapi

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var hddCountPrefix = regexp.MustCompile(`(?i)^\s*(\d+)\s*[x×]`)

// DiskCount infers physical drive count from presets/list hdd + description.
// InvAPI does not send a disk_count field: hdd is either "2x960" or a capacity like "1000".
func DiskCount(hdd, description string) int {
	if n := nxPrefix(hdd); n > 0 {
		return n
	}
	if n := diskCountFromDescription(description); n > 0 {
		return n
	}
	if strings.TrimSpace(hdd) != "" {
		return 1
	}
	return 0
}

func diskCountFromDescription(desc string) int {
	desc = strings.TrimSpace(desc)
	if i := strings.Index(desc, " + "); i >= 0 {
		desc = desc[:i]
	}
	parts := strings.Split(desc, "/")
	if len(parts) < 3 {
		return 0
	}
	last := strings.TrimSpace(parts[len(parts)-1])
	if n := nxPrefix(last); n > 0 {
		return n
	}
	if last != "" {
		return 1
	}
	return 0
}

func nxPrefix(s string) int {
	m := hddCountPrefix.FindStringSubmatch(s)
	if m == nil {
		return 0
	}
	n, err := strconv.Atoi(m[1])
	if err != nil || n < 1 {
		return 0
	}
	return n
}

// ValidateDiskMirror checks InvAPI disk_mirror against catalog disk count.
// 1 disk: RAID type is empty in the panel — omit the field (hba is not processed).
// 2+ disks: hba/raid0/raid1; raid10 needs 4+.
func ValidateDiskMirror(mirror string, disks int, dedicated bool) error {
	mirror = strings.ToLower(strings.TrimSpace(mirror))
	if mirror == "" {
		return nil
	}
	if !dedicated {
		return fmt.Errorf("disk_mirror is only valid on dedicated presets (catalog virtual=0)")
	}
	if disks < 1 {
		return fmt.Errorf("cannot verify disk_mirror: catalog hdd/description did not yield a disk count")
	}
	if disks < 2 {
		return fmt.Errorf("omit disk_mirror: catalog shows %d disk (panel RAID type is empty; sending hba/raid* is not processed)", disks)
	}
	switch mirror {
	case "hba", "raid0", "raid1":
		return nil
	case "raid10":
		if disks < 4 {
			return fmt.Errorf("disk_mirror=raid10 needs 4+ disks; catalog shows %d", disks)
		}
		return nil
	default:
		return fmt.Errorf("unsupported disk_mirror %q", mirror)
	}
}
