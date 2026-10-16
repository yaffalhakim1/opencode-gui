// Command opencode-gui is a native desktop GUI for the OpenCode server, with a
// WebSocket daemon so a phone can drive the same sessions over Tailscale.
//
// Everything lives in the src package; this file is only the entry point, so
// the root directory holds no logic of its own.
package main

import "opencode-gui/src"

func main() {
	src.Run()
}
