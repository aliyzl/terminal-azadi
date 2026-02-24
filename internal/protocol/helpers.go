package protocol

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// defaultString returns val if non-empty, else fallback.
func defaultString(val, fallback string) string {
	if val != "" {
		return val
	}
	return fallback
}

// decodeBase64 tries StdEncoding, RawStdEncoding, URLEncoding, RawURLEncoding
// in order, returning the first successful decode. Trims whitespace first.
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

// jsonFlexInt handles both string "443" and number 443 in JSON.
type jsonFlexInt int

func (f *jsonFlexInt) UnmarshalJSON(data []byte) error {
	// Try number first
	var n int
	if err := json.Unmarshal(data, &n); err == nil {
		*f = jsonFlexInt(n)
		return nil
	}
	// Try string
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		n, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf("cannot convert %q to int: %w", s, err)
		}
		*f = jsonFlexInt(n)
		return nil
	}
	return fmt.Errorf("jsonFlexInt: cannot unmarshal %s", string(data))
}
