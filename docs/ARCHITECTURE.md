# Architecture — opencode-gui (working name)

A MyGo desktop GUI for opencode, with a daemon so an Android app (later) can talk to it over Tailscale.

## Verdict on your two ideas

1. **Daemon WebSocket — correct, and it's the same daemon.** The MyGo desktop app *is* the daemon. One Go binary runs: (a) the native-UI GUI, (b) a small LAN WebSocket server. The Android app connects to the WS server over Tailscale. Don't build a separate daemon process — two processes, two sets of state, no benefit at this scale.

2. **Tailscale — correct, and it removes 90% of the security work.** Tailscale gives mutual auth + encryption on the WireGuard layer. The WS server binds to the Tailscale interface (or `0.0.0.0` + token) and never touches the public internet. Do NOT build your own auth crypto; a single shared token is enough defense-in-depth.

## The layering (bottom-up)

```
┌─────────────────────────────────────────────┐
│ Android (Jetpack Compose) — later           │
│   connects to: ws://<tailnet-ip>:9680/ws    │
├─────────────────────────────────────────────┤
│ Daemon layer (Go, inside the same binary)   │
│   - LAN WS server :9680 (tailscale iface)   │
│   - fan-out: opencode SSE events → WS       │
│   - fan-in:  WS commands → opencode HTTP    │
├─────────────────────────────────────────────┤
│ opencode server (child process, port :4096) │
│   `opencode serve` — HTTP + SSE, per-proj   │
├─────────────────────────────────────────────┤
│ MyGo desktop app (this repo, Windows first) │
│   - NATIVE UI (package ui, no webview)      │
│   - view = pure function of app state       │
└─────────────────────────────────────────────┘
```

### Key decisions

- **opencode as child process, not embedded SDK.** `createOpencode()` (JS SDK) is JS; we're Go. Go spawns `opencode serve --port 4096`, health-checks `GET /global/health`, kills it on exit. Simplest correct path.
- **One project = one opencode server instance.** opencode serves one working directory per instance. v1: single project (pick dir at startup). Multi-project later = spawn more instances on more ports, map them.
- **Go talks to opencode, frontend never does.** All `/session`, `/event`, `/file` calls go through bound services. The frontend has zero opencode HTTP knowledge — one contract owner, Android later gets the same contract via WS.
- **SSE → fan-out.** One goroutine consumes opencode's `GET /event` SSE stream, converts to typed Go events (`mygo.NewEvent`) for the GUI + broadcasts JSON to WS clients. Both consumers see the identical event stream.
- **WS protocol = the same JSON shapes as the bound services.** Request: `{id, method, params}`, Response: `{id, result|error}`. Not a formal RPC framework — a 100-line dispatcher that reuses the same handler functions the MyGo bindings call. Phone and desktop share one mental model.

### WS message contract (v1)

```
→ {"id":1,"method":"session.list","params":{}}
← {"id":1,"result":[...]}
→ {"id":2,"method":"session.prompt","params":{"id":"ses_..","text":"hi"}}
← {"id":2,"result":{...}}            (fires prompt_async under the hood)
← {"event":"message.updated","properties":{...}}   (SSE passthrough, no id)
← {"event":"permission.updated","properties":{...}}
→ {"id":3,"method":"permission.respond","params":{"id":"..","response":"once"}}
```

## Repo layout

```
opencode-gui/
├── main.go                 app entry: bind services, start daemon, run app
├── internal/
│   ├── opencode/           client of `opencode serve`: HTTP, SSE, process mgmt
│   │   ├── process.go      spawn/health/stop the child process
│   │   ├── client.go       typed REST calls (sessions, prompt, files)
│   │   └── events.go       SSE consumer → typed events
│   ├── daemon/             LAN WS server for Android
│   │   ├── server.go       :9680, token auth, tailscale iface bind
│   │   └── protocol.go     request/response/event envelope, dispatcher
│   └── app/                services called by the native view AND daemon handlers
│       └── service.go      SessionService, FileService, DaemonService
├── internal/ui/            native views (package ui): sidebar, chat, inspector
├── docs/
│   └── ARCHITECTURE.md     this file
├── mygo.json               app name, identifier, version (native template)
└── go.mod
```

## Security

- WS server requires `?token=` (or `Authorization: Bearer`), token generated at first run, shown in GUI settings, rotatable.
- Bind to `0.0.0.0:9680` but document tailscale-only firewall; v1.1: detect tailscale IP (100.x) and bind there. Treat the token as the real gate.
- Never expose `/file/content` or shell endpoints to WS in v1. Phone gets read/chat + permission approve. File browsing stays desktop-only until explicitly needed.

## 5-day plan

| Day | Deliverable |
|---|---|
| 1 | `mygo init` web-frontend app; `internal/opencode`: spawn `opencode serve`, health check, session list + prompt round-trip in bound service; GUI shows it |
| 2 | SSE consumer → mygo events; chat view streaming parts (text/tool/step), session create/switch, abort |
| 3 | Daemon: WS server, token auth, dispatcher sharing handlers with bound services; test with `wscat` |
| 4 | Permission flow (dialog in GUI + approve over WS), file diff view (`/session/:id/diff`), polish (markdown, code highlight) |
| 5 | Windows build (`mygo build`), installer, README, smoke-test over real Tailscale from phone browser (pre-Compose sanity check) |

Android Compose app = separate repo, later. It only needs: one WS URL + token, and the message contract above.
