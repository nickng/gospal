package build

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"log"
	"os"

	"github.com/nickng/gospal/ssa"
	"github.com/pkg/errors"
)

// Builder builds SSA IR and metainfo.
type Builder interface {
	Build() (*ssa.Info, error)
}

// FileSrc is a set of filenames.
type FileSrc struct {
	Files []string
}

// FromFiles returns a non-nil Builder from a slice of filenames.
func FromFiles(files []string) Configurer {
	return newConfig(&FileSrc{Files: files})
}

// File returns an io.Reader for file[i].
func (s *FileSrc) Reader(i int) io.Reader {
	if i < len(s.Files) {
		f, err := os.Open(s.Files[i])
		defer f.Close()
		if err != nil {
			log.Fatal(errors.Wrapf(err, "failed to read from file: %s", s.Files[i]))
		}
		return bufio.NewReader(f)
	}
	return nil
}

// NewReader returns an io.Reader for reading all files.
func (s *FileSrc) NewReader() io.Reader {
	var rds []io.Reader
	for i := range s.Files {
		rds = append(rds, s.Reader(i))
	}
	return io.MultiReader(rds...)
}

// CachedSrc is source file from a reader.
type CachedSrc struct {
	cached []byte
}

// FromReader returns a non-nil Builder for a reader.
// This is typically used for testing or building a temporary file.
func FromReader(r io.Reader) Configurer {
	b, err := ioutil.ReadAll(r)
	if err != nil {
		log.Fatal(errors.Wrap(err, "failed to read from reader"))
	}
	return newConfig(&CachedSrc{cached: b})
}

// NewReader returns a reader for reading the string content.
func (s *CachedSrc) NewReader() io.Reader {
	return bytes.NewReader(s.cached)
}
