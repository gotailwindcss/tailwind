package tailwind_test

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/gotailwindcss/tailwind"
	"github.com/gotailwindcss/tailwind/twembed"
	"github.com/tdewolff/minify/v2"
	"github.com/tdewolff/minify/v2/css"
)

// type mapFS map[string]string

// // Open implements an http.FileSystem as a map of names and contents.
// func (m mapFS) Open(name string) (http.File, error) {
// 	b, ok := m[name]
// 	rd := bytes.NewReader()
// }

// type mapFSFile struct {
// 	bytes.Reader
// }

// func (mf *mapFSFile) Readdir(count int) ([]os.FileInfo, error) {
// 	panic("not implemented")
// }
// func (mf *mapFSFile) Stat() (os.FileInfo, error) {
// 	panic("not implemented")
// }

func ExampleConverter_SetPostProcFunc() {

	var buf bytes.Buffer
	conv := tailwind.New(&buf, twembed.New())
	conv.SetPostProcFunc(func(out io.Writer, in io.Reader) error {
		m := minify.New()
		m.AddFunc("text/css", css.Minify)
		return m.Minify("text/css", out, in)
	})
	conv.AddReader("input.css", strings.NewReader(`.test1 { @apply font-bold; }`), false)
	err := conv.Run()
	if err != nil {
		panic(err)
	}
	fmt.Printf("%s", buf.String())

	// notice the missing trailing semicolon

	// Output: .test1{font-weight:700}
}

func TestConverter(t *testing.T) {

	type tcase struct {
		name   string            // test case name
		in     map[string]string // input files (processed alphabetical by filename)
		out    []*regexp.Regexp  // output must match these regexps
		outerr *regexp.Regexp    // must result in an error with text that matches this (if non-nil)
	}

	tcaseList := []tcase{
		{
			name: "simple1",
			in: map[string]string{
				"001.css": `.test1 { display: block; }`,
			},
			out: []*regexp.Regexp{
				regexp.MustCompile(`^` + regexp.QuoteMeta(`.test1{display:block;}`) + `$`),
			},
		},
		{
			name: "two-files1",
			in: map[string]string{
				"001.css": `.test1 { display: block; }`,
				"002.css": `.test2 { display: inline; }`,
			},
			out: []*regexp.Regexp{
				regexp.MustCompile(`^` + regexp.QuoteMeta(`.test1{display:block;}.test2{display:inline;}`) + `$`),
			},
		},
		{
			name: "two-files2", // verify the sequence is correct
			in: map[string]string{
				"012.css": `.test2 { display: inline; }`,
				"021.css": `.test1 { display: block; }`,
			},
			out: []*regexp.Regexp{
				regexp.MustCompile(`^` + regexp.QuoteMeta(`.test2{display:inline;}.test1{display:block;}`) + `$`),
			},
		},
		{
			name: "bad1",
			in: map[string]string{
				"001.css": `.test1 { display: block; ! }`,
			},
			outerr: regexp.MustCompile(`expected colon in declaration`),
		},
		{
			name: "atrule1",
			in: map[string]string{
				"001.css": `@charset "utf-8"; .test1 { display: block; }`,
			},
			out: []*regexp.Regexp{
				regexp.MustCompile(`^` + regexp.QuoteMeta(`@charset "utf-8";.test1{display:block;}`) + `$`),
			},
		},
		{
			name: "tailwind-base1",
			in: map[string]string{
				"001.css": `@tailwind base;`,
			},
			out: []*regexp.Regexp{
				regexp.MustCompile(regexp.QuoteMeta(`html{line-height:1.15;`)),
				regexp.MustCompile(regexp.QuoteMeta(`b,strong{`)), // ensure QualifiedRuleGrammar is working
			},
		},
		{
			name: "tailwind-components1",
			in: map[string]string{
				"001.css": `@tailwind components;`,
			},
			out: []*regexp.Regexp{
				regexp.MustCompile(regexp.QuoteMeta(`.container{width:100%`)),
			},
		},
		{
			name: "tailwind-utilities1",
			in: map[string]string{
				"001.css": `@tailwind utilities;`,
			},
			out: []*regexp.Regexp{
				regexp.MustCompile(regexp.QuoteMeta(`var(--bg-opacity)`)),
			},
		},
		{
			name: "tailwind-utilities2",
			in: map[string]string{
				"001.css": `@tailwind utilities;`,
			},
			out: []*regexp.Regexp{
				regexp.MustCompile(regexp.QuoteMeta(`--bg-opacity: 1`)),
			},
		},
		{
			name: "tailwind-unknown1",
			in: map[string]string{
				"001.css": `@tailwind otherthing;`,
			},
			outerr: regexp.MustCompile(regexp.QuoteMeta(`@tailwind followed by unknown identifier: otherthing`)),
		},
		{
			name: "apply1",
			in: map[string]string{
				"001.css": `.test { @apply px-1; }`,
			},
			out: []*regexp.Regexp{
				regexp.MustCompile(regexp.QuoteMeta(`.test{padding-left:0.25rem;padding-right:0.25rem;}`)),
			},
		},
		{
			name: "apply2",
			in: map[string]string{
				"001.css": `.test { @apply px-1 py-2; }`,
			},
			out: []*regexp.Regexp{
				regexp.MustCompile(regexp.QuoteMeta(`.test{padding-left:0.25rem;padding-right:0.25rem;padding-top:0.5rem;padding-bottom:0.5rem;}`)),
			},
		},
	}

	for _, tc := range tcaseList {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			c := tailwind.New(&buf, twembed.New())
			klist := make([]string, len(tc.in))
			for k := range tc.in {
				klist = append(klist, k)
			}
			sort.Strings(klist)
			for _, k := range klist {
				c.AddReader(k, strings.NewReader(tc.in[k]), !strings.HasSuffix(k, ".css"))
			}
			err := c.Run()
			if err != nil { // got error
				if tc.outerr != nil { // expected error
					if !tc.outerr.MatchString(err.Error()) {
						t.Errorf("error failed to match regexp: %s - %v", tc.outerr.String(), err.Error())
					}
				} else {
					t.Fatal(err) // error not expected
				}
			}
			bufstr := buf.String()
			for _, outre := range tc.out {
				if !outre.MatchString(bufstr) {
					t.Errorf("output failed to match regexp: %s", outre.String())
				}
			}
			if t.Failed() {
				t.Logf("OUTPUT: (err=%v)\n%s", err, bufstr)
			}
		})
	}

}
