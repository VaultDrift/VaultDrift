package web

import (
	"embed"
	"io/fs"
	"net/http"
	"path"
)

//go:embed dist
var dist embed.FS

// FS returns the embedded filesystem for the web UI
func FS() http.FileSystem {
	fsys, err := fs.Sub(dist, "dist")
	if err != nil {
		panic(err)
	}
	return http.FS(fsys)
}

// Handler returns an http.Handler for serving the web UI
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fsys := FS()

		// Try to open the requested file
		f, err := fsys.Open(path.Clean(r.URL.Path))
		if err != nil {
			// File not found, serve index.html for SPA routing
			http.ServeFile(w, r, "dist/index.html")
			return
		}
		defer f.Close()

		// Check if it's a directory
		stat, err := f.Stat()
		if err != nil || stat.IsDir() {
			http.ServeFile(w, r, "dist/index.html")
			return
		}

		// Serve the file
		http.FileServer(fsys).ServeHTTP(w, r)
	})
}
