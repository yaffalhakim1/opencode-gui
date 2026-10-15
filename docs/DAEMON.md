# DAEMON — LAN WebSocket server design

The daemon is a goroutine inside the desktop app, not a separate process.

## Lifecycle

```
App.WhenReady
 ├─ load-or-create token        (mygo.Secrets, key "daemon-token")
 ├─ ensure opencode serve up    (internal/opencode)
 ├─ http.ListenAndServe(":9680") in goroutine
 │    └─ /ws → token check → upgrade → register conn
 ├─ SSE bridge goroutine: opencode /event → Broadcast to all conns + GUI events
 └─ App exit → graceful close: daemon.shutdown event, 500ms, close listener, kill opencode
```

## Components (`internal/daemon/`)

```
server.go     http.Server, /ws handler, token gate, ping loop (30s), idle cutoff (60s)
registry.go   conn set, per-conn 256-event buffer, broadcast (drop slow conns)
protocol.go   envelope types, dispatcher: method → handler func
handlers.go   method implementations, call internal/app services (shared with MyGo bindings)
```

### Handler sharing

The same `internal/app` service methods serve both:
- MyGo bound services (frontend → Go)
- daemon handlers (Android → dispatcher → same methods)

A method is "WS-exposed" only if it is in `docs/PROTOCOL.md`. The dispatcher maps `session.prompt` → `app.SessionService.PromptAsyncWS` — thin wrappers that strip desktop-only args. Desktop-only capabilities (file read, dialogs, shell) never get a WS wrapper.

## Binding & security

- Default bind: `0.0.0.0:9680` (works whether or not Tailscale is up). When `tailscale ip -4` succeeds, the Settings page shows the tailscale address as the recommended connect target.
- Token: 32 random bytes, base64url, in Windows Credential Manager via `mygo.Secrets` (survives reinstall, encrypted per-user). Compared with `subtle.ConstantTimeCompare`.
- 401 on bad token, connection never upgraded. No rate-limit needed — the port is only reachable from the tailnet/LAN, and a wrong-token flood is cheap to reject.
- No TLS: Tailscale is the transport security. GUI warns if user connects over plain LAN IP.

## Windows notes

- First run, Windows Firewall may prompt for the listener — the installer README mentions allowing it on **private networks** only. When using Tailscale exclusively, the prompt can be denied (loopback/tailnet traffic doesn't traverse the firewall rule for inbound public).
- Port 9680 chosen: uncommon, memorable, away from opencode's 4096.

## Tailscale detection

```go
func tailscaleIPv4() (string, error) {
    out, err := exec.Command("tailscale", "ip", "-4").Output()
    // trim, validate 100.x
}
```

Also read MagicDNS name via `tailscale status --json` → `Self.DNSName` for display. CLI not found / logged out → GUI shows "Tailscale: not connected" and LAN IP as fallback.
