package protocol

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

func TestParseURI(t *testing.T) {
	// Build a valid VMess payload for testing
	vmessPayload, _ := json.Marshal(map[string]interface{}{
		"v": 2, "ps": "Test", "add": "vmess.example.com",
		"port": 443, "id": "b831381d-6324-4d53-ad4f-8cda48b30811",
		"aid": 0, "net": "tcp", "type": "none", "host": "", "path": "", "tls": "",
	})
	vmessURI := "vmess://" + base64.StdEncoding.EncodeToString(vmessPayload)

	ssUserinfo := base64.StdEncoding.EncodeToString([]byte("aes-256-gcm:pass"))

	tests := []struct {
		name         string
		uri          string
		wantErr      bool
		errSubstr    string
		wantProtocol Protocol
	}{
		{
			name:         "vless:// dispatches to ParseVLESS",
			uri:          "vless://b831381d-6324-4d53-ad4f-8cda48b30811@example.com:443?type=tcp#Test",
			wantProtocol: ProtocolVLESS,
		},
		{
			name:         "vmess:// dispatches to ParseVMess",
			uri:          vmessURI,
			wantProtocol: ProtocolVMess,
		},
		{
			name:         "trojan:// dispatches to ParseTrojan",
			uri:          "trojan://password@example.com:443?security=tls#Test",
			wantProtocol: ProtocolTrojan,
		},
		{
			name:         "ss:// dispatches to ParseShadowsocks",
			uri:          "ss://" + ssUserinfo + "@example.com:8388#Test",
			wantProtocol: ProtocolShadowsocks,
		},
		{
			name:      "empty URI returns error",
			uri:       "",
			wantErr:   true,
			errSubstr: "empty",
		},
		{
			name:      "unsupported scheme returns error",
			uri:       "http://example.com",
			wantErr:   true,
			errSubstr: "unsupported",
		},
		{
			name:         "whitespace-trimmed URI still parses",
			uri:          "  vless://b831381d-6324-4d53-ad4f-8cda48b30811@example.com:443?type=tcp#Test  ",
			wantProtocol: ProtocolVLESS,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := ParseURI(tt.uri)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errSubstr)
				}
				if !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if s.Protocol != tt.wantProtocol {
				t.Errorf("Protocol = %q, want %q", s.Protocol, tt.wantProtocol)
			}
		})
	}
}
