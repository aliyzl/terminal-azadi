package protocol

import (
	"testing"
)

func TestParseVLESS(t *testing.T) {
	tests := []struct {
		name      string
		uri       string
		wantErr   bool
		errSubstr string
		check     func(t *testing.T, s *Server)
	}{
		{
			name: "standard VLESS with all common params",
			uri:  "vless://b831381d-6324-4d53-ad4f-8cda48b30811@example.com:443?type=ws&security=tls&sni=example.com&fp=chrome&path=%2Fws&host=example.com&encryption=none#MyServer",
			check: func(t *testing.T, s *Server) {
				if s.Protocol != ProtocolVLESS {
					t.Errorf("Protocol = %q, want %q", s.Protocol, ProtocolVLESS)
				}
				if s.UUID != "b831381d-6324-4d53-ad4f-8cda48b30811" {
					t.Errorf("UUID = %q, want %q", s.UUID, "b831381d-6324-4d53-ad4f-8cda48b30811")
				}
				if s.Address != "example.com" {
					t.Errorf("Address = %q, want %q", s.Address, "example.com")
				}
				if s.Port != 443 {
					t.Errorf("Port = %d, want %d", s.Port, 443)
				}
				if s.Name != "MyServer" {
					t.Errorf("Name = %q, want %q", s.Name, "MyServer")
				}
				if s.Network != "ws" {
					t.Errorf("Network = %q, want %q", s.Network, "ws")
				}
				if s.TLS != "tls" {
					t.Errorf("TLS = %q, want %q", s.TLS, "tls")
				}
				if s.SNI != "example.com" {
					t.Errorf("SNI = %q, want %q", s.SNI, "example.com")
				}
				if s.Fingerprint != "chrome" {
					t.Errorf("Fingerprint = %q, want %q", s.Fingerprint, "chrome")
				}
				if s.Path != "/ws" {
					t.Errorf("Path = %q, want %q", s.Path, "/ws")
				}
				if s.Host != "example.com" {
					t.Errorf("Host = %q, want %q", s.Host, "example.com")
				}
				if s.Encryption != "none" {
					t.Errorf("Encryption = %q, want %q", s.Encryption, "none")
				}
				if s.RawURI == "" {
					t.Error("RawURI should not be empty")
				}
			},
		},
		{
			name: "VLESS with REALITY params",
			uri:  "vless://b831381d-6324-4d53-ad4f-8cda48b30811@reality.example.com:443?type=tcp&security=reality&pbk=abc123publickey&sid=deadbeef&spx=%2F&sni=www.google.com&fp=chrome&flow=xtls-rprx-vision#RealityServer",
			check: func(t *testing.T, s *Server) {
				if s.TLS != "reality" {
					t.Errorf("TLS = %q, want %q", s.TLS, "reality")
				}
				if s.PublicKey != "abc123publickey" {
					t.Errorf("PublicKey = %q, want %q", s.PublicKey, "abc123publickey")
				}
				if s.ShortID != "deadbeef" {
					t.Errorf("ShortID = %q, want %q", s.ShortID, "deadbeef")
				}
				if s.SpiderX != "/" {
					t.Errorf("SpiderX = %q, want %q", s.SpiderX, "/")
				}
				if s.Flow != "xtls-rprx-vision" {
					t.Errorf("Flow = %q, want %q", s.Flow, "xtls-rprx-vision")
				}
				if s.Name != "RealityServer" {
					t.Errorf("Name = %q, want %q", s.Name, "RealityServer")
				}
			},
		},
		{
			name: "VLESS with flow xtls-rprx-vision",
			uri:  "vless://b831381d-6324-4d53-ad4f-8cda48b30811@flow.example.com:443?type=tcp&security=tls&flow=xtls-rprx-vision#FlowServer",
			check: func(t *testing.T, s *Server) {
				if s.Flow != "xtls-rprx-vision" {
					t.Errorf("Flow = %q, want %q", s.Flow, "xtls-rprx-vision")
				}
			},
		},
		{
			name: "VLESS with gRPC transport",
			uri:  "vless://b831381d-6324-4d53-ad4f-8cda48b30811@grpc.example.com:443?type=grpc&security=tls&serviceName=mygrpc&sni=grpc.example.com#GrpcServer",
			check: func(t *testing.T, s *Server) {
				if s.Network != "grpc" {
					t.Errorf("Network = %q, want %q", s.Network, "grpc")
				}
				if s.ServiceName != "mygrpc" {
					t.Errorf("ServiceName = %q, want %q", s.ServiceName, "mygrpc")
				}
			},
		},
		{
			name: "VLESS with IPv6 address",
			uri:  "vless://b831381d-6324-4d53-ad4f-8cda48b30811@[2001:db8::1]:443?type=tcp&security=tls#IPv6Server",
			check: func(t *testing.T, s *Server) {
				if s.Address != "2001:db8::1" {
					t.Errorf("Address = %q, want %q", s.Address, "2001:db8::1")
				}
				if s.Port != 443 {
					t.Errorf("Port = %d, want %d", s.Port, 443)
				}
				if s.Name != "IPv6Server" {
					t.Errorf("Name = %q, want %q", s.Name, "IPv6Server")
				}
			},
		},
		{
			name: "VLESS with no fragment defaults to host:port",
			uri:  "vless://b831381d-6324-4d53-ad4f-8cda48b30811@nofrag.example.com:8443?type=tcp&security=none",
			check: func(t *testing.T, s *Server) {
				if s.Name != "nofrag.example.com:8443" {
					t.Errorf("Name = %q, want %q", s.Name, "nofrag.example.com:8443")
				}
				if s.TLS != "none" {
					t.Errorf("TLS = %q, want %q", s.TLS, "none")
				}
			},
		},
		{
			name:      "empty URI returns error",
			uri:       "",
			wantErr:   true,
			errSubstr: "missing",
		},
		{
			name:      "missing UUID returns error",
			uri:       "vless://@example.com:443?type=tcp",
			wantErr:   true,
			errSubstr: "UUID",
		},
		{
			name:      "missing host returns error",
			uri:       "vless://b831381d-6324-4d53-ad4f-8cda48b30811@:443?type=tcp",
			wantErr:   true,
			errSubstr: "host",
		},
		{
			name:      "invalid port returns error",
			uri:       "vless://b831381d-6324-4d53-ad4f-8cda48b30811@example.com:notaport?type=tcp",
			wantErr:   true,
			errSubstr: "port",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := ParseVLESS(tt.uri)
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
