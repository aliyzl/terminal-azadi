package protocol

import (
	"crypto/rand"
	"fmt"
	"time"
)

// Protocol represents a supported proxy protocol.
type Protocol string

const (
	ProtocolVLESS       Protocol = "vless"
	ProtocolVMess       Protocol = "vmess"
	ProtocolTrojan      Protocol = "trojan"
	ProtocolShadowsocks Protocol = "shadowsocks"
)

// Server represents a parsed proxy server with rich metadata.
type Server struct {
	// Identity
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Protocol Protocol `json:"protocol"`

	// Connection
	Address string `json:"address"`
	Port    int    `json:"port"`

	// Authentication (protocol-dependent)
	UUID     string `json:"uuid,omitempty"`
	Password string `json:"password,omitempty"`
	AlterID  int    `json:"alter_id,omitempty"`

	// Encryption
	Encryption string `json:"encryption,omitempty"`
	Method     string `json:"method,omitempty"`
	Security   string `json:"security,omitempty"`

	// Transport
	Network     string `json:"network,omitempty"`
	Path        string `json:"path,omitempty"`
	Host        string `json:"host,omitempty"`
	ServiceName string `json:"service_name,omitempty"`

	// TLS
	TLS           string `json:"tls,omitempty"`
	SNI           string `json:"sni,omitempty"`
	ALPN          string `json:"alpn,omitempty"`
	Fingerprint   string `json:"fingerprint,omitempty"`
	AllowInsecure bool   `json:"allow_insecure,omitempty"`

	// REALITY-specific
	PublicKey string `json:"public_key,omitempty"`
	ShortID   string `json:"short_id,omitempty"`
	SpiderX   string `json:"spider_x,omitempty"`

	// Flow
	Flow string `json:"flow,omitempty"`

	// VMess-specific
	Type string `json:"type,omitempty"`

	// Metadata
	SubscriptionSource string    `json:"subscription_source,omitempty"`
	LatencyMs          int       `json:"latency_ms,omitempty"`
	LastConnected      time.Time `json:"last_connected,omitempty"`
	AddedAt            time.Time `json:"added_at"`
	RawURI             string    `json:"raw_uri,omitempty"`
}

// NewID generates a UUID v4 string using crypto/rand.
func NewID() string {
	var uuid [16]byte
	_, _ = rand.Read(uuid[:])
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // variant 2
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])
}
