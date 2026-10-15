# opencode integration — what we call and how

opencode runs as a **child process** of the desktop app.

## Process management (`internal/opencode/process.go`)

```
start:    exec.Command("opencode", "serve", "--port", "4096", "--hostname", "127.0.0.1")
health:   GET /global/health → {"healthy":true,"version":"..."}  (poll 250ms until ok, 10s timeout)
crash:    process exits → restart with backoff (1s, 2s, 4s … 30s), emit "daemon" event to GUI + WS
shutdown: app quit → POST nothing; kill process group (Windows: cmd.Process.Kill on the job)
```

If port 4096 is taken by an unrelated server: try 4097..4105, remember the chosen port for the session. Never kill an existing listener we didn't spawn.

## Endpoints used (from official docs)

Only what the GUI + daemon need:

| Endpoint | Used by |
|---|---|
| `GET /global/health` | process supervisor |
| `GET /event` (SSE) | event bridge (single consumer, fans out) |
| `GET /session`, `POST /session`, `GET /session/:id`, `DELETE /session/:id`, `GET /session/:id/message` | session list / chat history |
| `POST /session/:id/prompt_async` | send prompt without blocking |
| `POST /session/:id/abort` | stop button |
| `GET /session/:id/diff` | diff view |
| `POST /session/:id/permissions/:permissionID` | permission respond |
| `GET /agent`, `GET /config/providers` | model/agent pickers |

Not used (v1): `/file*`, `/find*`, `/tui/*`, `/mcp`, `/log`, share endpoints.

## SSE bridge (`internal/opencode/events.go`)

- One goroutine, `GET /event`, parse `event:` + `data:` lines, decode into `Event{Type, Properties}` (properties stay raw JSON).
- On stream error: reconnect with backoff, no event loss guarantee (opencode provides none) — after reconnect, refresh sessions from REST to resync.
- Fan-out to: (a) MyGo typed events per window, (b) daemon broadcast. Ordering preserved per consumer.

## Message shapes

Prompt body:

```json
{ "parts": [{ "type": "text", "text": "..." }], "agent": "build" }
```

Model selection (optional): `"model": { "providerID": "...", "modelID": "..." }`.

Message list returns `{ info: Message, parts: Part[] }[]` — the GUI renders parts by `type` (text / tool / step / patch), phone renders text + tool-status only in v1.

## Versioning

`GET /global/health` gives the version; the GUI displays it. If opencode's API drifts, the OpenAPI spec lives at `/doc` — regen point for typed client if needed. We hand-roll a thin client (≈10 endpoints); regen tooling is YAGNI until drift hurts.
