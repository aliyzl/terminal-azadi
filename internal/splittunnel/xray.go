package splittunnel

import (
	"strings"
)

// XrayRoutingRule represents a routing rule in Xray format.
// Defined here to avoid circular dependency with engine package.
type XrayRoutingRule struct {
	Type        string   `json:"type"`
	OutboundTag string   `json:"outboundTag"`
	IP          []string `json:"ip,omitempty"`
	Domain      []string `json:"domain,omitempty"`
}

// ToXrayRules converts split tunnel rules to Xray routing rules.
// The outboundTag depends on the mode:
//   - Exclusive mode: rules -> "direct" (bypass VPN)
//   - Inclusive mode: rules -> "proxy" (use VPN)
func ToXrayRules(rules []Rule, mode Mode) []XrayRoutingRule {
	var domains []string
	var ips []string

	for _, r := range rules {
		switch r.Type {
		case RuleTypeIP, RuleTypeCIDR:
			ips = append(ips, r.Value)
		case RuleTypeDomain:
			// full: prefix for exact domain match
			domains = append(domains, "full:"+r.Value)
			// Also match subdomains
			domains = append(domains, "domain:"+r.Value)
		case RuleTypeWildcard:
			// *.example.com -> domain:example.com in Xray
			base := strings.TrimPrefix(r.Value, "*.")
			domains = append(domains, "domain:"+base)
		}
	}

	tag := "direct"
	if mode == ModeInclusive {
		tag = "proxy"
	}

	var xrayRules []XrayRoutingRule
	if len(domains) > 0 {
		xrayRules = append(xrayRules, XrayRoutingRule{
			Type:        "field",
			Domain:      domains,
			OutboundTag: tag,
		})
	}
	if len(ips) > 0 {
		xrayRules = append(xrayRules, XrayRoutingRule{
			Type:        "field",
			IP:          ips,
			OutboundTag: tag,
		})
	}

	return xrayRules
}
