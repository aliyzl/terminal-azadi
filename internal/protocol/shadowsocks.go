package protocol

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ParseShadowsocks parses an ss:// URI into a Server struct.
func ParseShadowsocks(uri string) (*Server, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("invalid Shadowsocks URI: %w", err)
	}
	if u.Scheme != "ss" {
		return nil, fmt.Errorf("expected ss:// scheme, got %s://", u.Scheme)
	}

	host := u.Hostname()
	portStr := u.Port()
	if host == "" || portStr == "" {
		return nil, fmt.Errorf("Shadowsocks URI missing host or port")
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("Shadowsocks URI invalid port: %w", err)
	}

	// Detect userinfo format: base64 or plaintext
	var method, password string
	userinfo := u.User.String()
	if strings.Contains(userinfo, ":") {
		// Plaintext format: method:password (AEAD-2022 or percent-encoded)
		method = u.User.Username()
		password, _ = u.User.Password()
	} else {
		// Base64-encoded format: base64(method:password)
		decoded, err := decodeBase64(userinfo)
		if err != nil {
			return nil, fmt.Errorf("Shadowsocks URI userinfo decode failed: %w", err)
		}
		parts := strings.SplitN(string(decoded), ":", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("Shadowsocks URI: decoded userinfo missing method:password")
		}
		method = parts[0]
		password = parts[1]
	}

	name := u.Fragment
	if name == "" {
		name = fmt.Sprintf("%s:%d", host, port)
	}

	return &Server{
		ID:       NewID(),
		Name:     name,
		Protocol: ProtocolShadowsocks,
		Address:  host,
		Port:     port,
		Method:   method,
		Password: password,
		AddedAt:  time.Now(),
		RawURI:   uri,
	}, nil
}
