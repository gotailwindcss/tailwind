# A Tailwind CSS implementation in Go

This project provides an implementation of Tailwind CSS functionality in pure Go.  It includes the ability to embed an HTTP handler which processes Tailwind directives on the fly, facilities to purge unneeded styles, and a command line tool.

## Documentation

Godoc can be found in the usual place: https://pkg.go.dev/github.com/gotailwindcss/tailwind?tab=doc

## Typical Usage

For development, the typical use is to integrate the handler found in `twhandler` so Tailwind CSS processing is done as your CSS file is served.  Example:

**main.go**
```go
// ...
import "github.com/gotailwindcss/tailwind/twembed"
import "github.com/gotailwindcss/tailwind/twhandler"

func main() {
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir("static")))
	mux.Handle("/css/", twhandler.New(http.Dir("css"), "/css", twembed.New()))

	s := &http.Server{Addr: ":8182", Handler: mux}
	log.Fatal(s.ListenAndServe())
}
```

**static/index.html**
```html
<html>
    <head><link rel="stylesheet" href="/css/main.css"/></head>
    <body><a href="#" class="button">Test Button</a></body>
</html>
```

**css/main.css**
```css
@tailwind base;
@tailwind components;
.button { @apply inline-block m-2 p-2 rounded-md bg-green-400; }
@tailwind utilities;
```

## In Production

In production we recommend you use a simple static file server whever possible, e.g. `http.FileServer(distDir)`.

See *Procesing CSS Files* below for more info on how to create output from the command line, or *Library Usage* for how to perform Tailwind CSS conversion from Go.

## Supported Tailwind CSS Directives

The following Tailwind directives are supported:

- `@tailwind`
- `@apply`

These are intended to work with the same behavior as the [Tailwind](https://tailwindcss.com/) project.  If differences are encountered/necessary this section will be updated as applicable.

## Command Line

To install the gotailwindcss command, do:

```
go get github.com/gotailwindcss/tailwind/cmd/gotailwindcss
```

Once installed, for help:

```
gotailwindcss --help
```

### Processing CSS Files

Use the `build` subcommand to perform processing on one or more CSS files.

```
gotailwindcss build -o out.css in1.css in2.css
```

<!--
### Test Server
TODO: Create test server as part of `gotailwindcss` command line tool.
-->

## Library Usage

This project is organized into the following packages:

- **[tailwind](https://pkg.go.dev/github.com/gotailwindcss/tailwind)** - Handles CSS conversion and Tailwind processing logic
- **[twhandler](https://pkg.go.dev/github.com/gotailwindcss/tailwind/twhandler)** - HTTP Handler for processing CSS files
- **[twpurge](https://pkg.go.dev/github.com/gotailwindcss/tailwind/twpurge)** - Handles purging unused style rules
- **[twembed](https://pkg.go.dev/github.com/gotailwindcss/tailwind/twembed)** - Contains an embedded copy of Tailwind CSS styles
- **[twfiles](https://pkg.go.dev/github.com/gotailwindcss/tailwind/twfiles)** - Facilitates using a directory as source for Tailwind CSS styles

### Embedded TailwindCSS

To process "convert" files, a "Dist" (distribution) of Tailwind CSS is required.  The `twembed` package provides this.   Importing it embeds this data into your application, which is usually file for server applications.

Calling `twembed.New()` will return a new `Dist` corresponding to this embedded CSS.  It is intentionally inexpensive to call and there is no need to retain an instance as opposed ot calling `twembed.New()` again.

### Performing Conversion

A `tailwind.Convert` is used to perform processing of directives like `@tailwind` and `@apply`. Example:

```
var w bytes.Buffer
conv := tailwind.New(&w, twembed.New())
conv.AddReader("base.css", strings.NewReader(`@tailwind base;`), false)
err := conv.Run()
// w now has the processed CSS output
```

## HTTP Handler

The `twhandler` package has an HTTP handler intended to be useful during development by performing CSS processing on the fly as the file is requested.  Creating a handler is simple:

```
h := twhandler.New(
	http.Dir("/path/to/css"), // directory from which to read input CSS files
	"/css",                   // HTTP path prefix to expect
	twembed.New(),            // Tailwind distribution
)
```

From there it is used like any other `http.Handler`.

### Compression

The [SetWriteCloserFunc](https://pkg.go.dev/github.com/gotailwindcss/tailwind/twhandler?tab=doc#Handler.SetWriteCloserFunc) can be used in conjunction with [brotli.HTTPCompressor](https://pkg.go.dev/github.com/andybalholm/brotli?tab=doc#HTTPCompressor) in order to enable brotli and gzip compression.  Example:

```
h := twhandler.New(http.Dir("/path/to/css"), "/css", twembed.New())
h.SetWriteCloserFunc(brotli.HTTPCompressor)
// ...
```

### Caching

By default, caching is enabled on handlers created.  Meaning the same output will be served without re-processing as long as the underlying input CSS file's timestamp is not modified.

And by default, responses do not have a browser caching max-age, so each load results in a new request back to the server to check for a modified file.  This can be adjusted with [SetMaxAge](https://pkg.go.dev/github.com/gotailwindcss/tailwind/twhandler?tab=doc#Handler.SetMaxAge) if needed.

## Purging

TODO: write doc and example

<!--
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
-->

## See Also

This project was created as part of research while developing [Vugu](https://vugu.org/) ([doc](https://godoc.org/github.com/vugu/vugu)).

## Roadmap

- [x] Command line build tool
- [x] Pure Go library, no npm/node dependency
- [x] HTTP Handler
- [x] Purge functionality to minimize output file size
- [ ] Test server for prototyping
