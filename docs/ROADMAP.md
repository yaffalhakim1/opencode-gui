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

## Day 2 — Streaming chat
- SSE consumer of `/api/event`, reconnect on drop
- Update GUI state from SSE (no polling): message parts appended live
- Markdown streaming (mygo `markdown.Stream`), session switch, new session, interrupt button
- **Read first:** https://mygo.egoist.dev/docs/windows (Windows window options, taskbar, DIPs) before finalizing window code
- **Done when:** streaming feels live and interrupt works.

## Day 3 — Daemon
- `internal/daemon`: WS server `:9680`, token auth (Secrets store, constant-time compare), protocol dispatcher per `docs/PROTOCOL.md`
- SSE fan-out also broadcasts to WS clients
- `tailscale ip -4` detection, Settings page: QR + token + copy, regenerate button
- **Done when:** `wscat` (or a curl→WS script) completes the example flow in `PROTOCOL.md` over the Tailscale IP.

## Day 4 — Permissions + polish
- Permission flow: SSE `permission.updated` → GUI dialog → `POST /session/:id/permissions/:id`; same surfaced to Android as event + `permission.respond`
- Session diff view (`/session/:id/diff`)
- Error states: opencode crash/restart, daemon client disconnects, missing Tailscale
- Tray icon + "daemon running" indicator
- **Done when:** a tool-call permission can be approved from the phone while the desktop shows the same dialog.

## Day 5 — Ship desktop
- `mygo build` → Windows installer (`build/windows-amd64/`)
- README: install, first run, connect phone (QR)
- Real test: phone on cellular via Tailscale, run a prompt, approve a permission
- Tag v0.1.0

## Post-v1 (Android repo, Jetpack Compose)
- Connect page per `docs/PAIRING.md` (QR scan + manual)
- Chat list → session → streaming chat
- Permission approve/reject notifications (foreground service optional)
- Repo: separate, consumes only `docs/PROTOCOL.md` — contract is frozen there first.
