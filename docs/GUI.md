# GUI — desktop frontend design

MyGo app, web frontend (Vite + TypeScript). Go owns data; the page renders.

## Screens (v1, single window 1200×800)

```
┌──────────┬──────────────────────────────────┐
│ Sessions │  Chat                            │
│ ──────── │  ┌ messages (markdown) ───────┐  │
│ + New    │  │ user: ...                   │  │
│ ses_1  ● │  │ assistant: ... [tool: read] │  │
│ ses_2    │  └─────────────────────────────┘  │
│ ses_3    │  [prompt input            ] [↵]   │
│ ──────── │                                    │
│ Settings │  [diff] [abort] [model: …]        │
└──────────┴──────────────────────────────────┘
```

- **Sessions sidebar** — list, activity dot (running/idle), delete
- **Chat** — parts rendered by type; tool calls collapsible; streaming appends live
- **Settings** — daemon section (QR, URL, token, regenerate, tailscale status), opencode section (version, project dir, restart)
- **Permission dialog** — modal on `permission.updated`, Approve once / Always / Deny; mirrors to phone via daemon

## Data flow

```
Go service (internal/app)
  → mygo.NewEvent("session.updated" | "message.updated" | ...) → src/state.ts subscribes
  → svelte-free tiny store: Map + listeners, re-render via template funcs
  → commands: await SessionService.list() (generated src/mygo.ts)
```

No framework decision ceremony: v1 uses **plain TS + lit-html** (tiny, no build config beyond Vite default). If the UI grows past ~10 components, swap to Preact — one import change.

State: single `AppStore` with `{ sessions, messagesBySession, pendingPermissions, daemon }`. Events mutate it; views are pure functions of it.

## Error handling

- opencode down → banner "opencode restarting…" (from supervisor event)
- prompt fails → inline error row in chat
- daemon client dropped → Settings shows connected clients count

## Out of scope v1

Multi-project, themes, keybinds, file browser (use TUI for that), share links.
