package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gotailwindcss/tailwind"
	"github.com/gotailwindcss/tailwind/twembed"
	"github.com/gotailwindcss/tailwind/twpurge"
	"gopkg.in/alecthomas/kingpin.v2"
)

// NOTES:
// command line tool should provide:
// - processing similar to npx tailwindcss build
// - output as Go source code - skip for now, the workflow of CSS being served instead of embedded in a client seems to be a better way to go, let's try that first
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
	v   = app.Flag("verbose", "Print verbose output").Short('v').Bool()

	build          = app.Command("build", "Build CSS output")
	buildOutput    = build.Flag("output", "Output file name, use hyphen for stdout").Short('o').Default("-").String()
	buildPurgescan = build.Flag("purgescan", "Scan file/folder recursively for purge keys").String()
	buildPurgeext  = build.Flag("purgeext", "Comma separated list of file extensions (no periods) to scan for purge keys").Default("html,vue,jsx,vugu").String()
	buildInput     = build.Arg("input", "Input file name(s)").Strings()

	purgescan       = app.Command("purgescan", "Perform a purge scan of one or more files/dirs and output the purge keys found")
	purgescanExt    = build.Flag("ext", "Comma separated list of file extensions (no periods) to scan for purge keys").Default("html,vue,jsx,vugu").String()
	purgescanOutput = purgescan.Flag("output", "Output file name - extension can be .go, .txt or .json and determines format").Short('o').Default("-").String()
	purgescanNogen  = purgescan.Flag("nogen", "For .go output, do not emit a //go:generate line").Bool()
	purgescanInput  = purgescan.Arg("input", "Input files/dirs").Strings()

	// serve

)

var dist = twembed.New()

func main() {

	switch kingpin.MustParse(app.Parse(os.Args[1:])) {

	case build.FullCommand():

		runBuild()

	case purgescan.FullCommand():

		runPurgescan()

	default:
		fmt.Fprintf(os.Stderr, "No command specified\n")
		os.Exit(1)
	}

}

func runBuild() {

	if *v {
		log.Printf("Starting build...")
	}

	w := mkout(*buildOutput)
	defer w.Close()
	// var w io.Writer
	// outpath := *buildOutput
	// if outpath == "" || outpath == "-" {
	// 	if *v {
	// 		log.Printf("Using stdout")
	// 	}
	// 	w = os.Stdout
	// } else {
	// 	if *v {
	// 		log.Printf("Creating output file: %s", outpath)
	// 	}
	// 	f, err := os.Create(outpath)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	defer f.Close()
	// 	w = f
	// }

	conv := tailwind.New(w, dist)

	if *buildPurgescan != "" {
		if *v {
			log.Printf("Performing purge scan on: %s", *buildPurgescan)
		}

		extParts := strings.Split(*buildPurgeext, ",")
		extMap := make(map[string]bool, len(extParts))
		for _, p := range extParts {
			extMap["."+strings.TrimPrefix(p, ".")] = true
		}

		pscanner, err := twpurge.NewScannerFromDist(dist)
		if err != nil {
			log.Fatal(err)
		}

		err = filepath.Walk(*buildPurgescan, pscanner.WalkFunc(func(fn string) bool {
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

func runPurgescan() {

	if *v {
		log.Printf("Starting purge scan...")
	}

	outExt := filepath.Ext(*purgescanOutput)

	w := mkout(*purgescanOutput)
	defer w.Close()

	extParts := strings.Split(*purgescanExt, ",")
	extMap := make(map[string]bool, len(extParts))
	for _, p := range extParts {
		extMap["."+strings.TrimPrefix(p, ".")] = true
	}

	pscanner, err := twpurge.NewScannerFromDist(dist)
	if err != nil {
		log.Fatal(err)
	}

	for _, in := range *purgescanInput {
		err = filepath.Walk(in, pscanner.WalkFunc(func(fn string) bool {
			return extMap[filepath.Ext(fn)]
		}))
		if err != nil {
			log.Fatal(err)
		}
	}

	m := pscanner.Map()
	mk := mkeys(m)

	pkgName := "test123"

	switch outExt {
	case ".go":

		fmt.Fprintf(w, `package %s`+"\n", pkgName)
		fmt.Fprintf(w, "\n")
		fmt.Fprintf(w, "// WARNING: DO NOT EDIT, THIS IS A GENERATED FILE\n")
		fmt.Fprintf(w, "\n")
		if !*purgescanNogen {
			fmt.Fprintf(w, "//go:generate gotailwindcss -o %s %s\n", *purgescanOutput, strings.Join(*purgescanInput, " "))
			fmt.Fprintf(w, "\n")
		}
		fmt.Fprintf(w, "// PurgeKeyMap is a list of keys which should not be purged from CSS output.\n")
		fmt.Fprintf(w, "var PurgeKeyMap = %#v\n", (map[string]struct{})(m))
		fmt.Fprintf(w, "\n")

	case ".txt":
		for _, k := range mk {
			fmt.Fprintln(w, k)
		}

	case ".json":
		fmt.Fprintf(w, "[\n")
		for i := 0; i < len(mk); i++ {
			k := mk[i]
			if i < len(mk)-1 {
				fmt.Fprintf(w, `"%s",`, k)
			} else {
				fmt.Fprintf(w, `"%s"`, k)
			}
		}
		fmt.Fprintf(w, "]\n")

	}

}

func mkout(outpath string) io.WriteCloser {

	var ret io.WriteCloser
	if outpath == "" || outpath == "-" {
		if *v {
			log.Printf("Using stdout")
		}
		ret = nopWriteCloser{Writer: os.Stdout}
	} else {
		if *v {
			log.Printf("Creating output file: %s", outpath)
		}
		f, err := os.Create(outpath)
		if err != nil {
			log.Fatal(err)
		}
		ret = f
	}

	return ret
}

func mkeys(m map[string]struct{}) (ret []string) {
	for k := range m {
		ret = append(ret, k)
	}
	sort.Strings(ret)
	return
}

type nopWriteCloser struct {
	io.Writer
}

func (w nopWriteCloser) Close() error {
	return nil
}
