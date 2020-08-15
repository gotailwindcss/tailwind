package twpurge

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/tdewolff/parse/v2/css"
)

// TODO: PurgeDir - reads directory, can reload upon demand (first call to ShouldPurgeKey) after X time, by default only loads first time
// Should we/can we abstract this out to some sort of reload function that gets invoked?  Or is that too much, maybe we just go simple.

// type Purger struct {
// }

// // ShouldInclude implements ...
// // The name var has a class like "..." (decide what the rules are here - what about these crazy colons and stuff)
// func (p *Purger) ShouldInclude(name string) bool {
// 	panic(errors.New("not yet implemented"))
// }

// // NameParser extracts possible names of things to allow
// type NameParser struct {
// }

// PurgeKeyParser is anything that can read a stream (containing usually HTML or HTML-like layout content)
// and parse purge keys from it.
//
// A "purge key" is an identifier like "px-1", it can contain layout other contstraints like "sm:px-1", "md:w-full",
// or "sm:focus:placeholder-green-200".  It does not have a period prefix, no backslashes should appear,
// and should not contain any colon suffixes (prefixes shown before are correct, but things like :focus
// at the end, etc.)
//
// The purgeKeyMap map, if not-nil, provides a list of all possible purge keys, which can be used to
// discard keys found that aren't in the map.
// type PurgeKeyParser interface {
// 	ParsePurgeKeys(r io.Reader, purgeKeyMap map[string]struct{}) error
// }

// read a single file
// read a tree
// read a tree every X seconds
// each needs to provide a ShouldPurgeKey(k string) bool

// MatchDefault is a filename matcher function which will return true for files
// end in .html, .vugu, .jsx or .vue.
var MatchDefault = func(fn string) bool {
	ext := strings.ToLower(filepath.Ext(fn))
	switch ext {
	case ".html", ".vugu", ".jsx", ".vue":
		return true
	}
	return false
}

// Purger can parse markup and accumulate a list of purge keys which can be used to
// vet the output of tailwind.Converter to eliminate unused styles.
type Purger struct {
	purgeKeyMap  map[string]struct{} // all possible purge keys, passed in from New
	parsedKeyMap map[string]struct{} // the keys parsed from the markup (filtered to include only purgeKeyMap entries if not nil), these are the keys to be kept during conversion
	tokenizer    Tokenizer
}

// TODO:
// New(Dist)
// NewFromMap
// BuildPurgeKeyMap(Dist) map
// and then twfiles does not have purge support, but twembed can have it all preprocessed
// New(Dist) looks for interface and calls BuildPurgeKeyMap if not implemeneted

// MustNew is like New but panics upon error.
func MustNew(dist Dist) *Purger {
	ret, err := New(dist)
	if err != nil {
		panic(err)
	}
	return ret
}

// New returns a new Purger instance.  Uses PurgeKeysFromDist.
func New(dist Dist) (*Purger, error) {

	// moved to PurgeKeysFromDist
	// pkmr, ok := dist.(purgeKeyMapper)
	// if ok {
	// 	return NewFromMap(pkmr.PurgeKeyMap()), nil
	// }

	pkmap, err := PurgeKeysFromDist(dist)
	if err != nil {
		return nil, err
	}
	return NewFromMap(pkmap), nil
}

type purgeKeyMapper interface {
	PurgeKeyMap() map[string]struct{}
}

// NewFromMap returns a new Purger instance. If purgeKeyMap is not nil, it is a map
// of all the possible keys that can be purged, which is then used during
// markup parsing to be able to scan for just the purge keys that are relevant.
// Passing nil will still result in proper function but will use more memory
// and potentially be slower.
func NewFromMap(purgeKeyMap map[string]struct{}) *Purger {
	return &Purger{purgeKeyMap: purgeKeyMap, parsedKeyMap: make(map[string]struct{})}
}

// WalkFunc returns a function which can be called by filepath.Walk
func (p *Purger) WalkFunc(fnmatch func(fn string) bool) filepath.WalkFunc {
	if fnmatch == nil {
		fnmatch = MatchDefault
	}
	return filepath.WalkFunc(func(fpath string, info os.FileInfo, err error) error {
		if info.IsDir() { // ignore dirs
			return nil
		}
		if err != nil { // any stat errors get returned as-is
			return err
		}
		if !fnmatch(fpath) { // ignore if filename doesn't match
			return nil
		}
		return p.ParseFile(fpath)
	})
}

func (p *Purger) SetTokenizer(t Tokenizer) {
	p.tokenizer = t
}

func (p *Purger) ParseReader(r io.Reader) error {
	if p.parsedKeyMap == nil {
		p.parsedKeyMap = make(map[string]struct{})
	}

	t := NewDefaultTokenizer(r)
	for {
		b, err := t.NextToken()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		bstr := string(b) // FIXME: cheating with zero-alloc unsafe cast would be appropriate here
		found := true
		if p.purgeKeyMap != nil {
			_, found = p.purgeKeyMap[bstr]
		}
		if found {
			p.parsedKeyMap[bstr] = struct{}{}
		}
	}

}

func (p *Purger) ParseFile(fpath string) error {
	f, err := os.Open(fpath)
	if err != nil {
		return err
	}
	defer f.Close()
	return p.ParseReader(f)
}

func (p *Purger) ShouldPurgeKey(k string) bool {
	_, ok := p.parsedKeyMap[k]
	return !ok
}

// Tokenizer returns the next token from a markup file.
type Tokenizer interface {
	NextToken() ([]byte, error) // returns a token or error (not both), io.EOF indicates end of stream
}

func isbr(c byte) bool {
	switch c {
	// NOTE: We're going to assume ASCII is fine here - we could do some UTF-8 fanciness but I don't know
	// of any situation where it would matter for our purposes here.
	case '<', '>', '"', '\'', '`',
		'\t', '\n', '\v', '\f', '\r', ' ':
		return true
	}
	return false
}

func NewDefaultTokenizer(r io.Reader) *DefaultTokenizer {
	s := bufio.NewScanner(r)
	s.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {

		// log.Printf("Split(data=%q, atEOF=%v)", data, atEOF)
		// defer func() {
		// 	log.Printf("Split(data=%q, atEOF=%v) returning (advance=%d, token=%q, err=%v)", data, atEOF, advance, token, err)
		// }()

		// consume any break text
		for len(data) > 0 {
			if !isbr(data[0]) {
				break
			}
			data = data[1:]
			advance++
		}

		// now read thorugh any non-break text
		var i int
		for i = 0; i < len(data); i++ {

			if isbr(data[i]) {
				// if we encounter a break, then return what we've read so far as the token
				if i > 0 {
					token = data[:i]
				}
				advance += i
				return
			}

			// otherwise just continue
		}

		// if we get here it means we read until the end of the buffer
		// and it's still in the middle of non-break text

		if atEOF { // this is the end of the stream, return this last as a token
			if i > 0 {
				token = data[:i]
			}
			advance += i
			return
		}

		// not end of stream, tell it we need more (advance may have been incremented above)
		return advance, nil, nil
	})
	return &DefaultTokenizer{
		s: s,
	}
}

// DefaultTokenizer implements Tokenizer with a sensible default tokenization.
type DefaultTokenizer struct {
	s *bufio.Scanner
}

func (t *DefaultTokenizer) NextToken() ([]byte, error) {
	for t.s.Scan() {
		// fmt.Println(len(scanner.Bytes()) == 6)
		b := t.s.Bytes()
		if len(b) == 0 {
			continue
		}
		b = bytes.Trim(b, `/\:=`)
		return b, nil
	}
	if err := t.s.Err(); err != nil {
		// fmt.Fprintln(os.Stderr, "shouldn't see an error scanning a string")
		return nil, err
	}
	return nil, io.EOF
}

// Dist matches tailwind.Dist
type Dist interface {
	OpenDist(name string) (io.ReadCloser, error)
}

// PurgeKeysFromDist runs PurgeKeysFromReader on the appropriate(s) file from the dist.
// A check is done to see if Dist implements interface { PurgeKeyMap() map[string]struct{} }
// and this is used if avialable.  Otherwise the appropriate files(s) are processed from
// the dist using PurgeKeysFromReader.
func PurgeKeysFromDist(dist Dist) (map[string]struct{}, error) {

	pkmr, ok := dist.(purgeKeyMapper)
	if ok {
		return pkmr.PurgeKeyMap(), nil
	}

	f, err := dist.OpenDist("utilities")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return PurgeKeysFromReader(f)
}

// takes the rule info from a BeginRulesetGrammar returns the purge key if there is one or else empty string
func ruleToPurgeKey(data []byte, tokens []css.Token) string {

	if len(data) != 0 {
		panic("unexpected data")
	}

	// we're looking for Delim('.') followed by Ident() - we disregard everything after
	if len(tokens) < 2 {
		return ""
	}
	if tokens[0].TokenType != css.DelimToken || !bytes.Equal(tokens[0].Data, []byte(".")) {
		return ""
	}
	if tokens[1].TokenType != css.IdentToken {
		return ""
	}

	// if we get here we're good, we just need to unescape the ident (e.g. `\:` becomes just `:`)
	return cssUnescape(tokens[1].Data)
}

func cssUnescape(b []byte) string {

	var buf bytes.Buffer

	var i int
	for i = 0; i < len(b); i++ {
		if b[i] == '\\' {
			// set up buf with the stuff we've already scanned
			buf.Grow(len(b))
			buf.Write(b[:i])
			goto foundEsc
		}
		continue
	}
	// no escaping needed
	return string(b)

foundEsc:
	inEsc := false
	for ; i < len(b); i++ {
		if b[i] == '\\' && !inEsc {
			inEsc = true
			continue
		}
		buf.WriteByte(b[i])
		inEsc = false
	}
	return buf.String()
}

// PurgeKeysFromReader parses the contents of this reader as CSS and builds a map
// of purge keys.
func PurgeKeysFromReader(r io.Reader) (map[string]struct{}, error) {
	ret := make(map[string]struct{})

	// func (c *Converter) runParse(name string, p *css.Parser, w io.Writer, doPurge bool) error {
	p := css.NewParser(r, false)

mainLoop:
	for {

		gt, tt, data := p.Next()
		_, _ = tt, data

		switch gt {

		case css.ErrorGrammar:
			err := p.Err()
			if errors.Is(err, io.EOF) {
				break mainLoop
			}
			return ret, err

		case css.AtRuleGrammar:
		case css.BeginAtRuleGrammar:
		case css.EndAtRuleGrammar:
		case css.QualifiedRuleGrammar:
			k := ruleToPurgeKey(nil, p.Values())
			if k != "" {
				ret[k] = struct{}{}
			}
		case css.BeginRulesetGrammar:
			k := ruleToPurgeKey(nil, p.Values())
			if k != "" {
				ret[k] = struct{}{}
			}
		case css.DeclarationGrammar:
		case css.CustomPropertyGrammar:
		case css.EndRulesetGrammar:
		case css.TokenGrammar:
		case css.CommentGrammar:

		default: // verify we aren't missing a type
			panic(fmt.Errorf("unexpected grammar type %v at offset %v", gt, p.Offset()))

		}

	}

	return ret, nil
}
