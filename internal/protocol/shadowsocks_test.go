package protocol

import (
	"encoding/base64"
	"testing"
)

func TestParseShadowsocks(t *testing.T) {
	tests := []struct {
		name      string
		uri       string
		wantErr   bool
		errSubstr string
		check     func(t *testing.T, s *Server)
	}{
		{
			name: "SS with base64-encoded method:password (SIP002 legacy/AEAD)",
			uri:  "ss://" + base64.StdEncoding.EncodeToString([]byte("aes-256-gcm:mypassword")) + "@ss.example.com:8388#SSServer",
			check: func(t *testing.T, s *Server) {
				if s.Protocol != ProtocolShadowsocks {
					t.Errorf("Protocol = %q, want %q", s.Protocol, ProtocolShadowsocks)
				}
				if s.Method != "aes-256-gcm" {
					t.Errorf("Method = %q, want %q", s.Method, "aes-256-gcm")
				}
				if s.Password != "mypassword" {
					t.Errorf("Password = %q, want %q", s.Password, "mypassword")
				}
				if s.Address != "ss.example.com" {
					t.Errorf("Address = %q, want %q", s.Address, "ss.example.com")
				}
				if s.Port != 8388 {
					t.Errorf("Port = %d, want %d", s.Port, 8388)
				}
				if s.Name != "SSServer" {
					t.Errorf("Name = %q, want %q", s.Name, "SSServer")
				}
				if s.RawURI == "" {
					t.Error("RawURI should not be empty")
				}
			},
		},
		{
			name: "SS with plaintext method:password (AEAD-2022)",
			uri:  "ss://2022-blake3-aes-128-gcm:YWJjZGVmZzEyMzQ1Njc4@aead.example.com:8388#AEAD2022",
			check: func(t *testing.T, s *Server) {
				if s.Method != "2022-blake3-aes-128-gcm" {
					t.Errorf("Method = %q, want %q", s.Method, "2022-blake3-aes-128-gcm")
				}
				if s.Password != "YWJjZGVmZzEyMzQ1Njc4" {
					t.Errorf("Password = %q, want %q", s.Password, "YWJjZGVmZzEyMzQ1Njc4")
				}
				if s.Address != "aead.example.com" {
					t.Errorf("Address = %q, want %q", s.Address, "aead.example.com")
				}
			},
		},
		{
			name: "SS with URL-encoded password containing special chars",
			uri:  "ss://aes-256-gcm:p%40ss%3Aword%21@encoded.example.com:8388#EncodedPass",
			check: func(t *testing.T, s *Server) {
				if s.Method != "aes-256-gcm" {
					t.Errorf("Method = %q, want %q", s.Method, "aes-256-gcm")
				}
				if s.Password != "p@ss:word!" {
					t.Errorf("Password = %q, want %q", s.Password, "p@ss:word!")
				}
			},
		},
		{
			name: "SS with no fragment defaults to host:port",
			uri:  "ss://" + base64.StdEncoding.EncodeToString([]byte("aes-256-gcm:mypassword")) + "@nofrag.example.com:8388",
			check: func(t *testing.T, s *Server) {
				if s.Name != "nofrag.example.com:8388" {
					t.Errorf("Name = %q, want %q", s.Name, "nofrag.example.com:8388")
				}
			},
		},
		{
			name:      "SS with missing host or port",
			uri:       "ss://" + base64.StdEncoding.EncodeToString([]byte("aes-256-gcm:mypassword")) + "@:8388",
			wantErr:   true,
			errSubstr: "host",
		},
		{
			name:      "SS with un-decodable base64 userinfo and no colon",
			uri:       "ss://notbase64nocolon@bad.example.com:8388#BadSS",
			wantErr:   true,
			errSubstr: "method:password",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := ParseShadowsocks(tt.uri)
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
