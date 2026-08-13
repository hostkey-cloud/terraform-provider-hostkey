package invapi

import "testing"

func TestPowerStateFromStatus(t *testing.T) {
	cases := map[string]string{
		"":          "",
		"rent":      "on",
		"presale":   "on",
		"power_off": "off",
		"Power_Off": "off",
		"POWEROFF":  "off",
		"off":       "off",
		"stopped":   "off",
	}
	for in, want := range cases {
		if got := PowerStateFromStatus(in); got != want {
			t.Fatalf("PowerStateFromStatus(%q)=%q want %q", in, got, want)
		}
	}
}
