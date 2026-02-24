package protocol

import (
	"testing"
)

func TestParseTrojan(t *testing.T) {
	tests := []struct {
		name      string
		uri       string
		wantErr   bool
		errSubstr string
		check     func(t *testing.T, s *Server)
	}{
		{
			name: "standard Trojan with TLS and SNI",
			uri:  "trojan://mypassword123@trojan.example.com:443?security=tls&sni=trojan.example.com&fp=chrome&alpn=h2,http/1.1#TrojanServer",
			check: func(t *testing.T, s *Server) {
				if s.Protocol != ProtocolTrojan {
					t.Errorf("Protocol = %q, want %q", s.Protocol, ProtocolTrojan)
				}
				if s.Password != "mypassword123" {
					t.Errorf("Password = %q, want %q", s.Password, "mypassword123")
				}
				if s.Address != "trojan.example.com" {
					t.Errorf("Address = %q, want %q", s.Address, "trojan.example.com")
				}
				if s.Port != 443 {
					t.Errorf("Port = %d, want %d", s.Port, 443)
				}
				if s.Name != "TrojanServer" {
					t.Errorf("Name = %q, want %q", s.Name, "TrojanServer")
				}
				if s.TLS != "tls" {
					t.Errorf("TLS = %q, want %q", s.TLS, "tls")
				}
				if s.SNI != "trojan.example.com" {
					t.Errorf("SNI = %q, want %q", s.SNI, "trojan.example.com")
				}
				if s.Fingerprint != "chrome" {
					t.Errorf("Fingerprint = %q, want %q", s.Fingerprint, "chrome")
				}
				if s.ALPN != "h2,http/1.1" {
					t.Errorf("ALPN = %q, want %q", s.ALPN, "h2,http/1.1")
				}
				if s.RawURI == "" {
					t.Error("RawURI should not be empty")
				}
			},
		},
		{
			name: "Trojan with WebSocket transport",
			uri:  "trojan://mypassword123@ws.example.com:443?type=ws&security=tls&path=%2Fws&host=ws.example.com&sni=ws.example.com#TrojanWS",
			check: func(t *testing.T, s *Server) {
				if s.Network != "ws" {
					t.Errorf("Network = %q, want %q", s.Network, "ws")
				}
				if s.Path != "/ws" {
					t.Errorf("Path = %q, want %q", s.Path, "/ws")
				}
				if s.Host != "ws.example.com" {
					t.Errorf("Host = %q, want %q", s.Host, "ws.example.com")
				}
			},
		},
		{
			name: "Trojan with gRPC transport",
			uri:  "trojan://mypassword123@grpc.example.com:443?type=grpc&security=tls&serviceName=trojangrpc&sni=grpc.example.com#TrojanGRPC",
			check: func(t *testing.T, s *Server) {
				if s.Network != "grpc" {
					t.Errorf("Network = %q, want %q", s.Network, "grpc")
				}
				if s.ServiceName != "trojangrpc" {
					t.Errorf("ServiceName = %q, want %q", s.ServiceName, "trojangrpc")
				}
			},
		},
		{
			name: "Trojan with no port defaults to 443",
			uri:  "trojan://mypassword123@defaultport.example.com?security=tls#DefaultPort",
			check: func(t *testing.T, s *Server) {
				if s.Port != 443 {
					t.Errorf("Port = %d, want %d", s.Port, 443)
				}
			},
		},
		{
			name: "Trojan with REALITY params",
			uri:  "trojan://mypassword123@reality.example.com:443?type=tcp&security=reality&pbk=realpubkey&sid=abcd1234&spx=%2F&sni=www.google.com&fp=chrome#TrojanReality",
			check: func(t *testing.T, s *Server) {
				if s.TLS != "reality" {
					t.Errorf("TLS = %q, want %q", s.TLS, "reality")
				}
				if s.PublicKey != "realpubkey" {
					t.Errorf("PublicKey = %q, want %q", s.PublicKey, "realpubkey")
				}
				if s.ShortID != "abcd1234" {
					t.Errorf("ShortID = %q, want %q", s.ShortID, "abcd1234")
				}
				if s.SpiderX != "/" {
					t.Errorf("SpiderX = %q, want %q", s.SpiderX, "/")
				}
			},
		},
		{
			name: "Trojan with no fragment defaults to host:port",
			uri:  "trojan://mypassword123@nofrag.example.com:8443?security=tls",
			check: func(t *testing.T, s *Server) {
				if s.Name != "nofrag.example.com:8443" {
					t.Errorf("Name = %q, want %q", s.Name, "nofrag.example.com:8443")
				}
			},
		},
		{
			name:      "Trojan with empty password",
			uri:       "trojan://@empty.example.com:443",
			wantErr:   true,
			errSubstr: "password",
		},
		{
			name:      "Trojan with missing host",
			uri:       "trojan://mypassword123@:443",
			wantErr:   true,
			errSubstr: "host",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := ParseTrojan(tt.uri)
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
