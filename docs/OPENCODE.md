# OPENCODE — integration notes (verified against v2)

Everything here is confirmed against the running v2 server, the OpenAPI spec
(`https://opencode.ai/v2/openapi.json`), and the docs. No assumptions.

## The service model

opencode v2 runs **one shared background server per user**. Clients discover
it rather than starting their own:

- `opencode service status | start | restart | stop`
- It registers itself at `~/.local/state/opencode/service.json`:
  ```json
  { "id": "...", "version": "2.0.28", "url": "http://127.0.0.1:49374", "pid": 20048, "password": "..." }
  ```
- Clients authenticate with HTTP Basic, user `opencode`, password from that file.
  (This is exactly what the official JS `Service.headers(endpoint)` builds.)

Our app calls `opencode.Discover()` (reads that file) then `opencode.Attach(ep)`.
It never spawns or kills a server.

## Endpoints used

| Purpose | Method & path | Notes |
|---|---|---|
| Health | `GET /api/info` | `{version, pid, urls, capabilities}` |
| List sessions | `GET /api/session?order=desc` | `{data: [...]}` |
| Create session | `POST /api/session` | body `{title?}` |
| Send prompt | `POST /api/session/{id}/prompt` | body `{text}` — **async** |
| List messages | `GET /api/session/{id}/message` | `{data: [...]}` |
| Interrupt | `POST /api/session/{id}/interrupt` | |
| Diff | `GET /api/session/{id}/diff` | `{data: [{file,patch,additions,deletions,status}]}` |
| Events | `GET /api/event` | SSE, `event:` + JSON `data:` |

## Shapes that differ from v1 (these broke the first attempt)

- Responses are wrapped: `{ "location"?: ..., "data": ... }`.
- Messages are `{id, type: "assistant"|"user", content: [{type, text}], finish}` —
  **not** `{info, parts}` with a `role`. Assistant content blocks may be
  `text`, `reasoning`, or `tool`; the chat view renders `text` only.
- `finish` is present on an assistant message once the model stopped; while it
  is empty the reply is still streaming.
- Prompts are **async**: the call returns once the input is admitted; the reply
  arrives in the message list (we refresh on SSE events).

## Events

`GET /api/event` is a live-only SSE stream (no replay). Our `RefreshLoop`
subscribes, and when an event's payload mentions the active session, re-reads
that session's messages and diffs. It reconnects with backoff when the stream
drops. Subscriptions are per-process; the daemon re-broadcasts a
`session.updated` event to phones from the same refresh path.

## Debugging

- `opencode api get /api/info` — the CLI does discovery + auth for you.
- `opencode api agent.list` — confirm agents (build, plan, …).
- Logs: `~/.local/share/opencode/log/opencode.log`, filter `role=server`.
