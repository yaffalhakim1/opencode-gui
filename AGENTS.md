# opencode-gui — development guidance

> **UI must match Kerenzikov: `C:\Users\yafit\Documents\Learn\rust\waku`**
>
> That checkout is the visual and behavioural source of truth. Recheck it
> whenever a question about layout, spacing, type, colour, copy or component
> choice comes up — **never assume**. Read the Rust source (`src/app/*.rs`,
> `crates/waku-protocol/src/*.rs`, `locales/app.yml`) and take the real values.
>
> Recheck the MyGo docs (https://mygo.egoist.dev/docs/ui) before building any
> component by hand: the framework ships most of them, and a hand-rolled
> equivalent loses keyboard, focus, and accessibility behaviour.

---

## Development runtime

- Assume `go tool mygo dev` may already be running and owns the app process.
  Source changes rebuild and relaunch automatically. Only start it yourself if
  it is confirmed unavailable.
- Validate the freshly relaunched app after an edit; do not start a second
  watcher.
- For layout or visual changes, inspect the rendered window — do not infer the
  result from source alone.

### Never kill an `opencode` process

- **The coding agent is often a child of the app it is working on.** An
  OpenCode session runs as `opencode.exe` (and the shared background service),
  so the daemon is not a bystander: killing it kills the session that issued
  the command, mid-tool-call, and the agent resumes with partial context and no
  memory of its own recent work.
- Never `Stop-Process`/`taskkill`/`kill` an `opencode` process. If one is in
  the way, report it and stop; do not clear it yourself.
- Before touching any `opencode*` process, answer all three, with evidence:
  1. **Is it my ancestor?** Walk `ParentProcessId`
     (`Get-CimInstance Win32_Process`) or `ppid`. If the agent process is a
     descendant, it is the host — never kill it.
  2. **Is it the shared service?** `opencode service status` names the
     registered server. That one owns the user's sessions; it is not
     disposable.
  3. **Is it my build?** A process older than the binary you just built is not
     your artifact — that alone rules out killing it.
- A held port is never a reason to kill. Use `opencode service status`, or
  launch against a different port.
- The app under development (`OpenCode GUI.exe`) **is** safe to stop: it is
  this project's own artifact and holds no agent session. That is the only
  process here you may stop freely.

---

## What this project is

A native desktop GUI for [OpenCode](https://opencode.ai) built with
[MyGo](https://mygo.egoist.dev) (`github.com/egoist/mygo`), plus a WebSocket
daemon so a phone can drive the same sessions over Tailscale.

- **Desktop**: pure Go native UI (`package ui`), drawn on the GPU, no webview.
- **Daemon**: `internal/daemon`, WS on `:9680`, token auth, tailnet-reachable.
- One binary serves both. See `docs/ARCHITECTURE.md`.

## Layout of the code

| Path | Responsibility |
|---|---|
| `main.go` | app state, the view tree, the OpenCode event loop |
| `composer.go` | the prompt card: model/agent/access menus, queue, attachments |
| `panel.go` | the right pane: Files / Review / Browser / Terminal surfaces |
| `settings.go` | the Settings page and its seven sections |
| `theme.go` | the Kerenzikov palette mapped onto `ui.Theme`, shared styling |
| `layout.go` | pure panel-fitting maths (ported from the reference) |
| `daemon_wire.go` | daemon methods, from `docs/PROTOCOL.md` |
| `internal/opencode` | HTTP client of the OpenCode server + SSE |
| `internal/app` | the service layer the view calls |
| `internal/daemon` | the WS server for phones |

## Rules

### Check the docs first, never assume

- **MyGo**: https://mygo.egoist.dev/docs/ui — read the page for a component
  before using it, and check `go doc github.com/egoist/mygo/ui <Name>` for the
  exact signature of the pinned version. The published docs may describe a
  newer release than `go.mod` pins.
- **OpenCode**: https://opencode.ai/v2/docs/api and the running server's own
  spec (`opencode api get /api/info`, or `/openapi.json`). The v2 API differs
  from v1 in ways that break naive clients — see `docs/OPENCODE.md` for the
  verified facts (envelopes, epoch-millisecond ranges, async prompts).
- **Kerenzikov**: the checkout above, for anything visual.

### Use framework components

MyGo ships 95 UI constructors. Before writing a widget, check the list. Hand-rolled
replacements lose focus handling, keyboard navigation, and accessibility.

Components already in use, and the guards that keep them there:

| Need | Use |
|---|---|
| scrollable/virtualized rows | `ui.List`, `ui.Table` |
| search box | `ui.SearchField` |
| type-to-filter choice | `ui.Combobox`, `ui.Select` |
| dropdown menu | `ui.MenuButton` |
| dialog over the window | `ui.Modal`, `ui.AlertDialog` |
| tab strip | `ui.Tabs` |
| chips / token input | `ui.TokenField` |
| top bar | `ui.Toolbar` |
| on/off setting | `ui.Switch` — **always with `.Label()`** |
| short message | `c.Toast`, `c.ToastAction` |
| diffs | `ui/editor.DiffView` with `HunkActions`, `Stats()`, `GoToHunk()` |
| charts | `ui/chart` |
| markdown | `ui/markdown` |

Tests in `settings_test.go` fail if a switch loses its label or a framework
component is replaced by hand-rolled rows. Do not delete them to make a change
pass.

`ui.Form`/`ui.Field` puts labels in their own right-aligned column (macOS
style), which is **not** the reference's shape — the settings rows are a plain
`Row` deliberately. There is a comment in `settings.go` saying so.

### Match the reference, don't guess

Take measurements from the Rust source, and record them as named constants with
the source in a comment, as `theme.go` and `layout.go` do:

- sidebar 252 (min 180, max 420), right panel 460 (min 280)
- chat and composer share a centred 720-wide column
- settings content 760 wide
- labels 13.5px weight 500, descriptions 12.5px, page titles 18px weight 500
- cards radius 13; hovered/selected rows a 6% neutral layer, not an opaque fill

Never `Bold()` (700) for UI labels — the reference uses 500 throughout.

### Views, state, and threading

- The view is a pure function of `state`. Change state from goroutines with
  `win.Update(func(){...})`, which runs on the main thread and redraws.
- Never store a `ui.Element` in state: elements expire each build pass. Store
  model data, `ui.ListState`, and `ui.Handle`.
- Keep I/O off the UI thread; publish results through `win.Update`.
- Guard `state` with its mutex: the SSE loop, daemon handlers, and the view all
  touch it.

## Verification

```powershell
go build ./...            # compiles
go tool mygo vet .        # Go vet + MyGo's native UI build-lifetime checks
go test ./...             # 59 tests: layout, ui, settings, daemon, opencode
go tool mygo build        # packaged app in build/windows-amd64/
```

`go tool mygo vet` is the one that catches a `ui.Element` stored in state or
captured in a goroutine; plain `go vet` does not.

- `ui.NewTester(view, w, h)` renders a view without a window: use it for
  layout and interaction assertions (`tt.Click`, `tt.HasText`, `tt.Texts`).
- For anything visual, launch the app and inspect the window. Screenshots of
  the reference live in `docs/`.
- Report what was verified and what was not. Do not claim a visual match
  without having looked at it.

## Docs in this repo

`docs/ARCHITECTURE.md`, `ROADMAP.md`, `PROTOCOL.md`, `PAIRING.md`, `DAEMON.md`,
`OPENCODE.md`, `GUI.md`. Keep `OPENCODE.md` current when API behaviour is
learned the hard way — it exists so the next session does not repeat the
mistake.
