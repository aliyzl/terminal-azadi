package subscription

import (
	"encoding/base64"
	"fmt"
	"strings"
)

// utf8BOM is the byte sequence for a UTF-8 Byte Order Mark.
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// DecodeSubscription decodes a subscription response body.
// It strips BOM, decodes base64 (trying all 4 variants),
// and normalizes line endings to \n.
func DecodeSubscription(body []byte) (string, error) {
	// Strip UTF-8 BOM if present
	if len(body) >= 3 && body[0] == utf8BOM[0] && body[1] == utf8BOM[1] && body[2] == utf8BOM[2] {
		body = body[3:]
	}

	// Trim whitespace
	s := strings.TrimSpace(string(body))
	if s == "" {
		return "", fmt.Errorf("empty subscription body")
	}

	// Decode base64 using fallback chain
	decoded, err := decodeBase64(s)
	if err != nil {
		return "", fmt.Errorf("decoding subscription body: %w", err)
	}

	// Normalize line endings: \r\n -> \n, standalone \r -> \n
	result := strings.ReplaceAll(string(decoded), "\r\n", "\n")
	result = strings.ReplaceAll(result, "\r", "\n")

	return result, nil
}

// decodeBase64 tries StdEncoding, RawStdEncoding, URLEncoding, RawURLEncoding
// in order, returning the first successful decode.
func decodeBase64(s string) ([]byte, error) {
	s = strings.TrimSpace(s)

	// Try standard encoding (with padding)
	if decoded, err := base64.StdEncoding.DecodeString(s); err == nil {
		return decoded, nil
	}

	// Try standard encoding without padding
	if decoded, err := base64.RawStdEncoding.DecodeString(s); err == nil {
		return decoded, nil
	}

	// Try URL-safe encoding (with padding)
	if decoded, err := base64.URLEncoding.DecodeString(s); err == nil {
		return decoded, nil
	}

	// Try URL-safe encoding without padding
	if decoded, err := base64.RawURLEncoding.DecodeString(s); err == nil {
		return decoded, nil
	}

	return nil, fmt.Errorf("failed to decode base64: not valid in any encoding variant")
}
