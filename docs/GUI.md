# GUI — native UI design (MyGo `package ui`)

No webview, no Bun/Vite/TypeScript. One Go binary; the window is drawn on the GPU. Design language follows **Apple's fluid-interface principles** (WWDC): instant response, interruptible springs, translucent depth, size-specific typography.

## Window layout

```
┌──────────────────────────────────────────────────────────────┐
│  ui.Split (horizontal, draggable divider)                    │
│ ┌──────────┬───────────────────────────┬───────────────────┐ │
│ │ Sidebar  │  Chat                     │  Inspector pane   │ │
│ │ (Source  │                           │  (Collapsible)    │ │
│ │  list)   │  messages stream in       │                   │ │
│ │          │                           │  Segmented:       │ │
│ │ Sessions │  ui.Markdown              │   Diff | Terminal │ │
│ │  ● ses_1 │   (streams LLM replies,   │       | Files     │ │
│ │  ○ ses_2 │    selectable, copyable)  │                   │ │
│ │          │                           │  Diff viewer      │ │
│ │ Projects │  [prompt — Text input,    │  (unified/side)   │ │
│ │  Y:/Go   │   multi-line, ↵ send]     │  File tree        │ │
│ │          │                           │  Terminal         │ │
│ │ Daemon ● │                           │                   │ │
│ └──────────┴───────────────────────────┴───────────────────┘ │
│  Toolbar: [model ▾] [agent ▾] [abort] spinner while running  │
└──────────────────────────────────────────────────────────────┘
```

### Panes

- **Sidebar** (`ui.Sidebar`) — source list in sections: *Sessions* (activity dot: spinner while running, idle otherwise), *Projects* (v1: one), *Daemon* status row. Native Finder-style, follows desktop theme.
- **Chat** (`ui.Split` center pane) — `ui.Scroll` of messages; assistant text is `ui.Markdown` (built-in: CommonMark + GitHub flavor, **designed for streaming replies**, selectable). Tool calls = `ui.Collapsible` rows with a `ui.Spinner` while running, status `ui.Badge` when done.
- **Inspector** (`ui.Collapsible` / second `ui.Split` pane, toggled in toolbar) — `ui.Segmented` control:
  - **Diff** → `ui.DiffViewer` (built-in: unified/side-by-side, fed from `GET /session/:id/diff`)
  - **Files** → `ui.Tree` of the project root
  - **Terminal** → v2 (v1 ships Diff + Files; terminal needs a pty, defer)

## State & view model

View is a pure function of state (`app struct`); SSE events mutate state from goroutines (`ui.Notify` / state-change pattern per docs/ui/views). No data binding ceremony.

```
appState {
  sessions []Session          activeSessionID
  parts    map[sessionID][]Part   // appended live by SSE
  running  map[sessionID]bool
  pendingPermission  *Permission
  inspectorTab  (diff|files)
}
```

## Apple-design rules applied

| Principle | In this app |
|---|---|
| **Respond instantly** | Send button highlights on press (MyGo native default); prompt submits on Enter, never waits on network for feedback — "sent" state is local, streaming follows |
| **Springs, interruptible** | Inspector pane open/close, sidebar collapse, permission dialog entry use MyGo `ui` motion springs (`damping 1.0, response 0.3–0.4`); bounce (`~0.8`) only for gesture-driven collapses |
| **Spatial consistency** | Inspector slides from the right and exits to the right; permission dialog scales from its trigger row |
| **Materials & depth** | Native sidebar vibrancy (follows desktop); toolbar translucent; no stacked translucent surfaces |
| **Typography** | System font; message text 13–14px, headings tightened tracking via MyGo text styles; body leading 1.5; weight for hierarchy, not size jumps |
| **Rubber-banding** | Chat scroll view uses native scroll physics |
| **Reduced motion** | Follows desktop Reduce Motion via MyGo `ui` defaults — cross-fades instead of slides |
| **Familiarity** | Close/danger in macOS position; same look ⇒ same behavior; delete session = `ui.AlertDialog` (only genuinely destructive action gets a confirm) |
| **Feedback kinds** | Status: spinner in toolbar + session row. Completion: badge + toast. Warning/error: `ui.Toast` with action |
| **Wayfinding** | Toolbar always shows: project, model, agent, daemon state — where am I / what's live |

## Screens beyond the main window

- **Permission dialog** (`ui.Dialog`, dimmed backdrop) — tool name, args summary, Approve once / Always / Deny. Also pushed to phone via daemon; first responder wins, other side sees "answered elsewhere".
- **Settings** (separate native window or dialog) — daemon: URL, QR (rendered as `ui.Image`), token, regenerate; opencode: version, project dir, restart.
- **Toasts** — opencode crash/restart, daemon client connect/disconnect.

## What's free from MyGo (don't build)

Sidebar, Split, Markdown streaming, DiffViewer, Tree/List virtualization, Dialog/Toast, Spinner, spring motion, theme following dark mode, accessibility, view unit tests (`go test` on views).

## Out of scope v1

Multi-project, themes, keybinds, file editor (`ui.CodeEditor` exists — v2 candidate), share links.
