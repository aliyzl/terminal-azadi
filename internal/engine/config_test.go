package engine

import (
	"encoding/json"
	"testing"

	"github.com/leejooy96/azad/internal/protocol"

	// Register all xray-core protocol handlers and JSON config loader.
	_ "github.com/xtls/xray-core/main/distro/all"
)

// testCase defines a table-driven test for BuildConfig.
type testCase struct {
	name     string
	server   protocol.Server
	socks    int
	http     int
	wantProto string // expected outbound protocol
	wantErr  bool
	check    func(t *testing.T, cfg *XrayConfig)
}

func TestBuildConfig(t *testing.T) {
	tests := []testCase{
		{
			name: "VLESS + REALITY + tcp",
			server: protocol.Server{
				Protocol:    protocol.ProtocolVLESS,
				Address:     "example.com",
				Port:        443,
				UUID:        "test-uuid-1234",
				Encryption:  "none",
				Flow:        "xtls-rprx-vision",
				Network:     "tcp",
				TLS:         "reality",
				SNI:         "www.microsoft.com",
				Fingerprint: "chrome",
				PublicKey:   "test-public-key",
				ShortID:     "abcd1234",
				SpiderX:    "/",
			},
			socks:     1080,
			http:      8080,
			wantProto: "vless",
			check: func(t *testing.T, cfg *XrayConfig) {
				out := cfg.Outbounds[0]
				// Check protocol-specific settings
				var settings struct {
					Vnext []struct {
						Address string `json:"address"`
						Port    int    `json:"port"`
						Users   []struct {
							ID         string `json:"id"`
							Encryption string `json:"encryption"`
							Flow       string `json:"flow"`
						} `json:"users"`
					} `json:"vnext"`
				}
				if err := json.Unmarshal(out.Settings, &settings); err != nil {
					t.Fatalf("unmarshal vless settings: %v", err)
				}
				if len(settings.Vnext) != 1 {
					t.Fatal("expected 1 vnext entry")
				}
				vn := settings.Vnext[0]
				if vn.Address != "example.com" {
					t.Errorf("address = %q, want example.com", vn.Address)
				}
				if vn.Port != 443 {
					t.Errorf("port = %d, want 443", vn.Port)
				}
				if len(vn.Users) != 1 {
					t.Fatal("expected 1 user")
				}
				if vn.Users[0].ID != "test-uuid-1234" {
					t.Errorf("user id = %q, want test-uuid-1234", vn.Users[0].ID)
				}
				if vn.Users[0].Encryption != "none" {
					t.Errorf("encryption = %q, want none", vn.Users[0].Encryption)
				}
				if vn.Users[0].Flow != "xtls-rprx-vision" {
					t.Errorf("flow = %q, want xtls-rprx-vision", vn.Users[0].Flow)
				}
				// Check stream settings
				ss := out.StreamSettings
				if ss == nil {
					t.Fatal("streamSettings is nil")
				}
				if ss.Network != "tcp" {
					t.Errorf("network = %q, want tcp", ss.Network)
				}
				if ss.Security != "reality" {
					t.Errorf("security = %q, want reality", ss.Security)
				}
				if ss.RealitySettings == nil {
					t.Fatal("realitySettings is nil")
				}
				if ss.RealitySettings.ServerName != "www.microsoft.com" {
					t.Errorf("serverName = %q, want www.microsoft.com", ss.RealitySettings.ServerName)
				}
				if ss.RealitySettings.Fingerprint != "chrome" {
					t.Errorf("fingerprint = %q, want chrome", ss.RealitySettings.Fingerprint)
				}
				if ss.RealitySettings.PublicKey != "test-public-key" {
					t.Errorf("publicKey = %q, want test-public-key", ss.RealitySettings.PublicKey)
				}
				if ss.RealitySettings.ShortID != "abcd1234" {
					t.Errorf("shortId = %q, want abcd1234", ss.RealitySettings.ShortID)
				}
			},
		},
		{
			name: "VLESS + TLS + ws",
			server: protocol.Server{
				Protocol:    protocol.ProtocolVLESS,
				Address:     "ws.example.com",
				Port:        443,
				UUID:        "ws-uuid-5678",
				Encryption:  "none",
				Network:     "ws",
				TLS:         "tls",
				SNI:         "ws.example.com",
				Path:        "/vless-ws",
				Host:        "ws.example.com",
				Fingerprint: "chrome",
				ALPN:        "h2,http/1.1",
			},
			socks:     1080,
			http:      8080,
			wantProto: "vless",
			check: func(t *testing.T, cfg *XrayConfig) {
				out := cfg.Outbounds[0]
				ss := out.StreamSettings
				if ss == nil {
					t.Fatal("streamSettings is nil")
				}
				if ss.Network != "ws" {
					t.Errorf("network = %q, want ws", ss.Network)
				}
				if ss.Security != "tls" {
					t.Errorf("security = %q, want tls", ss.Security)
				}
				if ss.WsSettings == nil {
					t.Fatal("wsSettings is nil")
				}
				if ss.WsSettings.Path != "/vless-ws" {
					t.Errorf("ws path = %q, want /vless-ws", ss.WsSettings.Path)
				}
				if ss.WsSettings.Headers["Host"] != "ws.example.com" {
					t.Errorf("ws host header = %q, want ws.example.com", ss.WsSettings.Headers["Host"])
				}
				if ss.TLSSettings == nil {
					t.Fatal("tlsSettings is nil")
				}
				if ss.TLSSettings.ServerName != "ws.example.com" {
					t.Errorf("serverName = %q, want ws.example.com", ss.TLSSettings.ServerName)
				}
				if ss.TLSSettings.Fingerprint != "chrome" {
					t.Errorf("fingerprint = %q, want chrome", ss.TLSSettings.Fingerprint)
				}
				// Check ALPN
				if len(ss.TLSSettings.ALPN) != 2 || ss.TLSSettings.ALPN[0] != "h2" || ss.TLSSettings.ALPN[1] != "http/1.1" {
					t.Errorf("alpn = %v, want [h2 http/1.1]", ss.TLSSettings.ALPN)
				}
				// VLESS+WS should not have flow set
				var settings struct {
					Vnext []struct {
						Users []struct {
							Flow string `json:"flow"`
						} `json:"users"`
					} `json:"vnext"`
				}
				if err := json.Unmarshal(out.Settings, &settings); err != nil {
					t.Fatalf("unmarshal: %v", err)
				}
				if len(settings.Vnext) > 0 && len(settings.Vnext[0].Users) > 0 {
					if settings.Vnext[0].Users[0].Flow != "" {
						t.Errorf("flow should be empty for ws, got %q", settings.Vnext[0].Users[0].Flow)
					}
				}
			},
		},
		{
			name: "VMess + TLS + ws",
			server: protocol.Server{
				Protocol: protocol.ProtocolVMess,
				Address:  "vmess.example.com",
				Port:     443,
				UUID:     "vmess-uuid-9999",
				AlterID:  0,
				Security: "auto",
				Network:  "ws",
				TLS:      "tls",
				SNI:      "vmess.example.com",
				Path:     "/vmess-path",
			},
			socks:     1080,
			http:      8080,
			wantProto: "vmess",
			check: func(t *testing.T, cfg *XrayConfig) {
				out := cfg.Outbounds[0]
				var settings struct {
					Vnext []struct {
						Address string `json:"address"`
						Port    int    `json:"port"`
						Users   []struct {
							ID       string `json:"id"`
							AlterID  int    `json:"alterId"`
							Security string `json:"security"`
						} `json:"users"`
					} `json:"vnext"`
				}
				if err := json.Unmarshal(out.Settings, &settings); err != nil {
					t.Fatalf("unmarshal vmess settings: %v", err)
				}
				if len(settings.Vnext) != 1 {
					t.Fatal("expected 1 vnext entry")
				}
				vn := settings.Vnext[0]
				if vn.Address != "vmess.example.com" {
					t.Errorf("address = %q, want vmess.example.com", vn.Address)
				}
				if vn.Users[0].ID != "vmess-uuid-9999" {
					t.Errorf("user id = %q, want vmess-uuid-9999", vn.Users[0].ID)
				}
				if vn.Users[0].AlterID != 0 {
					t.Errorf("alterId = %d, want 0", vn.Users[0].AlterID)
				}
				if vn.Users[0].Security != "auto" {
					t.Errorf("security = %q, want auto", vn.Users[0].Security)
				}
				// Stream settings
				ss := out.StreamSettings
				if ss.Network != "ws" {
					t.Errorf("network = %q, want ws", ss.Network)
				}
				if ss.Security != "tls" {
					t.Errorf("security = %q, want tls", ss.Security)
				}
				if ss.WsSettings == nil || ss.WsSettings.Path != "/vmess-path" {
					t.Error("wsSettings path incorrect")
				}
				if ss.TLSSettings == nil || ss.TLSSettings.ServerName != "vmess.example.com" {
					t.Error("tlsSettings serverName incorrect")
				}
			},
		},
		{
			name: "VMess + none + tcp",
			server: protocol.Server{
				Protocol: protocol.ProtocolVMess,
				Address:  "plain-vmess.example.com",
				Port:     10086,
				UUID:     "plain-vmess-uuid",
				AlterID:  0,
				Security: "auto",
				Network:  "tcp",
				TLS:      "none",
			},
			socks:     1080,
			http:      8080,
			wantProto: "vmess",
			check: func(t *testing.T, cfg *XrayConfig) {
				out := cfg.Outbounds[0]
				ss := out.StreamSettings
				if ss == nil {
					t.Fatal("streamSettings is nil")
				}
				if ss.Network != "tcp" {
					t.Errorf("network = %q, want tcp", ss.Network)
				}
				if ss.Security != "none" {
					t.Errorf("security = %q, want none", ss.Security)
				}
				if ss.TLSSettings != nil {
					t.Error("should not have tlsSettings for none security")
				}
				if ss.RealitySettings != nil {
					t.Error("should not have realitySettings for none security")
				}
			},
		},
		{
			name: "Trojan + TLS + tcp",
			server: protocol.Server{
				Protocol: protocol.ProtocolTrojan,
				Address:  "trojan.example.com",
				Port:     443,
				Password: "trojan-password-123",
				Network:  "tcp",
				TLS:      "tls",
				SNI:      "trojan.example.com",
			},
			socks:     1080,
			http:      8080,
			wantProto: "trojan",
			check: func(t *testing.T, cfg *XrayConfig) {
				out := cfg.Outbounds[0]
				var settings struct {
					Servers []struct {
						Address  string `json:"address"`
						Port     int    `json:"port"`
						Password string `json:"password"`
					} `json:"servers"`
				}
				if err := json.Unmarshal(out.Settings, &settings); err != nil {
					t.Fatalf("unmarshal trojan settings: %v", err)
				}
				if len(settings.Servers) != 1 {
					t.Fatal("expected 1 server entry")
				}
				srv := settings.Servers[0]
				if srv.Address != "trojan.example.com" {
					t.Errorf("address = %q, want trojan.example.com", srv.Address)
				}
				if srv.Port != 443 {
					t.Errorf("port = %d, want 443", srv.Port)
				}
				if srv.Password != "trojan-password-123" {
					t.Errorf("password = %q, want trojan-password-123", srv.Password)
				}
				// TLS settings
				ss := out.StreamSettings
				if ss.Security != "tls" {
					t.Errorf("security = %q, want tls", ss.Security)
				}
				if ss.TLSSettings == nil || ss.TLSSettings.ServerName != "trojan.example.com" {
					t.Error("tlsSettings serverName incorrect")
				}
			},
		},
		{
			name: "Trojan + TLS + grpc",
			server: protocol.Server{
				Protocol:    protocol.ProtocolTrojan,
				Address:     "grpc-trojan.example.com",
				Port:        443,
				Password:    "trojan-grpc-pass",
				Network:     "grpc",
				TLS:         "tls",
				SNI:         "grpc-trojan.example.com",
				ServiceName: "trojan-grpc",
			},
			socks:     1080,
			http:      8080,
			wantProto: "trojan",
			check: func(t *testing.T, cfg *XrayConfig) {
				out := cfg.Outbounds[0]
				ss := out.StreamSettings
				if ss.Network != "grpc" {
					t.Errorf("network = %q, want grpc", ss.Network)
				}
				if ss.GrpcSettings == nil {
					t.Fatal("grpcSettings is nil")
				}
				if ss.GrpcSettings.ServiceName != "trojan-grpc" {
					t.Errorf("serviceName = %q, want trojan-grpc", ss.GrpcSettings.ServiceName)
				}
				if ss.TLSSettings == nil || ss.TLSSettings.ServerName != "grpc-trojan.example.com" {
					t.Error("tlsSettings serverName incorrect")
				}
			},
		},
		{
			name: "Shadowsocks plain",
			server: protocol.Server{
				Protocol: protocol.ProtocolShadowsocks,
				Address:  "ss.example.com",
				Port:     8388,
				Method:   "aes-256-gcm",
				Password: "ss-password-456",
			},
			socks:     1080,
			http:      8080,
			wantProto: "shadowsocks",
			check: func(t *testing.T, cfg *XrayConfig) {
				out := cfg.Outbounds[0]
				var settings struct {
					Servers []struct {
						Address  string `json:"address"`
						Port     int    `json:"port"`
						Method   string `json:"method"`
						Password string `json:"password"`
					} `json:"servers"`
				}
				if err := json.Unmarshal(out.Settings, &settings); err != nil {
					t.Fatalf("unmarshal ss settings: %v", err)
				}
				if len(settings.Servers) != 1 {
					t.Fatal("expected 1 server entry")
				}
				srv := settings.Servers[0]
				if srv.Address != "ss.example.com" {
					t.Errorf("address = %q, want ss.example.com", srv.Address)
				}
				if srv.Port != 8388 {
					t.Errorf("port = %d, want 8388", srv.Port)
				}
				if srv.Method != "aes-256-gcm" {
					t.Errorf("method = %q, want aes-256-gcm", srv.Method)
				}
				if srv.Password != "ss-password-456" {
					t.Errorf("password = %q, want ss-password-456", srv.Password)
				}
				// Shadowsocks plain should have no stream settings
				if out.StreamSettings != nil {
					t.Error("plain shadowsocks should not have streamSettings")
				}
			},
		},
		{
			name: "Port configuration",
			server: protocol.Server{
				Protocol: protocol.ProtocolVLESS,
				Address:  "example.com",
				Port:     443,
				UUID:     "port-test-uuid",
				Network:  "tcp",
				TLS:      "none",
			},
			socks:     2080,
			http:      9080,
			wantProto: "vless",
			check: func(t *testing.T, cfg *XrayConfig) {
				if len(cfg.Inbounds) < 2 {
					t.Fatal("expected at least 2 inbounds")
				}
				socksIn := cfg.Inbounds[0]
				httpIn := cfg.Inbounds[1]
				if socksIn.Port != 2080 {
					t.Errorf("socks port = %d, want 2080", socksIn.Port)
				}
				if socksIn.Listen != "127.0.0.1" {
					t.Errorf("socks listen = %q, want 127.0.0.1", socksIn.Listen)
				}
				if socksIn.Protocol != "socks" {
					t.Errorf("socks protocol = %q, want socks", socksIn.Protocol)
				}
				if httpIn.Port != 9080 {
					t.Errorf("http port = %d, want 9080", httpIn.Port)
				}
				if httpIn.Listen != "127.0.0.1" {
					t.Errorf("http listen = %q, want 127.0.0.1", httpIn.Listen)
				}
				if httpIn.Protocol != "http" {
					t.Errorf("http protocol = %q, want http", httpIn.Protocol)
				}
			},
		},
		{
			name: "Invalid protocol returns error",
			server: protocol.Server{
				Protocol: "wireguard",
				Address:  "wg.example.com",
				Port:     51820,
			},
			socks:   1080,
			http:    8080,
			wantErr: true,
		},
		{
			name: "Routing config - geoip:private to direct",
			server: protocol.Server{
				Protocol: protocol.ProtocolVLESS,
				Address:  "example.com",
				Port:     443,
				UUID:     "routing-test-uuid",
				Network:  "tcp",
				TLS:      "none",
			},
			socks:     1080,
			http:      8080,
			wantProto: "vless",
			check: func(t *testing.T, cfg *XrayConfig) {
				// Check routing
				if cfg.Routing.DomainStrategy != "IPIfNonMatch" {
					t.Errorf("domainStrategy = %q, want IPIfNonMatch", cfg.Routing.DomainStrategy)
				}
				if len(cfg.Routing.Rules) < 1 {
					t.Fatal("expected at least 1 routing rule")
				}
				rule := cfg.Routing.Rules[0]
				if rule.OutboundTag != "direct" {
					t.Errorf("outboundTag = %q, want direct", rule.OutboundTag)
				}
				foundGeoip := false
				for _, ip := range rule.IP {
					if ip == "geoip:private" {
						foundGeoip = true
					}
				}
				if !foundGeoip {
					t.Error("expected geoip:private in routing rule IPs")
				}
				// Check direct outbound exists
				if len(cfg.Outbounds) < 2 {
					t.Fatal("expected at least 2 outbounds (proxy + direct)")
				}
				if cfg.Outbounds[1].Tag != "direct" || cfg.Outbounds[1].Protocol != "freedom" {
					t.Error("second outbound should be direct/freedom")
				}
			},
		},
		{
			name: "VLESS + httpupgrade transport",
			server: protocol.Server{
				Protocol: protocol.ProtocolVLESS,
				Address:  "hu.example.com",
				Port:     443,
				UUID:     "hu-uuid",
				Network:  "httpupgrade",
				TLS:      "tls",
				SNI:      "hu.example.com",
				Path:     "/hu-path",
				Host:     "hu.example.com",
			},
			socks:     1080,
			http:      8080,
			wantProto: "vless",
			check: func(t *testing.T, cfg *XrayConfig) {
				out := cfg.Outbounds[0]
				ss := out.StreamSettings
				if ss.Network != "httpupgrade" {
					t.Errorf("network = %q, want httpupgrade", ss.Network)
				}
				if ss.HttpUpgradeSettings == nil {
					t.Fatal("httpupgradeSettings is nil")
				}
				if ss.HttpUpgradeSettings.Path != "/hu-path" {
					t.Errorf("httpupgrade path = %q, want /hu-path", ss.HttpUpgradeSettings.Path)
				}
				if ss.HttpUpgradeSettings.Host != "hu.example.com" {
					t.Errorf("httpupgrade host = %q, want hu.example.com", ss.HttpUpgradeSettings.Host)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			xrayCfg, coreConfig, err := BuildConfig(tc.server, tc.socks, tc.http)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("BuildConfig error: %v", err)
			}
			if xrayCfg == nil {
				t.Fatal("xrayCfg is nil")
			}
			if coreConfig == nil {
				t.Fatal("coreConfig is nil (serial.LoadJSONConfig failed)")
			}
			// Check outbound protocol
			if xrayCfg.Outbounds[0].Protocol != tc.wantProto {
				t.Errorf("outbound protocol = %q, want %q", xrayCfg.Outbounds[0].Protocol, tc.wantProto)
			}
			// Run test-specific checks
			if tc.check != nil {
				tc.check(t, xrayCfg)
			}
		})
	}
}

// TestBuildConfigDefaults verifies default behaviors.
func TestBuildConfigDefaults(t *testing.T) {
	// VLESS with empty encryption should default to "none"
	srv := protocol.Server{
		Protocol: protocol.ProtocolVLESS,
		Address:  "example.com",
		Port:     443,
		UUID:     "test-uuid",
		Network:  "tcp",
		TLS:      "none",
	}
	xrayCfg, _, err := BuildConfig(srv, 1080, 8080)
	if err != nil {
		t.Fatalf("BuildConfig error: %v", err)
	}
	var settings struct {
		Vnext []struct {
			Users []struct {
				Encryption string `json:"encryption"`
			} `json:"users"`
		} `json:"vnext"`
	}
	if err := json.Unmarshal(xrayCfg.Outbounds[0].Settings, &settings); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if settings.Vnext[0].Users[0].Encryption != "none" {
		t.Errorf("default encryption = %q, want none", settings.Vnext[0].Users[0].Encryption)
	}

	// VMess with empty security should default to "auto"
	srv2 := protocol.Server{
		Protocol: protocol.ProtocolVMess,
		Address:  "example.com",
		Port:     443,
		UUID:     "test-uuid",
		Network:  "tcp",
		TLS:      "none",
	}
	xrayCfg2, _, err := BuildConfig(srv2, 1080, 8080)
	if err != nil {
		t.Fatalf("BuildConfig error: %v", err)
	}
	var settings2 struct {
		Vnext []struct {
			Users []struct {
				Security string `json:"security"`
			} `json:"users"`
		} `json:"vnext"`
	}
	if err := json.Unmarshal(xrayCfg2.Outbounds[0].Settings, &settings2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if settings2.Vnext[0].Users[0].Security != "auto" {
		t.Errorf("default security = %q, want auto", settings2.Vnext[0].Users[0].Security)
	}

	// REALITY with empty fingerprint should default to "chrome"
	srv3 := protocol.Server{
		Protocol:  protocol.ProtocolVLESS,
		Address:   "example.com",
		Port:      443,
		UUID:      "test-uuid",
		Network:   "tcp",
		TLS:       "reality",
		SNI:       "www.example.com",
		PublicKey: "pk123",
		ShortID:   "sid123",
	}
	xrayCfg3, _, err := BuildConfig(srv3, 1080, 8080)
	if err != nil {
		t.Fatalf("BuildConfig error: %v", err)
	}
	ss := xrayCfg3.Outbounds[0].StreamSettings
	if ss.RealitySettings.Fingerprint != "chrome" {
		t.Errorf("default fingerprint = %q, want chrome", ss.RealitySettings.Fingerprint)
	}
}
