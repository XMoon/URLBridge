//go:build windows

package guest

import (
	"net"
	"sort"
	"syscall"
	"unsafe"

	"github.com/xmoon/urlbridge/internal/bridge"
)

const (
	gaaFlagIncludeGateways = 0x80
	ifOperStatusUp         = 1
)

type socketAddress struct {
	Sockaddr       *syscall.RawSockaddrAny
	SockaddrLength int32
}

type ipAdapterGatewayAddress struct {
	Length   uint32
	Reserved uint32
	Next     *ipAdapterGatewayAddress
	Address  socketAddress
}

type ipAdapterAddresses struct {
	Length                 uint32
	IfIndex                uint32
	Next                   *ipAdapterAddresses
	AdapterName            *byte
	FirstUnicastAddress    uintptr
	FirstAnycastAddress    uintptr
	FirstMulticastAddress  uintptr
	FirstDNSServerAddress  uintptr
	DNSSuffix              *uint16
	Description            *uint16
	FriendlyName           *uint16
	PhysicalAddress        [syscall.MAX_ADAPTER_ADDRESS_LENGTH]byte
	PhysicalAddressLength  uint32
	Flags                  uint32
	Mtu                    uint32
	IfType                 uint32
	OperStatus             uint32
	IPv6IfIndex            uint32
	ZoneIndices            [16]uint32
	FirstPrefix            uintptr
	TransmitLinkSpeed      uint64
	ReceiveLinkSpeed       uint64
	FirstWINSServerAddress uintptr
	FirstGatewayAddress    *ipAdapterGatewayAddress
	IPv4Metric             uint32
	IPv6Metric             uint32
	Luid                   uint64
}

var (
	iphlpapi                 = syscall.NewLazyDLL("iphlpapi.dll")
	procGetAdaptersAddresses = iphlpapi.NewProc("GetAdaptersAddresses")
)

func defaultGatewayURLs() []string {
	ips, err := defaultGatewayIPv4s()
	if err != nil {
		return nil
	}

	return gatewayBaseURLs(ips)
}

func defaultGatewayIPv4s() ([]net.IP, error) {
	size := uint32(15 * 1024)

	for attempts := 0; attempts < 4; attempts++ {
		buffer := make([]byte, size)
		first := (*ipAdapterAddresses)(unsafe.Pointer(&buffer[0]))
		if err := getAdaptersAddresses(first, &size); err != nil {
			if err == syscall.ERROR_BUFFER_OVERFLOW {
				continue
			}
			return nil, err
		}

		return collectAdapterGatewayIPv4s(first), nil
	}

	return nil, syscall.ERROR_BUFFER_OVERFLOW
}

func getAdaptersAddresses(first *ipAdapterAddresses, size *uint32) error {
	result, _, _ := procGetAdaptersAddresses.Call(
		uintptr(syscall.AF_INET),
		gaaFlagIncludeGateways,
		0,
		uintptr(unsafe.Pointer(first)),
		uintptr(unsafe.Pointer(size)),
	)
	if errno := syscall.Errno(result); errno != 0 {
		return errno
	}

	return nil
}

func collectAdapterGatewayIPv4s(first *ipAdapterAddresses) []net.IP {
	var ips []net.IP

	for adapter := first; adapter != nil; adapter = adapter.Next {
		if adapter.OperStatus != ifOperStatusUp {
			continue
		}
		for gateway := adapter.FirstGatewayAddress; gateway != nil; gateway = gateway.Next {
			if ip := gateway.Address.ipv4(); ip != nil {
				ips = append(ips, ip)
			}
		}
	}

	return ips
}

func gatewayBaseURLs(ips []net.IP) []string {
	normalized := normalizeIPv4GatewayStrings(ips)
	urls := make([]string, 0, len(normalized))
	for _, ip := range normalized {
		urls = append(urls, formatCandidateBaseURL(ip, bridge.DefaultPort))
	}
	return urls
}

func normalizeIPv4GatewayStrings(ips []net.IP) []string {
	seen := make(map[string]struct{}, len(ips))
	normalized := make([]string, 0, len(ips))

	for _, ip := range ips {
		ipv4 := ip.To4()
		if ipv4 == nil {
			continue
		}

		text := ipv4.String()
		if _, exists := seen[text]; exists {
			continue
		}
		seen[text] = struct{}{}
		normalized = append(normalized, text)
	}

	sort.Strings(normalized)
	return normalized
}

func (addr socketAddress) ipv4() net.IP {
	if addr.Sockaddr == nil {
		return nil
	}
	if uintptr(addr.SockaddrLength) < unsafe.Sizeof(syscall.RawSockaddrInet4{}) {
		return nil
	}
	if addr.Sockaddr.Addr.Family != syscall.AF_INET {
		return nil
	}

	raw := (*syscall.RawSockaddrInet4)(unsafe.Pointer(addr.Sockaddr))
	return net.IPv4(raw.Addr[0], raw.Addr[1], raw.Addr[2], raw.Addr[3]).To4()
}
