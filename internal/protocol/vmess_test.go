package protocol

import (
	"encoding/base64"
	"encoding/json"
	"testing"
)

// vmessTestPayload builds a VMess JSON payload and returns the base64-encoded
// string using the specified encoding.
func vmessTestPayload(t *testing.T, encode func([]byte) string, overrides map[string]interface{}) string {
	t.Helper()
	payload := map[string]interface{}{
		"v":    2,
		"ps":   "TestServer",
		"add":  "vmess.example.com",
		"port": 443,
		"id":   "b831381d-6324-4d53-ad4f-8cda48b30811",
		"aid":  0,
		"net":  "ws",
		"type": "none",
		"host": "vmess.example.com",
		"path": "/ws",
		"tls":  "tls",
		"sni":  "vmess.example.com",
		"alpn": "h2,http/1.1",
		"fp":   "chrome",
	}
	for k, v := range overrides {
		payload[k] = v
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal vmess payload: %v", err)
	}
	return encode(data)
}

func TestParseVMess(t *testing.T) {
	stdEncode := func(data []byte) string {
		return base64.StdEncoding.EncodeToString(data)
	}
	rawEncode := func(data []byte) string {
		return base64.RawStdEncoding.EncodeToString(data)
	}
	urlEncode := func(data []byte) string {
		return base64.RawURLEncoding.EncodeToString(data)
	}

	tests := []struct {
		name      string
		uri       string
		wantErr   bool
		errSubstr string
		check     func(t *testing.T, s *Server)
	}{
		{
			name: "standard VMess with padded base64",
			uri:  "vmess://" + vmessTestPayload(t, stdEncode, nil),
			check: func(t *testing.T, s *Server) {
				if s.Protocol != ProtocolVMess {
					t.Errorf("Protocol = %q, want %q", s.Protocol, ProtocolVMess)
				}
				if s.Name != "TestServer" {
					t.Errorf("Name = %q, want %q", s.Name, "TestServer")
				}
				if s.Address != "vmess.example.com" {
					t.Errorf("Address = %q, want %q", s.Address, "vmess.example.com")
				}
				if s.Port != 443 {
					t.Errorf("Port = %d, want %d", s.Port, 443)
				}
				if s.UUID != "b831381d-6324-4d53-ad4f-8cda48b30811" {
					t.Errorf("UUID = %q, want %q", s.UUID, "b831381d-6324-4d53-ad4f-8cda48b30811")
				}
				if s.Network != "ws" {
					t.Errorf("Network = %q, want %q", s.Network, "ws")
				}
				if s.TLS != "tls" {
					t.Errorf("TLS = %q, want %q", s.TLS, "tls")
				}
				if s.Security != "auto" {
					t.Errorf("Security = %q, want %q", s.Security, "auto")
				}
				if s.SNI != "vmess.example.com" {
					t.Errorf("SNI = %q, want %q", s.SNI, "vmess.example.com")
				}
				if s.ALPN != "h2,http/1.1" {
					t.Errorf("ALPN = %q, want %q", s.ALPN, "h2,http/1.1")
				}
				if s.Fingerprint != "chrome" {
					t.Errorf("Fingerprint = %q, want %q", s.Fingerprint, "chrome")
				}
				if s.RawURI == "" {
					t.Error("RawURI should not be empty")
				}
			},
		},
		{
			name: "VMess with unpadded base64",
			uri:  "vmess://" + vmessTestPayload(t, rawEncode, nil),
			check: func(t *testing.T, s *Server) {
				if s.Address != "vmess.example.com" {
					t.Errorf("Address = %q, want %q", s.Address, "vmess.example.com")
				}
			},
		},
		{
			name: "VMess with URL-safe base64",
			uri:  "vmess://" + vmessTestPayload(t, urlEncode, nil),
			check: func(t *testing.T, s *Server) {
				if s.Address != "vmess.example.com" {
					t.Errorf("Address = %q, want %q", s.Address, "vmess.example.com")
				}
			},
		},
		{
			name: "VMess with port as string",
			uri:  "vmess://" + vmessTestPayload(t, stdEncode, map[string]interface{}{"port": "443"}),
			check: func(t *testing.T, s *Server) {
				if s.Port != 443 {
					t.Errorf("Port = %d, want %d", s.Port, 443)
				}
			},
		},
		{
			name: "VMess with alterId as string",
			uri:  "vmess://" + vmessTestPayload(t, stdEncode, map[string]interface{}{"aid": "64"}),
			check: func(t *testing.T, s *Server) {
				if s.AlterID != 64 {
					t.Errorf("AlterID = %d, want %d", s.AlterID, 64)
				}
			},
		},
		{
			name:      "VMess with missing address",
			uri:       "vmess://" + vmessTestPayload(t, stdEncode, map[string]interface{}{"add": ""}),
			wantErr:   true,
			errSubstr: "address",
		},
		{
			name:      "VMess with invalid base64",
			uri:       "vmess://not-valid-base64!!!",
			wantErr:   true,
			errSubstr: "base64",
		},
		{
			name:      "VMess with invalid JSON",
			uri:       "vmess://" + base64.StdEncoding.EncodeToString([]byte("not json")),
			wantErr:   true,
			errSubstr: "JSON",
		},
		{
			name:      "VMess with zero port",
			uri:       "vmess://" + vmessTestPayload(t, stdEncode, map[string]interface{}{"port": 0}),
			wantErr:   true,
			errSubstr: "port",
		},
		{
			name: "VMess with no ps field defaults to add:port",
			uri:  "vmess://" + vmessTestPayload(t, stdEncode, map[string]interface{}{"ps": ""}),
			check: func(t *testing.T, s *Server) {
				if s.Name != "vmess.example.com:443" {
					t.Errorf("Name = %q, want %q", s.Name, "vmess.example.com:443")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := ParseVMess(tt.uri)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errSubstr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.check != nil {
				tt.check(t, s)
			}
		})
	}
}
