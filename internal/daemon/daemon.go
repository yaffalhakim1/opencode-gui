// Package daemon hosts the LAN WebSocket server phones connect to. It is
// part of the desktop app's process, sharing its opencode connection.
package daemon

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
)

// Handler runs one daemon method. Params and the result are JSON.
type Handler func(ctx context.Context, params json.RawMessage) (any, error)

// Daemon is the WebSocket server and its method table.
type Daemon struct {
	addr     string
	token    string
	handlers map[string]Handler
	mu       sync.Mutex
	conns    map[*conn]struct{}
}

type conn struct {
	ws   *websocket.Conn
	send chan []byte
}

// New creates a daemon listening on addr, authenticating with token.
func New(addr, token string) *Daemon {
	return &Daemon{
		addr:     addr,
		token:    token,
		handlers: map[string]Handler{},
		conns:    map[*conn]struct{}{},
	}
}

// Handle registers a method the clients may call.
func (d *Daemon) Handle(method string, h Handler) { d.handlers[method] = h }

// Token returns the shared secret clients must present.
func (d *Daemon) Token() string { return d.token }

// Addr returns the listen address.
func (d *Daemon) Addr() string { return d.addr }

// ListenAndServe runs the server until ctx is canceled.
func (d *Daemon) ListenAndServe(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", d.serveWS)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ok":true}`)) //nolint:errcheck
	})
	srv := &http.Server{Addr: d.addr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go func() {
		<-ctx.Done()
		shutCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		srv.Shutdown(shutCtx) //nolint:errcheck
	}()
	err := srv.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}

func (d *Daemon) serveWS(w http.ResponseWriter, r *http.Request) {
	if !d.authorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	ws, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: []string{"*"}})
	if err != nil {
		return
	}
	c := &conn{ws: ws, send: make(chan []byte, 256)}
	d.mu.Lock()
	d.conns[c] = struct{}{}
	d.mu.Unlock()

	go d.writer(c)
	d.reader(r.Context(), c)
}

func (d *Daemon) authorized(r *http.Request) bool {
	got := r.URL.Query().Get("token")
	if got == "" {
		got = strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	}
	return got != "" && subtle.ConstantTimeCompare([]byte(got), []byte(d.token)) == 1
}

// reader decodes requests and dispatches them.
func (d *Daemon) reader(ctx context.Context, c *conn) {
	defer func() {
		d.mu.Lock()
		delete(d.conns, c)
		d.mu.Unlock()
		close(c.send)
		c.ws.Close(websocket.StatusNormalClosure, "") //nolint:errcheck
	}()
	// Greet the client.
	if b, err := json.Marshal(map[string]any{"event": "daemon.hello",
		"properties": map[string]any{"version": "0.1.0"}}); err == nil {
		d.enqueue(c, b)
	}
	for {
		typ, data, err := c.ws.Read(ctx)
		if err != nil {
			return
		}
		if typ != websocket.MessageText {
			continue
		}
		var req struct {
			ID     any             `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if err := json.Unmarshal(data, &req); err != nil {
			d.enqueue(c, mustJSON(map[string]any{"id": nil,
				"error": map[string]any{"code": "bad_request", "message": err.Error()}}))
			continue
		}
		h, ok := d.handlers[req.Method]
		if !ok {
			d.enqueue(c, mustJSON(map[string]any{"id": req.ID,
				"error": map[string]any{"code": "unknown_method", "message": req.Method}}))
			continue
		}
		go func(id any, h Handler, params json.RawMessage) {
			res, err := h(ctx, params)
			if err != nil {
				d.enqueue(c, mustJSON(map[string]any{"id": id,
					"error": map[string]any{"code": "upstream_error", "message": err.Error()}}))
				return
			}
			d.enqueue(c, mustJSON(map[string]any{"id": id, "result": res}))
		}(req.ID, h, req.Params)
	}
}

func (d *Daemon) writer(c *conn) {
	for b := range c.send {
		wctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err := c.ws.Write(wctx, websocket.MessageText, b)
		cancel()
		if err != nil {
			return
		}
	}
}

// enqueue queues a message, dropping the connection when it is too far behind.
func (d *Daemon) enqueue(c *conn, b []byte) {
	select {
	case c.send <- b:
	default:
		c.ws.Close(websocket.StatusPolicyViolation, "client too slow") //nolint:errcheck
	}
}

// Broadcast sends an event to every client.
func (d *Daemon) Broadcast(event string, properties any) {
	b := mustJSON(map[string]any{"event": event, "properties": properties})
	d.mu.Lock()
	defer d.mu.Unlock()
	for c := range d.conns {
		select {
		case c.send <- b:
		default:
		}
	}
}

// Clients reports how many clients are connected.
func (d *Daemon) Clients() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.conns)
}

// NewToken makes a URL-safe random token.
func NewToken() string {
	b := make([]byte, 32)
	rand.Read(b) //nolint:errcheck
	return base64.RawURLEncoding.EncodeToString(b)
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte(`{"error":{"code":"encode","message":"cannot encode"}}`)
	}
	return b
}

// TailscaleIPv4 returns this machine's Tailscale address, or "" when
// Tailscale is not running.
func TailscaleIPv4() string {
	out, err := exec.Command("tailscale", "ip", "-4").Output()
	if err != nil {
		return ""
	}
	ip := strings.TrimSpace(strings.Split(string(out), "\n")[0])
	if strings.HasPrefix(ip, "100.") {
		return ip
	}
	return ""
}

// TailscaleName returns this machine's MagicDNS name, or "".
func TailscaleName() string {
	out, err := exec.Command("tailscale", "status", "--json").Output()
	if err != nil {
		return ""
	}
	var st struct {
		Self struct {
			DNSName string `json:"DNSName"`
		} `json:"Self"`
	}
	if json.Unmarshal(out, &st) != nil {
		return ""
	}
	return strings.TrimSuffix(st.Self.DNSName, ".")
}

// Logf logs a daemon message.
func Logf(format string, args ...any) { log.Printf(format, args...) }
