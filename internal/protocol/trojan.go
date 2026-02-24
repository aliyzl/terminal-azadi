package protocol

import (
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// ParseTrojan parses a trojan:// URI into a Server struct.
func ParseTrojan(uri string) (*Server, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("invalid Trojan URI: %w", err)
	}
	if u.Scheme != "trojan" {
		return nil, fmt.Errorf("expected trojan:// scheme, got %s://", u.Scheme)
	}

	password := u.User.Username()
	if password == "" {
		return nil, fmt.Errorf("Trojan URI missing password")
	}

	host := u.Hostname()
	if host == "" {
		return nil, fmt.Errorf("Trojan URI missing host")
	}

	portStr := u.Port()
	port := 443 // default for Trojan
	if portStr != "" {
		port, err = strconv.Atoi(portStr)
		if err != nil {
			return nil, fmt.Errorf("Trojan URI invalid port: %w", err)
		}
	}

	q := u.Query()
	name := u.Fragment
	if name == "" {
		name = fmt.Sprintf("%s:%d", host, port)
	}

	return &Server{
		ID:          NewID(),
		Name:        name,
		Protocol:    ProtocolTrojan,
		Address:     host,
		Port:        port,
		Password:    password,
		Flow:        q.Get("flow"),
		Network:     defaultString(q.Get("type"), "tcp"),
		TLS:         defaultString(q.Get("security"), "tls"),
		SNI:         q.Get("sni"),
		Fingerprint: q.Get("fp"),
		Path:        q.Get("path"),
		Host:        q.Get("host"),
		ServiceName: q.Get("serviceName"),
		ALPN:        q.Get("alpn"),
		PublicKey:   q.Get("pbk"),
		ShortID:     q.Get("sid"),
		SpiderX:     q.Get("spx"),
		AddedAt:     time.Now(),
		RawURI:      uri,
	}, nil
}
