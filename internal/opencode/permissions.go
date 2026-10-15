package opencode

import "context"

// PermissionRequest is one tool call waiting for the user's answer, per
// Permission.Request.
type PermissionRequest struct {
	ID        string   `json:"id"`
	SessionID string   `json:"sessionID"`
	Action    string   `json:"action"`
	Resources []string `json:"resources"`
	Source    string   `json:"source"`
	Message   string   `json:"message"`
}

// PermissionDecision is the answer to a permission request.
type PermissionDecision string

const (
	PermissionOnce   PermissionDecision = "once"
	PermissionAlways PermissionDecision = "always"
	PermissionReject PermissionDecision = "reject"
)

// Permissions lists the requests waiting for an answer.
func (s *Server) Permissions(ctx context.Context) ([]PermissionRequest, error) {
	var out envelope[[]PermissionRequest]
	err := s.Get(ctx, "/api/permission/request", &out)
	return out.Data, err
}

// ReplyPermission answers a pending request.
func (s *Server) ReplyPermission(ctx context.Context, sessionID, requestID string, d PermissionDecision) error {
	var out any
	body := map[string]any{"decision": string(d)}
	return s.Post(ctx, "/api/session/"+sessionID+"/permission/"+requestID+"/reply", body, &out)
}
