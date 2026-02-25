package engine

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

// ipCheckURL is the service used to determine the external IP address.
const ipCheckURL = "https://icanhazip.com"

// VerifyIP fetches the external IP address through the SOCKS5 proxy
// running on the given port. This confirms that traffic is being routed
// through the proxy and returns the exit IP.
func VerifyIP(socksPort int) (string, error) {
	dialer, err := proxy.SOCKS5("tcp", fmt.Sprintf("127.0.0.1:%d", socksPort), nil, proxy.Direct)
	if err != nil {
		return "", fmt.Errorf("creating SOCKS5 dialer: %w", err)
	}

	transport := &http.Transport{
		Dial: dialer.Dial,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   10 * time.Second,
	}

	resp, err := client.Get(ipCheckURL)
	if err != nil {
		return "", fmt.Errorf("fetching IP through proxy: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading IP response: %w", err)
	}

	return strings.TrimSpace(string(body)), nil
}

// GetDirectIP fetches the external IP address without using a proxy.
// This is used to compare against the proxy IP to confirm routing.
func GetDirectIP() (string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(ipCheckURL)
	if err != nil {
		return "", fmt.Errorf("fetching direct IP: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading IP response: %w", err)
	}

	return strings.TrimSpace(string(body)), nil
}
