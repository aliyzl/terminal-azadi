package engine

import (
	"encoding/json"
	"fmt"

	"github.com/leejooy96/azad/internal/protocol"
	"github.com/xtls/xray-core/core"
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
	LogLevel string `json:"loglevel"`
}

// InboundConfig represents an Xray inbound listener.
type InboundConfig struct {
	Tag      string          `json:"tag"`
	Listen   string          `json:"listen"`
	Port     int             `json:"port"`
	Protocol string          `json:"protocol"`
	Settings json.RawMessage `json:"settings,omitempty"`
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
}

// BuildConfig converts a protocol.Server into a valid Xray *core.Config.
// It returns both the intermediate XrayConfig (for inspection/testing) and
// the loaded core.Config ready for core.New().
func BuildConfig(srv protocol.Server, socksPort, httpPort int) (*XrayConfig, *core.Config, error) {
	_ = srv
	_ = socksPort
	_ = httpPort
	return nil, nil, fmt.Errorf("not implemented")
}
