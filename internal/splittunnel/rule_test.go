package splittunnel

import (
	"testing"
)

func TestParseRule(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantType RuleType
		wantVal  string
		wantErr  bool
	}{
		{
			name:     "valid single IPv4",
			input:    "1.2.3.4",
			wantType: RuleTypeIP,
			wantVal:  "1.2.3.4",
		},
		{
			name:     "valid CIDR /8",
			input:    "10.0.0.0/8",
			wantType: RuleTypeCIDR,
			wantVal:  "10.0.0.0/8",
		},
		{
			name:     "valid CIDR /24",
			input:    "192.168.1.0/24",
			wantType: RuleTypeCIDR,
			wantVal:  "192.168.1.0/24",
		},
		{
			name:     "valid domain",
			input:    "example.com",
			wantType: RuleTypeDomain,
			wantVal:  "example.com",
		},
		{
			name:     "valid subdomain",
			input:    "sub.example.com",
			wantType: RuleTypeDomain,
			wantVal:  "sub.example.com",
		},
		{
			name:     "valid wildcard",
			input:    "*.google.com",
			wantType: RuleTypeWildcard,
			wantVal:  "*.google.com",
		},
		{
			name:    "invalid empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid garbage",
			input:   "not a rule at all",
			wantErr: true,
		},
		{
			name:    "invalid wildcard double dot",
			input:   "*..",
			wantErr: true,
		},
		{
			name:     "whitespace trimming",
			input:    "  1.2.3.4  ",
			wantType: RuleTypeIP,
			wantVal:  "1.2.3.4",
		},
		{
			name:     "IPv6 single IP",
			input:    "::1",
			wantType: RuleTypeIP,
			wantVal:  "::1",
		},
		{
			name:     "IPv6 CIDR",
			input:    "fd00::/8",
			wantType: RuleTypeCIDR,
			wantVal:  "fd00::/8",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			rule, err := ParseRule(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if rule.Type != tc.wantType {
				t.Errorf("type = %q, want %q", rule.Type, tc.wantType)
			}
			if rule.Value != tc.wantVal {
				t.Errorf("value = %q, want %q", rule.Value, tc.wantVal)
			}
		})
	}
}

func TestHasDomainRules(t *testing.T) {
	tests := []struct {
		name  string
		rules []Rule
		want  bool
	}{
		{
			name: "contains domain rule",
			rules: []Rule{
				{Value: "1.2.3.4", Type: RuleTypeIP},
				{Value: "example.com", Type: RuleTypeDomain},
			},
			want: true,
		},
		{
			name: "contains wildcard rule",
			rules: []Rule{
				{Value: "10.0.0.0/8", Type: RuleTypeCIDR},
				{Value: "*.google.com", Type: RuleTypeWildcard},
			},
			want: true,
		},
		{
			name: "IP and CIDR only",
			rules: []Rule{
				{Value: "1.2.3.4", Type: RuleTypeIP},
				{Value: "10.0.0.0/8", Type: RuleTypeCIDR},
			},
			want: false,
		},
		{
			name:  "empty rules",
			rules: nil,
			want:  false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := HasDomainRules(tc.rules)
			if got != tc.want {
				t.Errorf("HasDomainRules() = %v, want %v", got, tc.want)
			}
		})
	}
}
