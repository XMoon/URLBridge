//go:build linux

package guest

import (
	"net"
	"testing"
)

func TestParseLinuxRouteGateways(t *testing.T) {
	t.Parallel()

	input := `Iface	Destination	Gateway	Flags	RefCnt	Use	Metric	Mask	MTU	Window	IRTT
eth0	00000000	0202000A	0003	0	0	100	00000000	0	0	0
eth1	00000000	00000000	0001	0	0	100	00000000	0	0	0
eth2	0011A8C0	0101A8C0	0003	0	0	100	00000000	0	0	0
`

	got, err := parseLinuxRouteGateways(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || !got[0].Equal(net.ParseIP("10.0.2.2")) {
		t.Fatalf("got %v want [10.0.2.2]", got)
	}
}

func TestParseLinuxRouteIPv4UsesLittleEndianHex(t *testing.T) {
	t.Parallel()

	got, err := parseLinuxRouteIPv4("0102A8C0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !got.Equal(net.ParseIP("192.168.2.1")) {
		t.Fatalf("got %v want 192.168.2.1", got)
	}
}
