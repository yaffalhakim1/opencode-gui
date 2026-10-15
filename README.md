# opencode-gui

A desktop GUI for [opencode](https://opencode.ai), built with [MyGo](https://mygo.egoist.dev) (Go + system webview), with a built-in daemon so an Android app can drive it over Tailscale.

**One binary, two front doors:**

- Desktop GUI (webview) for chatting with opencode agents
- LAN WebSocket daemon (`:9680`) that phones connect to via the Tailscale IP + token

## Docs

| Doc | Contents |
|---|---|
| [ARCHITECTURE.md](docs/ARCHITECTURE.md) | Layering, decisions, repo layout |
| [ROADMAP.md](docs/ROADMAP.md) | 5-day build plan |
| [PROTOCOL.md](docs/PROTOCOL.md) | Daemon WS message contract (Android consumes this) |
| [PAIRING.md](docs/PAIRING.md) | Connect page: IP, token, QR, Tailscale research |
| [DAEMON.md](docs/DAEMON.md) | Daemon internals, security, Windows notes |
| [OPENCODE.md](docs/OPENCODE.md) | opencode child process + endpoints used |
| [GUI.md](docs/GUI.md) | Desktop frontend design |

## Quick start (once code lands, day 1+)

```bash
# prerequisites: Go 1.27+, Bun, opencode on PATH
bun run dev          # runs mygo dev (rebuilds Go + frontend)
bun run build        # packaged Windows app in build/
```

## Connecting a phone

1. Install Tailscale on desktop + phone, same tailnet
2. Desktop app → Settings → scan the QR (or copy URL + token)
3. Android app (separate repo) → Connect page → scan → chat
