package opencode

import "context"

// FSEntry is one file or directory, per FileSystem.Entry.
type FSEntry struct {
	Path string `json:"path"`
	Type string `json:"type"` // "file" | "directory"
}

// ListDir lists the entries of a directory. An empty path lists the
// workspace root.
func (s *Server) ListDir(ctx context.Context, path string) ([]FSEntry, error) {
	q := "/api/fs/list"
	if path != "" {
		q += "?path=" + path
	}
	var out envelope[[]FSEntry]
	err := s.Get(ctx, q, &out)
	return out.Data, err
}

// ReadFile reads a file's text. The path goes in the URL after /fs/read/.
func (s *Server) ReadFile(ctx context.Context, path string) (string, error) {
	var out envelope[struct {
		Type    string `json:"type"`
		Content string `json:"content"`
	}]
	if err := s.Get(ctx, "/api/fs/read/"+path, &out); err != nil {
		return "", err
	}
	return out.Data.Content, nil
}
