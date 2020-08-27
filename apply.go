package tailwind

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/tdewolff/parse/v2"
	"github.com/tdewolff/parse/v2/css"
)

type applier struct {
	m map[string]string
}

func (a *applier) apply(names []string) ([]byte, error) {
	ret := make([]byte, 0, len(names)*8)
	for _, name := range names {
		css, ok := a.m[name]
		if !ok {
			return ret, fmt.Errorf("unknown @apply name: %s", name)
		}
		ret = append(ret, css...)
	}
	return ret, nil
}

func newApplier(dist Dist) (*applier, error) {

	var a applier

	rc, err := dist.OpenDist("utilities")
	if err != nil {
		return nil, err
	}
	defer rc.Close()

	a.m = make(map[string]string, 128)

	depth := 0
	entryName := ""
	var entryData bytes.Buffer
	_ = entryData

	inp := parse.NewInput(rc)
	p := css.NewParser(inp, false)
parseLoop:
	for {

		gt, tt, data := p.Next()
		_ = tt
		_ = data

		switch gt {

		case css.ErrorGrammar:
			err := p.Err()
			if errors.Is(err, io.EOF) {
				break parseLoop
			}
			return nil, fmt.Errorf("applier.setupOnce: %w", err)

		case css.AtRuleGrammar:
			// ignored

		case css.BeginAtRuleGrammar:
			depth++

		case css.EndAtRuleGrammar:
			depth--

		case css.BeginRulesetGrammar:
			if depth != 0 { // ignore everything not at top level
				continue parseLoop
			}

			// only handle rules exactly maching pattern [Delim(".") Ident("someclass")]
			ts := trimTokenWs(p.Values())
			if !(len(ts) == 2 &&
				(ts[0].TokenType == css.DelimToken && bytes.Equal(ts[0].Data, []byte(`.`))) &&
				(ts[1].TokenType == css.IdentToken && len(ts[1].Data) > 0)) {
				continue parseLoop
			}
			name := string(ts[1].Data)
			if entryName != "" { // this should not be possible, just make sure
				panic(fmt.Errorf("about to start new entry %q but already in entry %q", name, entryName))
			}
			entryName = name

			// Delim('.') Ident('space-y-0')
			// log.Printf("applier BeginRulesetGrammar: data=%q, p.Values=%v", data, p.Values())
			// log.Printf("BeginRulesetGrammar: classes=%v (values=%v)", toklistClasses(p.Values()), p.Values())

			// inEntryName

			// err := write(w, data, p.Values(), '{')
			// if err != nil {
			// 	return err
			// }

		case css.EndRulesetGrammar:

			// we only need to look at entries that are closing
			if entryName == "" {
				continue parseLoop
			}

			b := entryData.Bytes()
			b = bytes.TrimSpace(b)
			a.m[entryName] = string(b)

			entryData.Reset()
			entryName = ""

		case css.DeclarationGrammar:
			if entryName == "" { // ignore content not inside an appropriate entry
				continue parseLoop
			}

			err := write(&entryData, data, ':', p.Values(), ';')
			if err != nil {
				return nil, err
			}

		case css.CustomPropertyGrammar:
			if entryName == "" { // ignore content not inside an appropriate entry
				continue parseLoop
			}

			// panic(fmt.Errorf(`CustomPropertyGrammar unsupported: data=%q, p.Values=%v`, data, p.Values()))
			err := write(&entryData, data, ':', p.Values(), ';')
			if err != nil {
				return nil, err
			}

		case css.QualifiedRuleGrammar:
			// FIXME: still not sure about when this is used...
			// panic(fmt.Errorf("applier: QualifiedRuleGrammar not yet implemented"))
			continue // should be okay to skip for now...

		case css.TokenGrammar:
			continue // just skip

		case css.CommentGrammar:
			continue // strip comments

		default: // verify we aren't missing a type
			panic(fmt.Errorf("unexpected grammar type %v at offset %v", gt, inp.Offset()))

		}

	}

	return &a, nil
}
