package protocol

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// vmessJSON represents the JSON payload inside a vmess:// URI.
type vmessJSON struct {
	V    jsonFlexInt `json:"v"`
	PS   string     `json:"ps"`
	Add  string     `json:"add"`
	Port jsonFlexInt `json:"port"`
	ID   string     `json:"id"`
	Aid  jsonFlexInt `json:"aid"`
	Net  string     `json:"net"`
	Type string     `json:"type"`
	Host string     `json:"host"`
	Path string     `json:"path"`
	TLS  string     `json:"tls"`
	SNI  string     `json:"sni"`
	ALPN string     `json:"alpn"`
	Fp   string     `json:"fp"`
}

// ParseVMess parses a vmess:// URI into a Server struct.
func ParseVMess(uri string) (*Server, error) {
	raw := strings.TrimPrefix(uri, "vmess://")
	data, err := decodeBase64(raw)
	if err != nil {
		return nil, fmt.Errorf("VMess URI base64 decode failed: %w", err)
	}

	var v vmessJSON
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("VMess URI JSON parse failed: %w", err)
	}

	if v.Add == "" {
		return nil, fmt.Errorf("VMess URI missing server address")
	}
	port := int(v.Port)
	if port == 0 {
		return nil, fmt.Errorf("VMess URI missing or zero port")
	}

	name := v.PS
	if name == "" {
		name = fmt.Sprintf("%s:%d", v.Add, port)
	}

	return &Server{
		ID:          NewID(),
		Name:        name,
		Protocol:    ProtocolVMess,
		Address:     v.Add,
		Port:        port,
		UUID:        v.ID,
		AlterID:     int(v.Aid),
		Security:    "auto",
		Network:     defaultString(v.Net, "tcp"),
		Type:        v.Type,
		Host:        v.Host,
		Path:        v.Path,
		TLS:         v.TLS,
		SNI:         v.SNI,
		ALPN:        v.ALPN,
		Fingerprint: v.Fp,
		AddedAt:     time.Now(),
		RawURI:      uri,
	}, nil
}
