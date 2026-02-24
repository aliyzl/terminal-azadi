package subscription

import (
	"fmt"
)

// DecodeSubscription decodes a subscription response body.
// It strips BOM, decodes base64 (trying all 4 variants),
// and normalizes line endings to \n.
func DecodeSubscription(body []byte) (string, error) {
	return "", fmt.Errorf("not implemented")
}
