package protocol

import (
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// ParseVLESS parses a vless:// URI into a Server struct.
func ParseVLESS(uri string) (*Server, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("invalid VLESS URI: %w", err)
	}
	if u.Scheme != "vless" {
		return nil, fmt.Errorf("expected vless:// scheme, got %s://", u.Scheme)
	}

	uuid := u.User.Username()
	if uuid == "" {
		return nil, fmt.Errorf("VLESS URI missing UUID")
	}

	host := u.Hostname()
	portStr := u.Port()
	if host == "" || portStr == "" {
		return nil, fmt.Errorf("VLESS URI missing host or port")
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("VLESS URI invalid port %q: %w", portStr, err)
	}

	q := u.Query()
	name := u.Fragment
	if name == "" {
		name = fmt.Sprintf("%s:%d", host, port)
	}

	return &Server{
		ID:            NewID(),
		Name:          name,
		Protocol:      ProtocolVLESS,
		Address:       host,
		Port:          port,
		UUID:          uuid,
		Encryption:    q.Get("encryption"),
		Flow:          q.Get("flow"),
		Network:       defaultString(q.Get("type"), "tcp"),
		TLS:           defaultString(q.Get("security"), "none"),
		SNI:           q.Get("sni"),
		Fingerprint:   q.Get("fp"),
		PublicKey:     q.Get("pbk"),
		ShortID:       q.Get("sid"),
		SpiderX:       q.Get("spx"),
		Path:          q.Get("path"),
		Host:          q.Get("host"),
		ServiceName:   q.Get("serviceName"),
		ALPN:          q.Get("alpn"),
		AllowInsecure: q.Get("allowInsecure") == "1" || q.Get("allowInsecure") == "true",
		AddedAt:       time.Now(),
		RawURI:        uri,
	}, nil
}
