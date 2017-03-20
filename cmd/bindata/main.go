// to include binary data in an application, put
//
//     //go:generate bindata static templates
//
// somewhere in the application code and run
//
//     $ go generate
//
// every time the files change before building.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"go/build"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	pkgName    = "bindata_files"
	fileName   = "bindata.go"
	importFile = `package %s

import _ "%s"
`
	dataFilePrefix = `package %s

import (
	"path/filepath"
	"time"

	"ktkr.us/pkg/vfs/bindata"
)

func init() {
`
	dataFileSuffix = `}`
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("bindata: ")

	flagSkip := flag.String("skip", "", "ListSeparator-delimited list of shell patterns matching file names to be skipped")
	flag.Parse()
	if flag.NArg() == 0 {
		return
	}

	skipPatterns := filepath.SplitList(*flagSkip)

	os.MkdirAll(pkgName, 0755)

	for _, dir := range flag.Args() {
		pc := []string{pkgName, dir + ".go"}
		p := filepath.Join(pc...)
		f, err := os.OpenFile(p, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Fprintf(f, dataFilePrefix, pkgName)
		err = filepath.Walk(dir, func(p string, fi os.FileInfo, err error) error {
			if fi.IsDir() || matchList(filepath.Base(p), skipPatterns) {
				return nil
			}
			return addFile(f, p)
		})
		if err != nil {
			log.Fatal(err)
		}

		fmt.Fprintln(f, dataFileSuffix)
		f.Close()
	}

	// get the name of the current package
	wd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	pkg, err := build.ImportDir(wd, 0)
	if err != nil {
		log.Fatal(err)
	}

	f, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Fprintf(f, importFile, pkg.Name, path.Join(pkg.ImportPath, pkgName))
	f.Close()
}

func addFile(w io.Writer, p string) error {
	f, err := os.Open(p)
	if err != nil {
		return err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return err
	}

	pc := strings.Split(filepath.Clean(p), string([]rune{filepath.Separator}))
	for i, s := range pc {
		pc[i] = strconv.Quote(s)
	}
	joinExpr := fmt.Sprintf(`filepath.Join(%s)`, strings.Join(pc, ", "))

	fmt.Fprintf(w, "\tbindata.RegisterFile(%s, time.Unix(%d, 0), []byte(\"", joinExpr, fi.ModTime().Unix())
	se := &stringEncoder{bufio.NewWriter(w)}
	_, err = io.Copy(se, f)
	fmt.Fprint(w, "\"))\n")
	return err
}

type stringEncoder struct {
	w *bufio.Writer
}

// Even if a utf-8 sequence is encountered and split down the middle on a
// buffer boundary, the raw bytes will be written, no problem. It will just
// look a little silly.
func (se *stringEncoder) Write(p []byte) (int, error) {
	for _, b := range p {
		var err error
		switch b {
		case '\n':
			_, err = se.w.WriteString(`\n`)
		case '\\':
			_, err = se.w.WriteString(`\\`)
		case '"':
			_, err = se.w.WriteString(`\"`)
		default:
			if 0x20 <= b && b < 0x7F {
				err = se.w.WriteByte(b)
			} else {
				_, err = fmt.Fprintf(se.w, `\x%02x`, b)
			}
		}

		if err != nil {
			return 0, err
		}
	}

	return len(p), se.w.Flush()
}

func matchList(name string, patterns []string) bool {
	if len(patterns) == 0 {
		return false
	}
	for _, pat := range patterns {
		if m, _ := filepath.Match(pat, name); m {
			return true
		}
	}
	return false
}
