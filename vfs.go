// Package vfs defines an interface for a virtual filesystem. It is meant as a
// wrapper around pluggable filesystem backends, e.g. on physical disk or in
// memory, with a consistent API subset.
package vfs

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
)

// FileSystem represents an abstract filesystem that can open files and walk
// directories.
type FileSystem interface {
	http.FileSystem
	Walk(root string, f filepath.WalkFunc) error
}

// nativeFS is an implementation of FileSystem that wraps the OS filesystem
type nativeFS struct {
	http.Dir
}

// Native returns a disk-backed FileSystem rooted at root. It returns an error
// if root does not exist.
func Native(root string) (FileSystem, error) {
	if _, err := os.Stat(root); err != nil {
		return nil, err
	}
	return &nativeFS{http.Dir(root)}, nil
}

func stripPrefixWalkFunc(f filepath.WalkFunc, prefix string) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		path = filepath.Clean(path)
		path, err = filepath.Rel(prefix, path)
		if err != nil {
			return err
		}
		return f(path, info, err)
	}
}

func (fs *nativeFS) Walk(root string, f filepath.WalkFunc) error {
	root = filepath.Join(string(fs.Dir), root)
	f = stripPrefixWalkFunc(f, string(fs.Dir))

	return filepath.Walk(root, f)
}

// Fallback returns a FileSystem that tries to perform operations on each given
// FileSystem in order until it succeeds.
func Fallback(fs ...FileSystem) FileSystem {
	return fallbackFS(fs)
}

// fallbackFS is a FileSystem wrapper that allows multiple filesystems to
// attempt to open a file, in ascending slice index order.
type fallbackFS []FileSystem

func (fs fallbackFS) Open(name string) (http.File, error) {
	var (
		f   http.File
		err error
	)

	for _, attempt := range fs {
		log.Printf("attempt %q in %v", name, attempt)
		f, err = attempt.Open(name)
		if err == nil {
			break
		}
	}

	return f, err
}

// Walk attempts to walk root in the filesystem list. If root is not found, the
// search will continue. If a different error was encountered, it is returned
// immediately without falling back.
func (fs fallbackFS) Walk(root string, f filepath.WalkFunc) error {
	for _, attempt := range fs {
		err := attempt.Walk(root, f)
		log.Printf("walk %q in %#v: %v", root, attempt, err)
		if err == nil {
			return nil
		}

		if os.IsNotExist(err) {
			continue
		}

		if pe, ok := err.(*os.PathError); ok {
			if os.IsNotExist(pe.Err) && pe.Path == root {
				// if we got an error opening the root dir, this means that
				// we haven't walked any files yet and can probably safely
				// try the next fallback. Otherwise, we can't safely assume
				// that we won't re-walk files. Pass the error down in that
				// case.
				continue
			}
		}

		return err
	}

	return nil
}

// Subdir returns a FileSystem that is rooted at path within fs. It does not
// check if the path exists, so errors will occur upon the first usage.
func Subdir(fs FileSystem, path string) FileSystem {
	return &subdir{fs, path}
}

type subdir struct {
	fs   FileSystem
	path string
}

func (fs *subdir) Open(name string) (http.File, error) {
	path := filepath.Join(fs.path, name)
	return fs.fs.Open(path)
}

func (fs *subdir) Walk(root string, f filepath.WalkFunc) error {
	path := filepath.Join(fs.path, root)
	return fs.fs.Walk(path, f)
}
