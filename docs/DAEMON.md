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

- Bind: `0.0.0.0:9680`, so it works with or without Tailscale up. The sidebar
  shows the Tailscale address (detected via `tailscale ip -4`, falling back to
  the MagicDNS name) as the connect target.
- Token: 32 random bytes, base64url, stored in the Windows Credential Manager
  via `mygo.Secrets` (service `dev.yafit.opencodegui`, account `daemon-token`),
  so a paired phone keeps working across restarts. Compared with
  `subtle.ConstantTimeCompare`.
- 401 on bad token, connection never upgraded (verified in `scripts/daemontest.go`).
- No TLS: Tailscale is the transport security. Plain-LAN use is unencrypted.
- Per-client 256-message send buffer; a client that falls behind is dropped
  rather than blocking the fan-out.

## Verified behaviour

`scripts/daemontest.go` (local) and `scripts/phoneflow.go` (over the tailnet IP)
both pass: bad token refused, `daemon.hello` on connect, and the documented
`system.health → session.create → session.prompt → session.messages` flow
returning the model's reply.

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
