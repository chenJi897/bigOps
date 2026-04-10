package service

import "testing"

func TestNormalizeGoldenDimension(t *testing.T) {
	cases := map[string]string{
		"service":   "service",
		"interface": "interface",
		"instance":  "instance",
		"operator":  "operator",
		"invalid":   "service",
		"":          "service",
	}
	for input, expect := range cases {
		got := normalizeGoldenDimension(input)
		if got != expect {
			t.Fatalf("normalizeGoldenDimension(%q) = %q, want %q", input, got, expect)
		}
	}
}
