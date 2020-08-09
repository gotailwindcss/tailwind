package tailwind

import "io"

// Dist is where tailwind CSS data can be read.
type Dist interface {
	// OpenDist should return a new ReadCloser for the specific tailwind section name.
	// Valid names are "base", "utilities" and "components" (only those exact strings,
	// without .css or anything like that) and will be updated along with
	// what the TailwindCSS project does.  The caller is responsible for ensuring
	// Close() is called on the response if the error is non-nil.
	OpenDist(name string) (io.ReadCloser, error)
}

// // DefaultDist is the Dist used by default by New().
// // Importing the twembed package sets this.
// var DefaultDist Dist = nil
