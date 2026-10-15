# PAIRING — Connecting Android to the Desktop daemon

The Android app has one **Connect page**. The desktop app gives it exactly what it needs.

## What the Connect page needs

| Field | Example | Source |
|---|---|---|
| Host | `100.101.102.103` or `desktop.tail1234.ts.net` | Tailscale auto-assigns, desktop app displays it |
| Port | `9680` | Fixed daemon port |
| Token | `9f3ac1...` (44 chars, base64) | Desktop generates, displays as QR + text |

Resulting URL: `ws://<host>:9680/ws?token=<token>`

## IP research — why Tailscale IP, not LAN IP

**Tailscale assigns every device a stable IPv4 in `100.64.0.0/10`** (CGNAT range) that:
- is permanent for the life of the device in the tailnet (does not change with Wi-Fi, DHCP, reboots)
- works from any network the phone is on (home Wi-Fi, cellular, office) — traffic rides the WireGuard tunnel
- is mutually authenticated and encrypted by Tailscale itself — no TLS config needed on our side

Fallback order on the Connect page:
1. **Tailscale IP** (default, 99% case)
2. **MagicDNS name** (`machine.tailnet.ts.net`) — shown as alternate copy, useful if user recreates the tailnet
3. **Manual entry** — user types any host (LAN IP like `192.168.1.x` if both devices are on the same Wi-Fi without Tailscale and the user opted in — warn: traffic is unencrypted plaintext on LAN)

### How the desktop app detects its Tailscale IP

```go
// exec: `tailscale ip -4` → "100.101.102.103\n"
// tailscale.exe is at %ProgramFiles%\Tailscale\tailscale.exe (on PATH normally)
out, err := exec.Command("tailscale", "ip", "-4").Output()
// err != nil or empty → "Not connected to Tailscale" state in GUI
```

MagicDNS name: `tailscale status --json` → `Self.DNSName`. Latest Tailscale versions also support `tailscale whois`-style local API; shelling out is the lazy correct path — the CLI is always installed where the IP exists.

## Token lifecycle

1. **First daemon start** → `crypto/rand` 32 bytes → base64url (44 chars). Stored via `mygo.Secrets` (Windows Credential Manager). Never written to a plain config file.
2. **Display** → Settings page in the GUI shows: QR code + the URL + the raw token, with copy buttons.
3. **Rotation** → "Regenerate" button: new token, daemon drops all connected clients, Android must re-scan. Old token invalid immediately.
4. **Validation** → daemon checks `?token=` against the stored value (constant-time `subtle.ConstantTimeCompare`).

## The Connect page (Android, later)

```
┌─────────────────────────────┐
│  Scan QR          [camera]  │  ← fastest path
│  ─────────────────────────  │
│  or enter manually          │
│  Host  [100.101.102.103 ]   │  ← prefill: detect tailscale IP via
│  Port  [9680              ] │     Tailscale Android API if installed
│  Token [paste here        ] │
│                             │
│         [Connect]           │
│  ⊙ Connecting… / ✓ Online   │
└─────────────────────────────┘
```

Desktop QR payload (plain, single string — phone scanner reads it raw):

```
oc-gui://100.101.102.103:9680/ws?token=<token>
```

The app deep-link-parses this scheme; raw `ws://...` URLs also accepted.

## Threat model (why this is enough)

| Threat | Mitigated by |
|---|---|
| Random internet host scanning port | Tailscale IP unreachable outside tailnet; daemon also drops non-local-source conns when bound to tailscale IP |
| Compromised device in tailnet | Token gate + regenerate button |
| LAN sniffing (LAN fallback only) | Plaintext warning in GUI; recommend Tailscale |
| Tailscale itself | Out of scope — WireGuard mutual auth |

No TLS termination, no CA, no JWT — Tailscale already does identity + encryption. Token is a second lock, not the first.
