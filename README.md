# A Tailwind CSS implementation in Go

Features/Roadmap:

- [x] Command line build tool
- [x] Pure Go library, no npm/node dependency
- [x] HTTP Handler
- [x] Purge functionality to minimize output file size
- [ ] Test server for prototyping

## Documentation

https://pkg.go.dev/github.com/gotailwindcss/tailwind/

## Typical Usage

(show small standalone Go program with HTTP handler, compression and purging)

### static/index.html
```html
<!doctype html>
<html>
    <head>
        <link rel="stylesheet" href="/css/main.css"/>
    </head>
    <body>
        <a href="#" class="button">Test Button</a>
    </body>
</html>
```

### css/main.css
```css
@tailwind base;
@tailwind components;

.button {
    @apply inline-block m-2 p-2 rounded-md text-lg leading-tight font-semibold text-gray-900 bg-green-400;
}

@tailwind utilities;
```

### main.go
```go
package main

import (
	"io"
	"log"
	"net/http"

	"github.com/gotailwindcss/tailwind"
	"github.com/gotailwindcss/tailwind/twembed"
	"github.com/gotailwindcss/tailwind/twhandler"
)

func main() {

	staticDir := http.Dir("static")

	indexH := http.FileServer(staticDir)

	tailwindH := twhandler.NewFromFunc(http.Dir("css"), "/css", func(w io.Writer) *tailwind.Converter {
		ret := tailwind.New(w, twembed.New())
		return ret
	})

	mux := http.NewServeMux()
	mux.Handle("/", indexH)
	mux.Handle("/css/", tailwindH)

	s := &http.Server{Addr: ":8182", Handler: mux}
	log.Fatal(s.ListenAndServe())

}
```

## In Production

In production we recommend you use a simple static file server whever possible, e.g. `http.FileServer(distDir)`.

See *Procesing CSS Files* below for more info on how to create output from the command line, or *Library Usage* for how to perform Tailwind CSS conversion from Go.

## Supported Tailwind CSS Directives

- `@tailwind`
- `@apply`

## Command Line

### Processing CSS Files

### Test Server

TODO: Create test server as part of `gotailwindcss` command line tool.

## Library Usage

### Embedded TailwindCSS

## HTTP Handler

### Compression

### Caching

## Purging

(reduce file size)

### Standalone Example

(less work to setup and maintain but runs slower)

```go
package main

import (
	"io"
	"log"
	"net/http"
	"path/filepath"

	"github.com/gotailwindcss/tailwind"
	"github.com/gotailwindcss/tailwind/twembed"
	"github.com/gotailwindcss/tailwind/twhandler"
	"github.com/gotailwindcss/tailwind/twpurge"
)

func main() {

	staticDir := http.Dir("static")

	indexH := http.FileServer(staticDir)

	pscanner, err := twpurge.NewScannerFromDist(twembed.New())
	if err != nil {
		panic(err)
	}
	err = filepath.Walk("static", pscanner.WalkFunc(twpurge.MatchDefault))
	if err != nil {
		panic(err)
	}

	tailwindH := twhandler.NewFromFunc(http.Dir("css"), "/css", func(w io.Writer) *tailwind.Converter {
		ret := tailwind.New(w, twembed.New())
		ret.SetPurgeChecker(pscanner.Map())
		return ret
	})

	mux := http.NewServeMux()
	mux.Handle("/", indexH)
	mux.Handle("/css/", tailwindH)

	s := &http.Server{Addr: ":8182", Handler: mux}
	log.Fatal(s.ListenAndServe())

}
```

### Purge Scan at Code-Generation-Time

(a bit more work to setup and use, but more efficient and gives the same results in dev and production)

## Embedding in Go Code
