package twpurge

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Hmmmm
// What if we broke up the purging into two steps - the scan for which classes to include is run manually
// but then it stays in a file and is used to filter results quickly.  This way you get purged files
// in development (one benefit of this is you don't have different styles in dev and production).

// From a naming perspective, breaking up the tasks of "scanning for things to not purge" and "implementing the ShouldPurgeKey method"
// should probably be two different things...
// twpurge.Scanner - scans text to exract purge keys
// twpurge.Checker - just has ShouldPurgeKey
// twpurge.CheckerFunc - implement ShouldPurgeKey as a function
// twpurge.Map - cast map[string]struct{}{} to type that implements Checker

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

// Checker is implemented by something that can answer the question "should this CSS rule be purged from the output because it is unused".
type Checker interface {
	ShouldPurgeKey(k string) bool
}

// Map is a set of strings that implements Checker.
// The output of a Scanner is a Map that can be used to during conversion
// to rapidly check if a style rule needs to be output.
type Map map[string]struct{}

// ShouldPurgeKey implements Checker.
func (m Map) ShouldPurgeKey(k string) bool {
	_, ok := m[k]
	return !ok
}

func (m Map) Merge(fromMap Map) {
	for k, v := range fromMap {
		m[k] = v
	}
}

// Scanner scans through textual files (generally HTML-like content) and looks for tokens
// to be preserved when purging.  The scanning is intentionally naive in order to keep
// it's rules simple to understand and reasonbly performant. (TODO: explain more)
type Scanner struct {
	tokenizerFunc func(r io.Reader) Tokenizer
	ruleNames     map[string]struct{}
	m             Map
}

func NewScanner(ruleNames map[string]struct{}) *Scanner {
	return &Scanner{ruleNames: ruleNames}
}

func NewScannerFromDist(dist Dist) (*Scanner, error) {
	pkmap, err := PurgeKeysFromDist(dist)
	if err != nil {
		return nil, err
	}
	return NewScanner(pkmap), nil
}

var defaultTokenizerFunc = func(r io.Reader) Tokenizer { return NewDefaultTokenizer(r) }

func (s *Scanner) Scan(r io.Reader) error {

	if s.m == nil {
		s.m = make(Map, len(s.ruleNames)/16)
	}

	tf := s.tokenizerFunc
	if tf == nil {
		tf = defaultTokenizerFunc
	}
	t := tf(r)

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
		if s.ruleNames != nil {
			_, found = s.ruleNames[bstr]
		}
		if found {
			s.m[bstr] = struct{}{}
		}
	}
}

func (s *Scanner) ScanFile(fpath string) error {
	f, err := os.Open(fpath)
	if err != nil {
		return err
	}
	defer f.Close()
	return s.Scan(f)
}

// WalkFunc returns a function which can be called by filepath.Walk to scan each matching file encountered.
// The fnmatch func says which files to scan, if nil is passed then MatchDefault will be used.
func (s *Scanner) WalkFunc(fnmatch func(fn string) bool) filepath.WalkFunc {
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
		return s.ScanFile(fpath)
	})
}

// Map returns the Map which is the result of all previous Scan calls.
func (s *Scanner) Map() Map {
	return s.m
}

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

// // Purger can parse markup and accumulate a list of purge keys which can be used to
// // vet the output of tailwind.Converter to eliminate unused styles.
// type Purger struct {
// 	purgeKeyMap  map[string]struct{} // all possible purge keys, passed in from New
// 	parsedKeyMap map[string]struct{} // the keys parsed from the markup (filtered to include only purgeKeyMap entries if not nil), these are the keys to be kept during conversion
// 	tokenizer    Tokenizer
// }

// // TODO:
// // New(Dist)
// // NewFromMap
// // BuildPurgeKeyMap(Dist) map
// // and then twfiles does not have purge support, but twembed can have it all preprocessed
// // New(Dist) looks for interface and calls BuildPurgeKeyMap if not implemeneted

// // MustNew is like New but panics upon error.
// func MustNew(dist Dist) *Purger {
// 	ret, err := New(dist)
// 	if err != nil {
// 		panic(err)
// 	}
// 	return ret
// }

// // New returns a new Purger instance.  Uses PurgeKeysFromDist.
// func New(dist Dist) (*Purger, error) {

// 	// moved to PurgeKeysFromDist
// 	// pkmr, ok := dist.(purgeKeyMapper)
// 	// if ok {
// 	// 	return NewFromMap(pkmr.PurgeKeyMap()), nil
// 	// }

// 	pkmap, err := PurgeKeysFromDist(dist)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return NewFromMap(pkmap), nil
// }

// // NewFromMap returns a new Purger instance. If purgeKeyMap is not nil, it is a map
// // of all the possible keys that can be purged, which is then used during
// // markup parsing to be able to scan for just the purge keys that are relevant.
// // Passing nil will still result in proper function but will use more memory
// // and potentially be slower.
// func NewFromMap(purgeKeyMap map[string]struct{}) *Purger {
// 	return &Purger{purgeKeyMap: purgeKeyMap, parsedKeyMap: make(map[string]struct{})}
// }

// // WalkFunc returns a function which can be called by filepath.Walk
// func (p *Purger) WalkFunc(fnmatch func(fn string) bool) filepath.WalkFunc {
// 	if fnmatch == nil {
// 		fnmatch = MatchDefault
// 	}
// 	return filepath.WalkFunc(func(fpath string, info os.FileInfo, err error) error {
// 		if info.IsDir() { // ignore dirs
// 			return nil
// 		}
// 		if err != nil { // any stat errors get returned as-is
// 			return err
// 		}
// 		if !fnmatch(fpath) { // ignore if filename doesn't match
// 			return nil
// 		}
// 		return p.ParseFile(fpath)
// 	})
// }

// func (p *Purger) SetTokenizer(t Tokenizer) {
// 	p.tokenizer = t
// }

// func (p *Purger) ParseReader(r io.Reader) error {
// 	if p.parsedKeyMap == nil {
// 		p.parsedKeyMap = make(map[string]struct{})
// 	}

// 	t := NewDefaultTokenizer(r)
// 	for {
// 		b, err := t.NextToken()
// 		if err != nil {
// 			if errors.Is(err, io.EOF) {
// 				return nil
// 			}
// 			return err
// 		}
// 		bstr := string(b) // FIXME: cheating with zero-alloc unsafe cast would be appropriate here
// 		found := true
// 		if p.purgeKeyMap != nil {
// 			_, found = p.purgeKeyMap[bstr]
// 		}
// 		if found {
// 			p.parsedKeyMap[bstr] = struct{}{}
// 		}
// 	}

// }

// func (p *Purger) ParseFile(fpath string) error {
// 	f, err := os.Open(fpath)
// 	if err != nil {
// 		return err
// 	}
// 	defer f.Close()
// 	return p.ParseReader(f)
// }

// func (p *Purger) ShouldPurgeKey(k string) bool {
// 	_, ok := p.parsedKeyMap[k]
// 	return !ok
// }

// Dist matches tailwind.Dist
type Dist interface {
	OpenDist(name string) (io.ReadCloser, error)
}
