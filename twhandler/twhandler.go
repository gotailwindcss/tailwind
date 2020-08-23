// Package twhandler provides an HTTP handler that performs processing on CSS files and serves them.
package twhandler

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/cespare/xxhash"
	"github.com/gotailwindcss/tailwind"
)

// New returns a Handler. TODO explain args
// The internal cache is enabled on the Handler returned.
func New(fs http.FileSystem, pathPrefix string, dist tailwind.Dist) *Handler {
	return NewFromFunc(fs, pathPrefix, func(w io.Writer) *tailwind.Converter {
		return tailwind.New(w, dist)
	})
}

// allows things like purger to be set
func NewFromFunc(fs http.FileSystem, pathPrefix string, converterFunc func(w io.Writer) *tailwind.Converter) *Handler {
	return &Handler{
		converterFunc: converterFunc,
		fs:            fs,
		pathPrefix:    pathPrefix,
		cache:         make(map[string]cacheValue),
		headerFunc:    defaultHeaderFunc,
	}
}

func defaultHeaderFunc(w http.ResponseWriter, r *http.Request) {
	cc := w.Header().Get("Cache-Control")
	if cc == "" {
		// Force browser to check each time, but 304 still works.
		w.Header().Set("Cache-Control", "no-cache")
	}
}

// // TODO: probably a shorthand that sets up the purger, etc. would make sense
// // enables purging for all default file types on dir
// func NewDev(dir, pathPrefix string, dist tailwind.Dist) {
// }

// Handler serves an HTTP response for a CSS file that is process using tailwind.
type Handler struct {
	// dist            tailwind.Dist
	converterFunc   func(w io.Writer) *tailwind.Converter
	fs              http.FileSystem
	notFound        http.Handler
	pathPrefix      string
	writeCloserFunc func(w http.ResponseWriter, r *http.Request) io.WriteCloser
	cache           map[string]cacheValue
	rwmu            sync.RWMutex
	headerFunc      func(w http.ResponseWriter, r *http.Request)
}

// SetMaxAge calls SetHeaderFunc with a function that sets the Cache-Control header (if not already set)
// with a corresponding maximum timeout specified in seconds.  If cache-breaking
// URLs are in use, this is a good option to set in production.
func (h *Handler) SetMaxAge(n int) {
	h.SetHeaderFunc(func(w http.ResponseWriter, r *http.Request) {
		cc := w.Header().Get("Cache-Control")
		if cc == "" {
			w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", n))
		}
	})
}

// SetHeaderFunc assigns a function that gets called immediately before a valid response is served.
// It was added so applications could customize cache headers.  By default, the Cache-Control
// header will be set to "no-cache" if it was not set earlier (causing the browser to check
// each time for an updated resource - which may result in a full response or a 304).
func (h *Handler) SetHeaderFunc(f func(w http.ResponseWriter, r *http.Request)) {
	h.headerFunc = f
}

// SetNotFoundHandler assigns the handler that gets called when something is not found.
func (h *Handler) SetNotFoundHandler(nfh http.Handler) {
	h.notFound = nfh
}

// SetCache with false will disable the cache.
func (h *Handler) SetCache(enabled bool) {
	if enabled {
		h.cache = make(map[string]cacheValue)
	} else {
		h.cache = nil
	}
}

// TODO: be sure to have clear example showing brotli
func (h *Handler) SetWriteCloserFunc(f func(w http.ResponseWriter, r *http.Request) io.WriteCloser) {
	h.writeCloserFunc = f
}

// not sure if we need something like this...
// // SetAllowFileFunc
// func (h *Handler) SetAllowFileFunc(f func(p string) bool) {
// }

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	p := path.Clean(r.URL.Path)
	p = path.Clean(strings.TrimPrefix(p, h.pathPrefix))

	f, err := h.fs.Open(p)
	if err != nil {
		code := 500
		if os.IsPermission(err) {
			code = 403
		} else if os.IsNotExist(err) {
			if h.notFound != nil {
				h.notFound.ServeHTTP(w, r)
				return
			}
			code = 404
		}
		http.Error(w, fmt.Sprintf("error opening %s: %v", r.URL.Path, err), code)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", "text/css")

	st, err := f.Stat()
	if err != nil {
		http.Error(w, fmt.Sprintf("stat failed for %s: %v", r.URL.Path, err), 500)
		return
	}

	if h.headerFunc != nil {
		h.headerFunc(w, r)
	}

	if h.cache != nil { // if cache enabled
		h.rwmu.RLock()
		cv, ok := h.cache[p]
		h.rwmu.RUnlock()
		if ok {

			// if h.check304(w, r, cv) {
			// 	return
			// }

			wc := h.makeW(w, r)
			defer wc.Close()

			// handle 304s properly with ServeContent
			http.ServeContent(
				&wwrap{Writer: wc, ResponseWriter: w},
				r,
				p,
				st.ModTime(),
				strings.NewReader(cv.content),
			)
			return
		}

		cv.content, cv.hash, err = h.process(w, r, f)
		if err != nil {
			http.Error(w, fmt.Sprintf("processing failed on %s: %v", r.URL.Path, err), 500)
			return
		}

		h.rwmu.Lock()
		h.cache[p] = cv
		h.rwmu.Unlock()
		return
	}

	_, _, err = h.process(w, r, f)
	if err != nil {
		http.Error(w, fmt.Sprintf("processing failed on %s: %v", r.URL.Path, err), 500)
		return
	}

	// // ck := cacheKey{
	// // 	tsnano: st.ModTime().UnixNano(),
	// // 	size:   st.Size(),
	// // 	path:   p,
	// // }

	// conv := tailwind.New(w, h.dist)
	// conv.AddReader(p, f, false)
	// err = conv.Run()
	// if err != nil {
	// 	http.Error(w, err.Error(), 500)
	// 	return
	// }
}

func (h *Handler) makeW(w http.ResponseWriter, r *http.Request) io.WriteCloser {
	var wc io.WriteCloser
	if h.writeCloserFunc != nil {
		wc = h.writeCloserFunc(w, r)
	} else {
		wc = &nopWriteCloser{Writer: w}
	}
	return wc
}

// func (h *Handler) send(w http.ResponseWriter, r *http.Request, r io.Reader) {
// 	var wc io.WriteCloser
// 	if h.writeCloserFunc != nil {
// 		wc = h.writeCloserFunc(w, r)
// 	} else {
// 		wc = &nopWriteCloser{Writer: w}
// 	}
// 	defer wc.Close()
// }

// // see if we can respond with a 304, returns true if we did
// func (h *Handler) check304(w http.ResponseWriter, r *http.Request, cv cacheValue) bool {
// 	// if t, err := time.Parse(TimeFormat, r.Header.Get("If-Modified-Since")); err == nil && modtime.Before(t.Add(1*time.Second)) {
// 	// TODO: use etag
// 	return false
// }

func (h *Handler) process(w http.ResponseWriter, r *http.Request, rd io.Reader) (content string, hash uint64, reterr error) {

	wc := h.makeW(w, r)
	defer wc.Close()

	var outbuf bytes.Buffer
	// outbuf.Grow(4096)

	d := xxhash.New()

	// write to response (optionally via compressor from makeW), cache buffer, and hash calc'er at the same time
	mw := io.MultiWriter(wc, &outbuf, d)

	conv := h.converterFunc(mw)
	// conv := tailwind.New(mw, h.dist)
	conv.AddReader(r.URL.Path, rd, false)
	err := conv.Run()
	if err != nil {
		reterr = err
		return
	}

	return outbuf.String(), d.Sum64(), nil
}

type nopWriteCloser struct {
	io.Writer
}

func (n *nopWriteCloser) Close() error {
	return nil
}

// type cacheKey struct {
// 	path   string
// 	size   int64
// 	tsnano int64
// }

type cacheValue struct {
	size    int64  // in bytes
	tsnano  int64  // file mod time
	content string // output
	hash    uint64 // for e-tag
}

// wwrap wraps a ResponseWriter allowing us to override where the Write calls go
type wwrap struct {
	io.Writer
	http.ResponseWriter
}

func (w *wwrap) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}
