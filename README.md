# opencode-gui

A native desktop GUI for [opencode](https://opencode.ai), built with [MyGo](https://mygo.egoist.dev) (pure Go, GPU-drawn, no webview), with a built-in WebSocket daemon so a phone can drive the same sessions over Tailscale.

One binary, two front doors:

- **Desktop**: sidebar of sessions, streaming Markdown chat, and a diff inspector — drawn natively in Go.
- **Phone**: an Android app (later) connects to the daemon over your Tailscale IP with a token, and shares the desktop's opencode sessions.

## Status

Day 1–5 of the plan in [`docs/ROADMAP.md`](docs/ROADMAP.md) are done and verified end-to-end:

- attaches to the shared opencode service (official v2 client/service model)
- sessions: list, create, switch
- prompts: send, stream the reply (SSE), interrupt
- diff inspector per session (`editor.DiffView`)
- WebSocket daemon on `:9680` with token auth, reachable over Tailscale
- packaged Windows app + installer

## Requirements

- Go 1.27+
- [opencode](https://opencode.ai) v2 installed and its background service running (`opencode service status`)
- [Tailscale](https://tailscale.com) (for phone access; optional on the desktop alone)

## Run

```powershell
go tool mygo dev      # develop with live reload
go test ./...         # all tests, no window needed
go tool mygo build    # packaged app in build/windows-amd64/
```

## Connect a phone

1. Install Tailscale on the desktop and the phone, on the same tailnet.
2. Start the app. In the sidebar, **Daemon → Copy connect URL** gives
   `ws://<tailscale-ip>:9680/ws?token=<token>`.
3. The Android app's Connect page takes that URL (or scans it as a QR). See
   [`docs/PAIRING.md`](docs/PAIRING.md) and [`docs/PROTOCOL.md`](docs/PROTOCOL.md).

The token is stored in the Windows Credential Manager and survives restarts.

## Docs

| Doc | Contents |
|---|---|
| [ARCHITECTURE.md](docs/ARCHITECTURE.md) | Layering, decisions, repo layout |
| [ROADMAP.md](docs/ROADMAP.md) | Build plan + verified API facts |
| [PROTOCOL.md](docs/PROTOCOL.md) | Daemon WS message contract (what Android consumes) |
| [PAIRING.md](docs/PAIRING.md) | Connect page: IP, token, QR, Tailscale |
| [DAEMON.md](docs/DAEMON.md) | Daemon internals and security |
| [OPENCODE.md](docs/OPENCODE.md) | opencode integration notes |
| [GUI.md](docs/GUI.md) | Desktop UI design |
