package bindata

import (
	"bytes"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var (
	errIsDirectory = errors.New("is a directory")
	errIsFile      = errors.New("is a file")
)

// dir is an in-memory implementation of vfs.FileSystem
type dir struct {
	name  string
	files map[string]*file
	dirs  map[string]*dir
}

// FileSystem interface

func (d *dir) Open(path string) (http.File, error) {
	path = strings.TrimLeft(filepath.Clean(path), string([]rune{filepath.Separator}))
	//log.Printf("bindata: open %q", path)

	switch path {
	case "", "..":
		return nil, &os.PathError{"open", path, os.ErrNotExist}
	case ".":
		return d, nil
	}

	components := strings.Split(path, string([]rune{os.PathSeparator}))
	current := d

	for i, c := range components {
		if i < len(components)-1 {
			// is a directory
			if current.dirs == nil {
				// current dir has no subdirs
				return nil, &os.PathError{"open", path, os.ErrNotExist}
			}
			if dd, ok := current.dirs[c]; ok {
				current = dd
			} else {
				// current dir has no such subdir
				return nil, &os.PathError{"open", path, os.ErrNotExist}
			}
		} else {
			// is the target file or directory
			if current.files != nil {
				if f := current.file(c); f != nil {
					return f, nil
				}
			}
			if current.dirs != nil {
				if d, ok := current.dirs[c]; ok {
					return d, nil
				}
			}
			return nil, &os.PathError{"open", path, os.ErrNotExist}
		}
	}

	return nil, os.ErrNotExist
}

func (d *dir) Walk(path string, fn filepath.WalkFunc) error {
	targetDir, err := d.Open(path)
	if err != nil {
		return err
	}
	if x, ok := targetDir.(*dir); ok {
		return x.walk(path, fn)
	}
	return &os.PathError{"walk", path, errIsFile}
}

// recursive; never returns an error
// the path argument is just a way for the full path to follow the stack
// downwards
func (d *dir) walk(path string, fn filepath.WalkFunc) error {
	fn(path, d, nil)
	if d.files != nil {
		for name := range d.files {
			fn(filepath.Join(path, name), d.file(name), nil)
		}
	}
	if d.dirs != nil {
		for name, dd := range d.dirs {
			dd.walk(filepath.Join(path, name), fn)
		}
	}
	return nil
}

// http.File interface

func (d *dir) Close() error {
	return nil
}

func (d *dir) Read(p []byte) (int, error) {
	return 0, &os.PathError{"read", d.name, errIsDirectory}
}

func (d *dir) Readdir(count int) ([]os.FileInfo, error) {
	fis := make([]os.FileInfo, 0, len(d.files)+len(d.dirs))
	for name := range d.files {
		fis = append(fis, d.file(name))
	}
	for _, dir := range d.dirs {
		fis = append(fis, dir)
	}
	return fis, nil
}

func (d *dir) Seek(offset int64, whence int) (int64, error) {
	return 0, &os.PathError{"seek", d.name, errIsDirectory}
}

func (d *dir) Stat() (os.FileInfo, error) {
	return d, nil
}

// os.FileInfo interface

func (d *dir) Name() string       { return d.name }
func (d *dir) Size() int64        { return 0 }
func (d *dir) Mode() os.FileMode  { return os.ModeDir | 0400 }
func (d *dir) ModTime() time.Time { return startupTime }
func (d *dir) IsDir() bool        { return true }
func (d *dir) Sys() interface{}   { return d }

func (d *dir) file(name string) *file {
	f, ok := d.files[name]
	if !ok {
		return nil
	}

	return &file{
		name:   f.name,
		mod:    f.mod,
		Reader: bytes.NewReader(f.data),
	}
}

type file struct {
	name string
	mod  time.Time

	// sort of like a union (either Reader when opened for reading or []byte
	// for storage)
	data []byte
	*bytes.Reader
}

// http.File interface
// Seek(int64, int) (int64, error) implemented by *bytes.Reader

func (f *file) Close() error {
	return nil
}

func (f *file) Readdir(count int) ([]os.FileInfo, error) {
	return nil, &os.PathError{"readdir", f.name, errIsFile}
}

func (f *file) Stat() (os.FileInfo, error) {
	return f, nil
}

// os.FileInfo interface
// Size() int64 is implemented in *bytes.Reader

func (f *file) Name() string       { return f.name }
func (f *file) Mode() os.FileMode  { return 0400 }
func (f *file) ModTime() time.Time { return f.mod }
func (f *file) IsDir() bool        { return false }
func (f *file) Sys() interface{}   { return f }
