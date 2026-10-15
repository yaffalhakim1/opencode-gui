# ROADMAP — 5-day plan

Budget: unlimited credits, 5 calendar days. Scope is frozen at the bottom of each day; anything unfinished rolls to the Android repo's backlog, not into the next day.

## Day 1 — Skeleton + round-trip
- `mygo init opencode-gui` (web frontend, Vite + TS)
- `internal/opencode`: spawn `opencode serve --port 4096` as child process, poll `/global/health`, clean shutdown
- `internal/app`: bound `SessionService` — list, create, prompt (sync round-trip)
- GUI: session list + prompt box + plain-text response
- **Done when:** you type in the desktop app and get an answer.

## Day 2 — Streaming chat
- `internal/opencode/events.go`: SSE consumer of `/event`, reconnected on drop
- Fan-out: Go typed events to frontend (`mygo.NewEvent`)
- Chat view: streaming parts (text / tool calls / steps), markdown + code highlight
- Session switch, new session, abort, `session.idle` stops the spinner
- **Done when:** streaming feels live and abort works.

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
