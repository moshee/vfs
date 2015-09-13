// Package vfs defines an interface for a virtual filesystem. It is meant as a
// wrapper around pluggable filesystem backends, e.g. on physical disk or in
// memory, with a consistent API subset.
package vfs

import (
	"net/http"
	"os"
	"path/filepath"
)

type FileSystem interface {
	http.FileSystem
	Walk(root string, f filepath.WalkFunc) error
}

// implementation of FileSystem that wraps the OS filesystem
type nativeFS struct {
	http.Dir
}

func NewNativeFS(root string) (FileSystem, error) {
	if _, err := os.Stat(root); err != nil {
		return nil, err
	}
	return &nativeFS{http.Dir(root)}, nil
}

func stripPrefixWalkFunc(f filepath.WalkFunc, prefix string) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		path = filepath.Clean(path)
		path, err = filepath.Rel(prefix, path)
		if err != nil {
			return err
		}
		return f(path, info, err)
	}
}

func (fs *nativeFS) Walk(root string, f filepath.WalkFunc) error {
	return filepath.Walk(filepath.Join(string(fs.Dir), root), stripPrefixWalkFunc(f, string(fs.Dir)))
}
