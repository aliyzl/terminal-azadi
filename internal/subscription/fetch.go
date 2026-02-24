package subscription

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/leejooy96/azad/internal/protocol"
)

// Fetch downloads a subscription URL, decodes the base64 response,
// and returns a list of parsed servers with SubscriptionSource set.
func Fetch(subscriptionURL string) ([]*protocol.Server, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	req, err := http.NewRequest("GET", subscriptionURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", "Azad/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching subscription: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("subscription returned HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading subscription body: %w", err)
	}

	if len(body) == 0 {
		return nil, fmt.Errorf("subscription returned empty body")
	}

	decoded, err := DecodeSubscription(body)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(decoded, "\n")
	var servers []*protocol.Server
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		server, err := protocol.ParseURI(line)
		if err != nil {
			// Skip invalid lines -- they may be comments or unsupported protocols
			continue
		}
		server.SubscriptionSource = subscriptionURL
		servers = append(servers, server)
	}

	if len(servers) == 0 {
		return nil, fmt.Errorf("subscription contained no valid server URIs")
	}

	return servers, nil
}
