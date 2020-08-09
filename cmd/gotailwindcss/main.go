package main

import "flag"

// command line tool should provide:
// - processing similar to npx tailwindcss build
// - output as Go source code
//   - plus internal gzipped format?
// - probably output file auto-detection can work,
//   if the output file is .go then it emits go src,
//   or if .css emit CSS
// - follow command line format of npx tailwindcss build unless there is a reason not to (i.e. -o for output)
// - should we provide some sort of minimal static server?
//   not super useful but easy to do and maybe people would find use for demos
// - allow and disallow - we need to implement "purge" instead
// - option to print all allow/disallow possibilities?

func main() {

	// FIXME: what about sequence?  if this doesn't work we should rethink: gotailwindcss in.css -o out.css
	// (might not because i think flags must precede args)

	fo := flag.String("o", "", "Output file to write, if not specified stdout is used")
	foutput := flag.String("output", "", "Alias for -o")
	fv := flag.String("v", "", "Verbose logging (via stderr)")

	flag.Parse()

	args := flag.Args()

}
