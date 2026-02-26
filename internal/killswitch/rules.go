package killswitch

import (
	"fmt"
	"strings"
)

// GenerateRules produces pf anchor rules for the kill switch.
// The rules block all traffic except:
//   - Loopback (for local SOCKS5/HTTP proxy)
//   - Traffic to the VPN server IP on the specified port
//   - DHCP (to maintain network connectivity)
//   - DNS (safety net for initial resolution)
//   - Split tunnel bypass IPs (direct traffic for listed destinations)
//
// IPv6 is explicitly blocked to prevent leak (research pitfall 3).
func GenerateRules(serverIP string, serverPort int, bypassIPs []string) string {
	var builder strings.Builder

	builder.WriteString(`# Azad Kill Switch - generated rules
# Anchor: com.azad.killswitch

# Block policy: drop silently (no RST/ICMP unreachable)
set block-policy drop

# Allow all loopback traffic (required for local SOCKS5/HTTP proxy)
pass quick on lo0 all

`)
	builder.WriteString(fmt.Sprintf("# Allow traffic to VPN server\npass out quick proto {tcp, udp} from any to %s port %d\n\n", serverIP, serverPort))

	// Pass direct split tunnel traffic through the firewall.
	// pf handles CIDR notation natively (e.g., pass out quick from any to 10.0.0.0/8).
	for _, ip := range bypassIPs {
		builder.WriteString(fmt.Sprintf("pass out quick from any to %s\n", ip))
	}
	if len(bypassIPs) > 0 {
		builder.WriteString("\n")
	}

	builder.WriteString(`# Allow DHCP
pass quick proto {tcp, udp} from any port 67:68 to any port 67:68

# Allow DNS (safety net for initial resolution)
pass out quick proto {tcp, udp} from any to any port 53

# Block everything else (IPv4)
block out all
block in all

# Block IPv6 to prevent leak
block out inet6 all
block in inet6 all
`)

	return builder.String()
}
