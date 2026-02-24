package protocol

import (
	"fmt"
	"strings"
)

// ParseURI detects the protocol scheme and delegates to the appropriate parser.
func ParseURI(uri string) (*Server, error) {
	uri = strings.TrimSpace(uri)
	if uri == "" {
		return nil, fmt.Errorf("empty URI")
	}

	switch {
	case strings.HasPrefix(uri, "vless://"):
		return ParseVLESS(uri)
	case strings.HasPrefix(uri, "vmess://"):
		return ParseVMess(uri)
	case strings.HasPrefix(uri, "trojan://"):
		return ParseTrojan(uri)
	case strings.HasPrefix(uri, "ss://"):
		return ParseShadowsocks(uri)
	default:
		return nil, fmt.Errorf("unsupported protocol scheme in URI: %q", uri)
	}
}
