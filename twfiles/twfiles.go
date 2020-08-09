// Package twfiles implements tailwind.Dist against a filesystem.
// Implementations are provided against the OS filesystem and for net/http.FileSystem.
package twfiles

import (
	"io"
	"net/http"
)

// New returns a tailwind.Dist instance that reads from the underlying OS directory you provide.
// Implementation is done via net/http Filesystem.
func New(baseDir string) *HTTPFiles {
	return &HTTPFiles{
		FileSystem: http.Dir(baseDir),
	}
}

// NewHTTP returns an HTTP file instance which reads from the underlying net/http.Filesystem.
// Files are mapped by default so e.g. requests for "base" look for "base.css".
func NewHTTP(fs http.FileSystem) *HTTPFiles {
	return &HTTPFiles{
		FileSystem: fs,
	}
}

// HTTPFiles implements tailwind.Dist against a net/http.FileSystem.
// By default the file name mapings are the name of the tailwind section plus
// ".css", e.g. "base.css", "utilities.css", "components.css".
type HTTPFiles struct {
	http.FileSystem                          // underlying http FileSystem
	NameMapFunc     func(name string) string // name conversion func, default returns name+".css"
}

// OpenDist implements tailwind.Dist.
func (hf *HTTPFiles) OpenDist(name string) (io.ReadCloser, error) {

	var fileName string
	if hf.NameMapFunc != nil {
		fileName = hf.NameMapFunc(name)
	} else {
		fileName = name + ".css"
	}

	f, err := hf.FileSystem.Open(fileName)
	if err != nil {
		return nil, err
	}

	return f, nil
}
