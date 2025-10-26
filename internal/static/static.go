package static

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/gin-contrib/static"
	log "github.com/sirupsen/logrus"
)

//go:embed all:build
var site embed.FS

type embedFileSystem struct {
	http.FileSystem
	indexes bool
}

func (e embedFileSystem) Exists(prefix string, path string) bool {
	f, err := e.Open(path)
	if err != nil {
		return false
	}

	s, _ := f.Stat()
	if s.IsDir() && !e.indexes {
		return false
	}

	return true
}

func GetSiteFS() static.ServeFileSystem {
	subFS, err := fs.Sub(site, "build")
	if err != nil {
		log.Fatalf("Failed to load static site: %v", err)
	}

	return embedFileSystem{
		FileSystem: http.FS(subFS),
		indexes:    true,
	}
}
