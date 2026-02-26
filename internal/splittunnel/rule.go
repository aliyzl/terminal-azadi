package splittunnel

import (
	"fmt"
	"net"
	"strings"
)

// RuleType represents the classification of a split tunnel rule.
type RuleType string

const (
	RuleTypeIP       RuleType = "ip"
	RuleTypeCIDR     RuleType = "cidr"
	RuleTypeDomain   RuleType = "domain"
	RuleTypeWildcard RuleType = "wildcard"
)

// Rule represents a single split tunnel entry.
type Rule struct {
	Value string   `koanf:"value" yaml:"value"`
	Type  RuleType `koanf:"type"  yaml:"type"`
}

// Mode determines how split tunnel rules are applied.
type Mode string

const (
	ModeExclusive Mode = "exclusive" // Listed rules bypass VPN (go direct)
	ModeInclusive Mode = "inclusive" // Listed rules use VPN, rest goes direct
)

// Config holds runtime split tunnel configuration.
type Config struct {
	Enabled bool   `koanf:"enabled"`
	Mode    Mode   `koanf:"mode"`
	Rules   []Rule `koanf:"rules"`
}

// ParseRule classifies and validates a user input string into a Rule.
func ParseRule(input string) (Rule, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return Rule{}, fmt.Errorf("empty rule input")
	}

	// Try single IP (IPv4 or IPv6).
	if ip := net.ParseIP(input); ip != nil {
		return Rule{Value: input, Type: RuleTypeIP}, nil
	}

	// Try CIDR range.
	if _, _, err := net.ParseCIDR(input); err == nil {
		return Rule{Value: input, Type: RuleTypeCIDR}, nil
	}

	// Try wildcard domain (*.example.com).
	if strings.HasPrefix(input, "*.") {
		domain := input[2:] // strip "*."
		if isValidDomain(domain) {
			return Rule{Value: input, Type: RuleTypeWildcard}, nil
		}
		return Rule{}, fmt.Errorf("invalid wildcard domain: %s", input)
	}

	// Try plain domain.
	if isValidDomain(input) {
		return Rule{Value: input, Type: RuleTypeDomain}, nil
	}

	return Rule{}, fmt.Errorf("invalid rule: %s (expected IP, CIDR, domain, or *.domain)", input)
}

// HasDomainRules returns true if any rule is a domain or wildcard type.
func HasDomainRules(rules []Rule) bool {
	for _, r := range rules {
		if r.Type == RuleTypeDomain || r.Type == RuleTypeWildcard {
			return true
		}
	}
	return false
}

// isValidDomain checks if the string is a valid domain name.
// Requires at least one dot, no leading/trailing dots, only alphanumeric + hyphen + dots,
// and each label must be non-empty.
func isValidDomain(s string) bool {
	if len(s) == 0 {
		return false
	}
	if !strings.Contains(s, ".") {
		return false
	}
	if strings.HasPrefix(s, ".") || strings.HasSuffix(s, ".") {
		return false
	}

	labels := strings.Split(s, ".")
	for _, label := range labels {
		if len(label) == 0 {
			return false
		}
		for _, c := range label {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-') {
				return false
			}
		}
	}
	return true
}
