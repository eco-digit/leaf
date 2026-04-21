package collector_test

import (
	"testing"

	"github.com/eco-digit/leaf/internal/collector"
	"github.com/stretchr/testify/assert"
)

func TestParseSCSFlavor(t *testing.T) {
	cases := []struct {
		flavor string
		vcpus  int
		ramGB  int
	}{
		{"SCS-1L-1", 1, 1},
		{"SCS-4V-16-50", 4, 16},
		{"SCS-2T-8-20s", 2, 8},
		{"SCS-16V-64", 16, 64},
		{"m1.small", 0, 0}, // not a scs flavor
		{"", 0, 0},
	}
	for _, tc := range cases {
		vcpus, ramGB := collector.ParseSCSFlavor(tc.flavor)
		assert.Equalf(t, tc.vcpus, vcpus, "vcpus for %q", tc.flavor)
		assert.Equalf(t, tc.ramGB, ramGB, "ramGB for %q", tc.flavor)
	}
}
