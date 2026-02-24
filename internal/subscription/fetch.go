package subscription

import (
	"fmt"

	"github.com/leejooy96/azad/internal/protocol"
)

// Fetch downloads a subscription URL, decodes the base64 response,
// and returns a list of parsed servers.
func Fetch(subscriptionURL string) ([]*protocol.Server, error) {
	return nil, fmt.Errorf("not implemented")
}
