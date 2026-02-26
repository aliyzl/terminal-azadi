package splittunnel

import (
	"testing"
)

func TestToXrayRules(t *testing.T) {
	tests := []struct {
		name      string
		rules     []Rule
		mode      Mode
		wantCount int
		check     func(t *testing.T, result []XrayRoutingRule)
	}{
		{
			name: "exclusive mode domain rule",
			rules: []Rule{
				{Value: "example.com", Type: RuleTypeDomain},
			},
			mode:      ModeExclusive,
			wantCount: 1,
			check: func(t *testing.T, result []XrayRoutingRule) {
				r := result[0]
				if r.OutboundTag != "direct" {
					t.Errorf("outboundTag = %q, want direct", r.OutboundTag)
				}
				if r.Type != "field" {
					t.Errorf("type = %q, want field", r.Type)
				}
				// Domain rule should produce full: and domain: prefixes
				if len(r.Domain) != 2 {
					t.Fatalf("domain count = %d, want 2", len(r.Domain))
				}
				foundFull := false
				foundDomain := false
				for _, d := range r.Domain {
					if d == "full:example.com" {
						foundFull = true
					}
					if d == "domain:example.com" {
						foundDomain = true
					}
				}
				if !foundFull {
					t.Error("expected full:example.com in domain list")
				}
				if !foundDomain {
					t.Error("expected domain:example.com in domain list")
				}
			},
		},
		{
			name: "exclusive mode IP rule",
			rules: []Rule{
				{Value: "1.2.3.4", Type: RuleTypeIP},
			},
			mode:      ModeExclusive,
			wantCount: 1,
			check: func(t *testing.T, result []XrayRoutingRule) {
				r := result[0]
				if r.OutboundTag != "direct" {
					t.Errorf("outboundTag = %q, want direct", r.OutboundTag)
				}
				if len(r.IP) != 1 || r.IP[0] != "1.2.3.4" {
					t.Errorf("ip = %v, want [1.2.3.4]", r.IP)
				}
			},
		},
		{
			name: "exclusive mode wildcard",
			rules: []Rule{
				{Value: "*.google.com", Type: RuleTypeWildcard},
			},
			mode:      ModeExclusive,
			wantCount: 1,
			check: func(t *testing.T, result []XrayRoutingRule) {
				r := result[0]
				if r.OutboundTag != "direct" {
					t.Errorf("outboundTag = %q, want direct", r.OutboundTag)
				}
				if len(r.Domain) != 1 || r.Domain[0] != "domain:google.com" {
					t.Errorf("domain = %v, want [domain:google.com]", r.Domain)
				}
			},
		},
		{
			name: "inclusive mode domain rule",
			rules: []Rule{
				{Value: "example.com", Type: RuleTypeDomain},
			},
			mode:      ModeInclusive,
			wantCount: 1,
			check: func(t *testing.T, result []XrayRoutingRule) {
				r := result[0]
				if r.OutboundTag != "proxy" {
					t.Errorf("outboundTag = %q, want proxy", r.OutboundTag)
				}
			},
		},
		{
			name: "mixed IP and domain rules produce separate entries",
			rules: []Rule{
				{Value: "example.com", Type: RuleTypeDomain},
				{Value: "1.2.3.4", Type: RuleTypeIP},
				{Value: "10.0.0.0/8", Type: RuleTypeCIDR},
			},
			mode:      ModeExclusive,
			wantCount: 2,
			check: func(t *testing.T, result []XrayRoutingRule) {
				// First rule should be domains
				domainRule := result[0]
				if len(domainRule.Domain) == 0 {
					t.Error("first rule should have domains")
				}
				if len(domainRule.IP) != 0 {
					t.Error("first rule should not have IPs")
				}
				// Second rule should be IPs
				ipRule := result[1]
				if len(ipRule.IP) == 0 {
					t.Error("second rule should have IPs")
				}
				if len(ipRule.Domain) != 0 {
					t.Error("second rule should not have domains")
				}
				// Should contain both IP and CIDR
				if len(ipRule.IP) != 2 {
					t.Errorf("ip count = %d, want 2", len(ipRule.IP))
				}
			},
		},
		{
			name:      "empty rules",
			rules:     nil,
			mode:      ModeExclusive,
			wantCount: 0,
			check: func(t *testing.T, result []XrayRoutingRule) {
				if len(result) != 0 {
					t.Errorf("expected empty result, got %d rules", len(result))
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ToXrayRules(tc.rules, tc.mode)
			if len(result) != tc.wantCount {
				t.Fatalf("rule count = %d, want %d", len(result), tc.wantCount)
			}
			if tc.check != nil {
				tc.check(t, result)
			}
		})
	}
}
