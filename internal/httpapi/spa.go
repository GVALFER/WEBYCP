package httpapi

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type spa struct {
	root  string
	files http.Handler
}

func newSPA(root string) http.Handler {
	return &spa{
		root:  root,
		files: http.FileServer(http.Dir(root)),
	}
}

func (s *spa) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	requestPath := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
	if requestPath == "" {
		requestPath = "index.html"
	}

	target := filepath.Join(s.root, filepath.FromSlash(requestPath))
	if info, err := os.Stat(target); err == nil && !info.IsDir() {
		s.files.ServeHTTP(w, r)
		return
	}

	http.ServeFile(w, r, filepath.Join(s.root, "index.html"))
}
