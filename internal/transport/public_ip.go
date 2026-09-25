package transport

import "net/netip"

// Conservative snapshot of special-purpose ranges relevant to outbound
// scanning, based on the IANA IPv4/IPv6 registries reviewed 2026-09-25.
// Explicit pins do not override these denials. Registry updates require a
// source review and tests; this list is not a claim of universal routability.
var specialIPv4 = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"),
	netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"),
	netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.31.196.0/24"),
	netip.MustParsePrefix("192.52.193.0/24"),
	netip.MustParsePrefix("192.88.99.0/24"),
	netip.MustParsePrefix("192.175.48.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"),
	netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"),
	netip.MustParsePrefix("240.0.0.0/4"),
}

var specialIPv6 = []netip.Prefix{
	netip.MustParsePrefix("2001::/23"),
	netip.MustParsePrefix("2001:db8::/32"),
	netip.MustParsePrefix("2002::/16"),
	netip.MustParsePrefix("2620:4f:8000::/48"),
	netip.MustParsePrefix("3fff::/20"),
}

var ipv6GlobalUnicast = netip.MustParsePrefix("2000::/3")

func publicDestination(ip netip.Addr) bool {
	if !ip.IsValid() || ip.Is4In6() || ip.Zone() != "" || !ip.IsGlobalUnicast() ||
		ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsMulticast() {
		return false
	}
	if ip.Is4() {
		for _, block := range specialIPv4 {
			if block.Contains(ip) {
				return false
			}
		}
		return true
	}
	if !ipv6GlobalUnicast.Contains(ip) {
		return false
	}
	for _, block := range specialIPv6 {
		if block.Contains(ip) {
			return false
		}
	}
	return true
}
