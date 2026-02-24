package subscription

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// vmessPayload constructs a valid vmess:// URI for testing.
func vmessPayload(name, addr string, port int, uuid string) string {
	payload := map[string]interface{}{
		"v":    2,
		"ps":   name,
		"add":  addr,
		"port": port,
		"id":   uuid,
		"aid":  0,
		"net":  "tcp",
		"type": "none",
		"host": "",
		"path": "",
		"tls":  "",
	}
	data, _ := json.Marshal(payload)
	return "vmess://" + base64.StdEncoding.EncodeToString(data)
}

func TestFetch_ValidSubscription(t *testing.T) {
	lines := []string{
		"vless://550e8400-e29b-41d4-a716-446655440000@server1.example.com:443?type=tcp&security=tls&sni=server1.example.com#Server1",
		"vless://660e8400-e29b-41d4-a716-446655440000@server2.example.com:443?type=ws&security=tls&sni=server2.example.com#Server2",
		"vless://770e8400-e29b-41d4-a716-446655440000@server3.example.com:443?type=grpc&security=tls&sni=server3.example.com&serviceName=grpc#Server3",
	}
	body := base64.StdEncoding.EncodeToString([]byte(strings.Join(lines, "\n")))

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, body)
	}))
	defer ts.Close()

	servers, err := Fetch(ts.URL)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(servers) != 3 {
		t.Fatalf("expected 3 servers, got %d", len(servers))
	}

	for _, s := range servers {
		if s.SubscriptionSource != ts.URL {
			t.Errorf("SubscriptionSource: got %q, want %q", s.SubscriptionSource, ts.URL)
		}
	}
}

func TestFetch_MixedProtocols(t *testing.T) {
	lines := []string{
		"vless://550e8400-e29b-41d4-a716-446655440000@vless.example.com:443?type=tcp&security=tls&sni=vless.example.com#VLESS",
		vmessPayload("VMess", "vmess.example.com", 8080, "660e8400-e29b-41d4-a716-446655440000"),
		"trojan://password123@trojan.example.com:443?type=tcp&security=tls&sni=trojan.example.com#Trojan",
		"ss://YWVzLTI1Ni1nY206cGFzc3dvcmQ@ss.example.com:8388#SS",
	}
	body := base64.StdEncoding.EncodeToString([]byte(strings.Join(lines, "\n")))

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, body)
	}))
	defer ts.Close()

	servers, err := Fetch(ts.URL)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(servers) != 4 {
		t.Fatalf("expected 4 servers, got %d", len(servers))
	}
}

func TestFetch_URLSafeBase64(t *testing.T) {
	lines := []string{
		"vless://550e8400-e29b-41d4-a716-446655440000@server1.example.com:443?type=tcp&security=tls#URLSafe",
	}
	// Use URL-safe base64 without padding
	body := base64.RawURLEncoding.EncodeToString([]byte(strings.Join(lines, "\n")))

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, body)
	}))
	defer ts.Close()

	servers, err := Fetch(ts.URL)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}
	if servers[0].Name != "URLSafe" {
		t.Errorf("name: got %q, want %q", servers[0].Name, "URLSafe")
	}
}

func TestFetch_WindowsLineEndings(t *testing.T) {
	lines := []string{
		"vless://550e8400-e29b-41d4-a716-446655440000@server1.example.com:443?type=tcp&security=tls#Win1",
		"vless://660e8400-e29b-41d4-a716-446655440000@server2.example.com:443?type=tcp&security=tls#Win2",
	}
	// Join with \r\n to simulate Windows line endings
	content := strings.Join(lines, "\r\n")
	body := base64.StdEncoding.EncodeToString([]byte(content))

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, body)
	}))
	defer ts.Close()

	servers, err := Fetch(ts.URL)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(servers) != 2 {
		t.Fatalf("expected 2 servers, got %d", len(servers))
	}
}

func TestFetch_BOMPrefix(t *testing.T) {
	lines := []string{
		"vless://550e8400-e29b-41d4-a716-446655440000@server1.example.com:443?type=tcp&security=tls#BOM",
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(strings.Join(lines, "\n")))
	// Prepend UTF-8 BOM
	bodyBytes := append([]byte{0xEF, 0xBB, 0xBF}, []byte(encoded)...)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(bodyBytes)
	}))
	defer ts.Close()

	servers, err := Fetch(ts.URL)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}
	if servers[0].Name != "BOM" {
		t.Errorf("name: got %q, want %q", servers[0].Name, "BOM")
	}
}

func TestFetch_InvalidLines(t *testing.T) {
	lines := []string{
		"vless://550e8400-e29b-41d4-a716-446655440000@server1.example.com:443?type=tcp&security=tls#Valid1",
		"this is garbage not a valid URI",
		"vless://660e8400-e29b-41d4-a716-446655440000@server2.example.com:443?type=tcp&security=tls#Valid2",
	}
	body := base64.StdEncoding.EncodeToString([]byte(strings.Join(lines, "\n")))

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, body)
	}))
	defer ts.Close()

	servers, err := Fetch(ts.URL)
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if len(servers) != 2 {
		t.Fatalf("expected 2 servers (garbage skipped), got %d", len(servers))
	}
}

func TestFetch_AllInvalid(t *testing.T) {
	lines := []string{
		"garbage line 1",
		"garbage line 2",
		"also not a valid URI",
	}
	body := base64.StdEncoding.EncodeToString([]byte(strings.Join(lines, "\n")))

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, body)
	}))
	defer ts.Close()

	_, err := Fetch(ts.URL)
	if err == nil {
		t.Fatal("expected error for all-invalid subscription")
	}
	if !strings.Contains(err.Error(), "no valid server URIs") {
		t.Errorf("error should mention 'no valid server URIs', got: %v", err)
	}
}

func TestFetch_HTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	_, err := Fetch(ts.URL)
	if err == nil {
		t.Fatal("expected error for HTTP 404")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("error should mention HTTP status, got: %v", err)
	}
}

func TestFetch_EmptyBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		// empty body
	}))
	defer ts.Close()

	_, err := Fetch(ts.URL)
	if err == nil {
		t.Fatal("expected error for empty body")
	}
}

func TestDecodeSubscription_Variants(t *testing.T) {
	original := "vless://uuid@host:443#name\ntrojan://pass@host2:443#name2"

	tests := []struct {
		name    string
		encoder func([]byte) string
	}{
		{"StdEncoding", func(b []byte) string { return base64.StdEncoding.EncodeToString(b) }},
		{"RawStdEncoding", func(b []byte) string { return base64.RawStdEncoding.EncodeToString(b) }},
		{"URLEncoding", func(b []byte) string { return base64.URLEncoding.EncodeToString(b) }},
		{"RawURLEncoding", func(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encoded := tt.encoder([]byte(original))
			decoded, err := DecodeSubscription([]byte(encoded))
			if err != nil {
				t.Fatalf("DecodeSubscription(%s): %v", tt.name, err)
			}
			// After decoding, line endings should be normalized
			if !strings.Contains(decoded, "vless://uuid@host:443#name") {
				t.Errorf("decoded content missing expected line, got: %q", decoded)
			}
		})
	}
}
