# Fix: Xray routing youtube.com as direct instead of proxy

## Symptom

Traffic to youtube.com logged as `[http-in -> direct]` instead of `[http-in >> proxy]`.

## Root Cause

Two issues combined:

1. **`sniffing` was on the outbound** — xray silently ignores it there. It belongs on each **inbound**. Without sniffing, xray can't extract domain names from TLS for routing.

2. **`domainStrategy: "IPIfNonMatch"`** resolves domains to IPs via system DNS. In restricted networks, poisoned DNS returns private/bogus IPs for blocked domains. Those match `geoip:private` rule → routed direct.

## Fix

1. Move `sniffing` from outbound to each inbound:
```json
"inbounds": [
  { "tag": "socks-in", ..., "sniffing": { "enabled": true, "destOverride": ["http", "tls"] } },
  { "tag": "http-in", ..., "sniffing": { "enabled": true, "destOverride": ["http", "tls"] } }
]
```

2. Change `domainStrategy` from `"IPIfNonMatch"` to `"AsIs"` — no domain-based rules exist, so DNS resolution is unnecessary. Unmatched traffic falls through to the first outbound (proxy).

## Files Changed

- `internal/engine/config.go` — Added SniffingConfig to InboundConfig, changed domainStrategy
- `config.json`, `config.template.json` — Same fixes in static configs
- `link-to-full-config.sh` — Same
- `vless-link-to-config.sh` — Removed incorrect outbound sniffing
