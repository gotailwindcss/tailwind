package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gotailwindcss/tailwind"
	"github.com/gotailwindcss/tailwind/twembed"
	"github.com/gotailwindcss/tailwind/twpurge"
	"gopkg.in/alecthomas/kingpin.v2"
)

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
// - for purging we just have code gen which scans a dir and makes the purge keys in a file, includes
//   go generate comment so it gets run again
// - option to print all allow/disallow possibilities?

var (
	app = kingpin.New("gotailwindcss", "Go+TailwindCSS tools")
	// debug    = app.Flag("debug", "Enable debug mode.").Bool()
	// serverIP = app.Flag("server", "Server address.").Default("127.0.0.1").IP()
	v = build.Flag("verbose", "Print verbose output").Short('v').Bool()

	build          = app.Command("build", "Build CSS output")
	buildOutput    = build.Flag("output", "Output file name, use hyphen for stdout").Short('o').Default("-").String()
	buildPurgeScan = build.Flag("purgescan", "Scan file/folder recursively for purge keys").String()
	buildPurgeExt  = build.Flag("purgeext", "Comma separated list of file extensions (no periods) to scan for purge keys").Default("html,vue,jsx,vugu").String()
	buildInput     = build.Arg("input", "Input file name(s)").Strings()
	// register     = app.Command("register", "Register a new user.")
	// registerNick = register.Arg("nick", "Nickname for user.").Required().String()
	// registerName = register.Arg("name", "Name of user.").Required().String()

	// post        = app.Command("post", "Post a message to a channel.")
	// postImage   = post.Flag("image", "Image to post.").File()
	// postChannel = post.Arg("channel", "Channel to post to.").Required().String()
	// postText    = post.Arg("text", "Text to post.").Strings()
)

var dist = twembed.New()

func main() {

	// FIXME: what about sequence?  if this doesn't work we should rethink: gotailwindcss in.css -o out.css
	// (might not because i think flags must precede args)

	// fo := flag.String("o", "", "Output file to write, if not specified stdout is used")
	// foutput := flag.String("output", "", "Alias for -o")
	// fv := flag.String("v", "", "Verbose logging (via stderr)")

	// flag.Parse()

	// args := flag.Args()

	// app.Parse()
	// log.Println(os.Args)

	switch kingpin.MustParse(app.Parse(os.Args[1:])) {

	case build.FullCommand():

		runBuild()

	// // Post message
	// case post.FullCommand():
	// 	if *postImage != nil {
	// 	}
	// 	text := strings.Join(*postText, " ")
	// 	println("Post:", text)
	default:
		fmt.Fprintf(os.Stderr, "No command specified\n")
		os.Exit(1)
	}

}

func runBuild() {

	if *v {
		log.Printf("Starting build...")
	}

	var w io.Writer
	outpath := *buildOutput
	if outpath == "" || outpath == "-" {
		if *v {
			log.Printf("Using stdout")
		}
		w = os.Stdout
	} else {
		if *v {
			log.Printf("Creating output file: %s", outpath)
		}
		f, err := os.Create(outpath)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		w = f
	}

	conv := tailwind.New(w, dist)

	if *buildPurgeScan != "" {
		if *v {
			log.Printf("Performing purge scan on: %s", *buildPurgeScan)
		}

		extParts := strings.Split(*buildPurgeExt, ",")
		extMap := make(map[string]bool, len(extParts))
		for _, p := range extParts {
			extMap["."+strings.TrimPrefix(p, ".")] = true
		}

		pscanner, err := twpurge.NewScannerFromDist(dist)
		if err != nil {
			log.Fatal(err)
		}

		err = filepath.Walk(*buildPurgeScan, pscanner.WalkFunc(func(fn string) bool {
			return extMap[filepath.Ext(fn)]
		}))
		if err != nil {
			log.Fatal(err)
		}

		conv.SetPurgeChecker(pscanner.Map())
	}

	for _, inPath := range *buildInput {
		if *v {
			log.Printf("Adding file: %s", inPath)
		}
		fin, err := os.Open(inPath)
		if err != nil {
			log.Fatal(err)
		}
		defer fin.Close()
		conv.AddReader(inPath, fin, false)
	}

	if *v {
		log.Printf("Performing conversion...")
	}

	err := conv.Run()
	if err != nil {
		log.Fatal(err)
	}

}
