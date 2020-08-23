package twpurge

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/tdewolff/parse/v2/css"
)

type purgeKeyMapper interface {
	PurgeKeyMap() map[string]struct{}
}

// FIXME: this should probably be called RuleNamesFromDist, and document the idea of "rule names" vs "purge keys".
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
func PurgeKeysFromReader(cssR io.Reader) (map[string]struct{}, error) {
	ret := make(map[string]struct{})

	p := css.NewParser(cssR, false)

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
