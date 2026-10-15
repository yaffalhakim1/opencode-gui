# ROADMAP — 5-day plan

Budget: unlimited credits, 5 calendar days. Scope is frozen at the bottom of each day; anything unfinished rolls to the Android repo's backlog, not into the next day.

## Verified API facts (from docs, not guesses)

- Shared service: `opencode service start/status/restart`; registers at `~/.local/state/opencode/service.json` with `{url, pid, password}`
- Auth: `Basic opencode:<password from service.json>` on every request
- Health: `GET /api/info` → `{version, pid, urls, capabilities}`
- Prompt: `POST /api/session/{id}/prompt` `{text}` — **async**, reply arrives in messages
- Messages: `GET /api/session/{id}/message` → `{data: [{type: "assistant"|"user", content: [{type:"text",text}]}]}` (no `role`/`parts` — v1 shapes are gone)
- Responses wrapped in `{data: ...}` envelopes
- Events: `GET /api/event` (SSE)
- Interrupt: `POST /api/session/{id}/interrupt`
- Debug tool: `opencode api get /api/info` (CLI does discovery+auth for you)
- Pairing (for Android later): `POST /api/pair` creates a code; `GET /auth/connect/{code}` returns a session token as password

## Day 1 — Skeleton + round-trip ✅
- ~~`mygo init -template native opencode-gui`~~ done
- ~~attach to registered opencode service (docs: client/service model)~~ done
- ~~CreateSession + Prompt + poll Messages round-trip~~ **verified: ROUNDTRIP-OK**
- ~~GUI: sidebar session list + prompt input + reply~~ wired (markdown replies)

## Day 2 — Streaming chat ✅
- SSE consumer of `/api/event` with reconnect (`RefreshLoop`)
- Live chat via `markdown.Stream` / `markdown.View`, follow-end list
- Session switch, new session, interrupt button

## Day 3 — Daemon ✅
- `internal/daemon`: WS server `:9680`, token auth (constant-time), dispatcher
- Token persisted in Windows Credential Manager (`mygo.Secrets`)
- Tailscale IP detection; sidebar shows connected phones + copy URL
- **Verified**: bad token refused; over the Tailscale IP, `system.health` →
  `session.create` → `session.prompt` → reply `PHONE-OK`

## Day 4 — Diff inspector ✅
- `editor.ParsePatch` + `editor.DiffView` in the right pane
- Changed-file list per session with add/remove counts, click to view

## Day 5 — Ship ✅
- `go tool mygo build` → `build/windows-amd64/OpenCode GUI.exe` (13.3 MB) + installer
- README + docs updated with verified facts

## Post-v1 (Android repo, Jetpack Compose)
- Connect page per `docs/PAIRING.md` (QR scan + manual)
- Chat list → session → streaming chat
- Permission approve/reject notifications
- Consumes only `docs/PROTOCOL.md` — frozen contract.
