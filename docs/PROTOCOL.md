# WS PROTOCOL — Daemon ⇄ Android (and any WS client)

One WebSocket endpoint: `ws://<host>:9680/ws?token=<token>`.

## Envelopes

**Request** (client → daemon):

```json
{ "id": 1, "method": "session.list", "params": {} }
```

**Response** (daemon → client, matches `id`):

```json
{ "id": 1, "result": [ ... ] }
```

```json
{ "id": 2, "error": { "code": "upstream_error", "message": "opencode not running" } }
```

**Event** (daemon → client, no `id`, pushed anytime):

```json
{ "event": "message.updated", "properties": { ... } }
```

Events are opencode SSE events passed through verbatim (`type` → `event`, `properties` untouched), plus daemon-native events listed at the bottom.

## Methods (v1)

### session

| Method | Params | Result |
|---|---|---|
| `session.list` | `{}` | `Session[]` |
| `session.get` | `{ id }` | `Session` |
| `session.create` | `{ title? }` | `Session` |
| `session.delete` | `{ id }` | `true` |
| `session.messages` | `{ id, limit? }` | `{ info, parts }[]` |
| `session.prompt` | `{ id, text, agent?, providerID?, modelID? }` | `true` (fires `prompt_async`; answers arrive as events) |
| `session.abort` | `{ id }` | `true` |
| `session.diff` | `{ id, messageID? }` | `FileDiff[]` |

### permission

| Method | Params | Result |
|---|---|---|
| `permission.list` | `{}` | pending permissions (daemon cache from SSE) |
| `permission.respond` | `{ sessionID, permissionID, response: "once"\|"always"\|"reject" }` | `true` |

### system

| Method | Params | Result |
|---|---|---|
| `system.health` | `{}` | `{ ok: true, version, project: "<dir>" }` |
| `system.agents` | `{}` | `Agent[]` |
| `system.providers` | `{}` | `{ providers, default }` |

**Deliberately absent in v1:** file read/write, shell, find — the phone gets chat + approvals only. Adding these later is one dispatcher case each, but they widen the attack surface for zero day-1 value.

## Event catalog

Passthrough of all opencode `/event` SSE events, notably:

- `message.updated`, `message.part.updated`, `message.removed`
- `session.updated`, `session.deleted`, `session.idle`
- `permission.updated`, `permission.replied`
- `session.error`

Daemon-native:

| Event | Properties | Meaning |
|---|---|---|
| `daemon.hello` | `{ version, project }` | sent immediately after connect succeeds |
| `daemon.shutdown` | `{}` | desktop app is quitting; client should reconnect later |

## Behavior rules

- **One request at a time per connection is NOT required** — clients may pipeline; daemon serializes upstream calls per opencode instance.
- **No `id` → dropped** with an error envelope using `"id": null`.
- **Unknown method** → error `unknown_method`.
- **Heartbeat**: server sends WS ping every 30s; client pong. Client silent > 60s → disconnected. (Reconnect with backoff 1s → 2s → 4s → max 30s.)
- **Auth**: token checked on the HTTP upgrade request. Fail = HTTP 401, socket never opened.
- **Backpressure**: SSE events broadcast with a per-client 256-event buffer; slow client whose buffer fills is dropped rather than blocking the fan-out.

## Example flow (Android chat)

```
→ {"id":1,"method":"system.health","params":{}}
← {"id":1,"result":{"ok":true,"version":"...","project":"Y:/Learn/Go"}}
← {"event":"daemon.hello","properties":{...}}

→ {"id":2,"method":"session.create","params":{"title":"Phone"}}
← {"id":2,"result":{"id":"ses_123",...}}

→ {"id":3,"method":"session.prompt","params":{"id":"ses_123","text":"explain this repo"}}
← {"id":3,"result":true}
← {"event":"message.updated","properties":{...}}
← {"event":"message.part.updated","properties":{...}}   ← repeat while streaming
← {"event":"session.idle","properties":{"sessionID":"ses_123"}}
```
