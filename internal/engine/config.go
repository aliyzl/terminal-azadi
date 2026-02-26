package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/leejooy96/azad/internal/protocol"
	"github.com/leejooy96/azad/internal/splittunnel"
	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/infra/conf/serial"
)

// XrayConfig represents the top-level Xray JSON configuration.
type XrayConfig struct {
	Log       LogConfig        `json:"log"`
	Inbounds  []InboundConfig  `json:"inbounds"`
	Outbounds []OutboundConfig `json:"outbounds"`
	Routing   RoutingConfig    `json:"routing"`
}

// LogConfig configures Xray logging.
type LogConfig struct {
	Access   string `json:"access"`
	Error    string `json:"error"`
	LogLevel string `json:"loglevel"`
}

// SniffingConfig configures traffic sniffing on an inbound.
type SniffingConfig struct {
	Enabled      bool     `json:"enabled"`
	DestOverride []string `json:"destOverride"`
}

// InboundConfig represents an Xray inbound listener.
type InboundConfig struct {
	Tag      string          `json:"tag"`
	Listen   string          `json:"listen"`
	Port     int             `json:"port"`
	Protocol string          `json:"protocol"`
	Settings json.RawMessage `json:"settings,omitempty"`
	Sniffing *SniffingConfig `json:"sniffing,omitempty"`
}

// OutboundConfig represents an Xray outbound connection.
type OutboundConfig struct {
	Tag            string          `json:"tag"`
	Protocol       string          `json:"protocol"`
	Settings       json.RawMessage `json:"settings,omitempty"`
	StreamSettings *StreamSettings `json:"streamSettings,omitempty"`
}

// StreamSettings configures transport and security for an outbound.
type StreamSettings struct {
	Network             string               `json:"network"`
	Security            string               `json:"security"`
	TLSSettings         *TLSSettings         `json:"tlsSettings,omitempty"`
	RealitySettings     *RealitySettings     `json:"realitySettings,omitempty"`
	WsSettings          *WsSettings          `json:"wsSettings,omitempty"`
	GrpcSettings        *GrpcSettings        `json:"grpcSettings,omitempty"`
	HttpUpgradeSettings *HttpUpgradeSettings `json:"httpupgradeSettings,omitempty"`
}

// TLSSettings configures TLS for an outbound.
type TLSSettings struct {
	ServerName    string   `json:"serverName,omitempty"`
	Fingerprint   string   `json:"fingerprint,omitempty"`
	ALPN          []string `json:"alpn,omitempty"`
	AllowInsecure bool     `json:"allowInsecure,omitempty"`
}

// RealitySettings configures REALITY for an outbound.
type RealitySettings struct {
	ServerName  string `json:"serverName,omitempty"`
	Fingerprint string `json:"fingerprint,omitempty"`
	PublicKey   string `json:"publicKey,omitempty"`
	ShortID     string `json:"shortId,omitempty"`
	SpiderX     string `json:"spiderX,omitempty"`
}

// WsSettings configures WebSocket transport.
type WsSettings struct {
	Path    string            `json:"path,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

// GrpcSettings configures gRPC transport.
type GrpcSettings struct {
	ServiceName string `json:"serviceName,omitempty"`
}

// HttpUpgradeSettings configures HTTPUpgrade transport.
type HttpUpgradeSettings struct {
	Path string `json:"path,omitempty"`
	Host string `json:"host,omitempty"`
}

// RoutingConfig configures Xray routing rules.
type RoutingConfig struct {
	DomainStrategy string        `json:"domainStrategy"`
	Rules          []RoutingRule `json:"rules"`
}

// RoutingRule represents a single routing rule.
type RoutingRule struct {
	Type        string   `json:"type"`
	OutboundTag string   `json:"outboundTag"`
	IP          []string `json:"ip,omitempty"`
	Domain      []string `json:"domain,omitempty"`
}

// BuildConfig converts a protocol.Server into a valid Xray *core.Config.
// It returns both the intermediate XrayConfig (for inspection/testing) and
// the loaded core.Config ready for core.New().
// splitCfg may be nil, in which case behavior is identical to pre-split-tunnel.
// accessLogPath sets the Xray access log destination; use "none" to disable.
func BuildConfig(srv protocol.Server, socksPort, httpPort int, splitCfg *splittunnel.Config, accessLogPath string) (*XrayConfig, *core.Config, error) {
	// Build protocol-specific outbound.
	outbound, err := buildOutbound(srv)
	if err != nil {
		return nil, nil, err
	}

	sniffing := &SniffingConfig{
		Enabled:      true,
		DestOverride: []string{"http", "tls"},
	}

	// Determine outbound ordering based on split tunnel mode.
	var outbounds []OutboundConfig
	if splitCfg != nil && splitCfg.Enabled && splitCfg.Mode == splittunnel.ModeInclusive {
		// Inclusive: direct first (default for unmatched), proxy for listed
		outbounds = []OutboundConfig{
			{Tag: "direct", Protocol: "freedom"},
			outbound,
		}
	} else {
		// Normal / Exclusive: proxy first (default for unmatched), direct for listed
		outbounds = []OutboundConfig{
			outbound,
			{Tag: "direct", Protocol: "freedom"},
		}
	}

	// Build routing rules.
	domainStrategy := "AsIs"
	var rules []RoutingRule

	if splitCfg != nil && splitCfg.Enabled && len(splitCfg.Rules) > 0 {
		// User split tunnel rules go FIRST (highest priority, before geoip:private).
		for _, xr := range splittunnel.ToXrayRules(splitCfg.Rules, splitCfg.Mode) {
			rules = append(rules, RoutingRule{
				Type:        xr.Type,
				OutboundTag: xr.OutboundTag,
				IP:          xr.IP,
				Domain:      xr.Domain,
			})
		}

		// Change domain strategy if any domain rules exist.
		if splittunnel.HasDomainRules(splitCfg.Rules) {
			domainStrategy = "IPIfNonMatch"
		}
	}

	// Private IPs always direct (safety net) -- always last.
	rules = append(rules, RoutingRule{
		Type:        "field",
		OutboundTag: "direct",
		IP:          []string{"geoip:private"},
	})

	cfg := &XrayConfig{
		Log: LogConfig{Access: accessLogPath, Error: "none", LogLevel: "warning"},
		Inbounds: []InboundConfig{
			{
				Tag:      "socks-in",
				Listen:   "127.0.0.1",
				Port:     socksPort,
				Protocol: "socks",
				Settings: json.RawMessage(`{"udp":true}`),
				Sniffing: sniffing,
			},
			{
				Tag:      "http-in",
				Listen:   "127.0.0.1",
				Port:     httpPort,
				Protocol: "http",
				Sniffing: sniffing,
			},
		},
		Outbounds: outbounds,
		Routing: RoutingConfig{
			DomainStrategy: domainStrategy,
			Rules:          rules,
		},
	}

	// Marshal to JSON and load through Xray's config pipeline.
	jsonBytes, err := json.Marshal(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("marshaling xray config: %w", err)
	}

	coreConfig, err := serial.LoadJSONConfig(bytes.NewReader(jsonBytes))
	if err != nil {
		return nil, nil, fmt.Errorf("loading xray config: %w", err)
	}

	return cfg, coreConfig, nil
}

// buildOutbound constructs the protocol-specific outbound config.
func buildOutbound(srv protocol.Server) (OutboundConfig, error) {
	var out OutboundConfig

	switch srv.Protocol {
	case protocol.ProtocolVLESS:
		out = buildVLESSOutbound(srv)
	case protocol.ProtocolVMess:
		out = buildVMessOutbound(srv)
	case protocol.ProtocolTrojan:
		out = buildTrojanOutbound(srv)
	case protocol.ProtocolShadowsocks:
		out = buildShadowsocksOutbound(srv)
	default:
		return OutboundConfig{}, fmt.Errorf("unsupported protocol: %s", srv.Protocol)
	}

	return out, nil
}

// buildVLESSOutbound constructs a VLESS outbound.
func buildVLESSOutbound(srv protocol.Server) OutboundConfig {
	encryption := srv.Encryption
	if encryption == "" {
		encryption = "none"
	}

	type vlessUser struct {
		ID         string `json:"id"`
		Encryption string `json:"encryption"`
		Flow       string `json:"flow,omitempty"`
	}
	type vnextEntry struct {
		Address string      `json:"address"`
		Port    int         `json:"port"`
		Users   []vlessUser `json:"users"`
	}
	type vlessSettings struct {
		Vnext []vnextEntry `json:"vnext"`
	}

	settings := vlessSettings{
		Vnext: []vnextEntry{
			{
				Address: srv.Address,
				Port:    srv.Port,
				Users: []vlessUser{
					{
						ID:         srv.UUID,
						Encryption: encryption,
						Flow:       srv.Flow,
					},
				},
			},
		},
	}

	settingsJSON, _ := json.Marshal(settings)

	out := OutboundConfig{
		Tag:      "proxy",
		Protocol: "vless",
		Settings: settingsJSON,
	}

	out.StreamSettings = buildStreamSettings(srv)
	return out
}

// buildVMessOutbound constructs a VMess outbound.
func buildVMessOutbound(srv protocol.Server) OutboundConfig {
	security := srv.Security
	if security == "" {
		security = "auto"
	}

	type vmessUser struct {
		ID       string `json:"id"`
		AlterID  int    `json:"alterId"`
		Security string `json:"security"`
	}
	type vnextEntry struct {
		Address string      `json:"address"`
		Port    int         `json:"port"`
		Users   []vmessUser `json:"users"`
	}
	type vmessSettings struct {
		Vnext []vnextEntry `json:"vnext"`
	}

	settings := vmessSettings{
		Vnext: []vnextEntry{
			{
				Address: srv.Address,
				Port:    srv.Port,
				Users: []vmessUser{
					{
						ID:       srv.UUID,
						AlterID:  srv.AlterID,
						Security: security,
					},
				},
			},
		},
	}

	settingsJSON, _ := json.Marshal(settings)

	out := OutboundConfig{
		Tag:      "proxy",
		Protocol: "vmess",
		Settings: settingsJSON,
	}

	out.StreamSettings = buildStreamSettings(srv)
	return out
}

// buildTrojanOutbound constructs a Trojan outbound.
func buildTrojanOutbound(srv protocol.Server) OutboundConfig {
	type trojanServer struct {
		Address  string `json:"address"`
		Port     int    `json:"port"`
		Password string `json:"password"`
	}
	type trojanSettings struct {
		Servers []trojanServer `json:"servers"`
	}

	settings := trojanSettings{
		Servers: []trojanServer{
			{
				Address:  srv.Address,
				Port:     srv.Port,
				Password: srv.Password,
			},
		},
	}

	settingsJSON, _ := json.Marshal(settings)

	out := OutboundConfig{
		Tag:      "proxy",
		Protocol: "trojan",
		Settings: settingsJSON,
	}

	out.StreamSettings = buildStreamSettings(srv)
	return out
}

// buildShadowsocksOutbound constructs a Shadowsocks outbound.
func buildShadowsocksOutbound(srv protocol.Server) OutboundConfig {
	type ssServer struct {
		Address  string `json:"address"`
		Port     int    `json:"port"`
		Method   string `json:"method"`
		Password string `json:"password"`
	}
	type ssSettings struct {
		Servers []ssServer `json:"servers"`
	}

	settings := ssSettings{
		Servers: []ssServer{
			{
				Address:  srv.Address,
				Port:     srv.Port,
				Method:   srv.Method,
				Password: srv.Password,
			},
		},
	}

	settingsJSON, _ := json.Marshal(settings)

	out := OutboundConfig{
		Tag:      "proxy",
		Protocol: "shadowsocks",
		Settings: settingsJSON,
	}

	// Shadowsocks with no network/TLS specified gets no stream settings.
	if srv.Network != "" && srv.Network != "tcp" {
		out.StreamSettings = buildStreamSettings(srv)
	}

	return out
}

// buildStreamSettings constructs transport and security settings.
func buildStreamSettings(srv protocol.Server) *StreamSettings {
	network := srv.Network
	if network == "" {
		network = "tcp"
	}

	security := srv.TLS
	if security == "" {
		security = "none"
	}

	ss := &StreamSettings{
		Network:  network,
		Security: security,
	}

	// Transport-specific settings.
	switch network {
	case "ws":
		ws := &WsSettings{
			Path: srv.Path,
		}
		if srv.Host != "" {
			ws.Headers = map[string]string{"Host": srv.Host}
		}
		ss.WsSettings = ws
	case "grpc":
		ss.GrpcSettings = &GrpcSettings{
			ServiceName: srv.ServiceName,
		}
	case "httpupgrade":
		ss.HttpUpgradeSettings = &HttpUpgradeSettings{
			Path: srv.Path,
			Host: srv.Host,
		}
	}

	// Security-specific settings.
	switch security {
	case "tls":
		tls := &TLSSettings{
			ServerName:  srv.SNI,
			Fingerprint: srv.Fingerprint,
		}
		if srv.ALPN != "" {
			tls.ALPN = strings.Split(srv.ALPN, ",")
		}
		if srv.AllowInsecure {
			tls.AllowInsecure = true
		}
		ss.TLSSettings = tls
	case "reality":
		fingerprint := srv.Fingerprint
		if fingerprint == "" {
			fingerprint = "chrome"
		}
		ss.RealitySettings = &RealitySettings{
			ServerName:  srv.SNI,
			Fingerprint: fingerprint,
			PublicKey:   srv.PublicKey,
			ShortID:     srv.ShortID,
			SpiderX:     srv.SpiderX,
		}
	}

	return ss
}
