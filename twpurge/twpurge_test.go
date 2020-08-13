package twpurge

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestDefaultTokenizer(t *testing.T) {

	tz := NewDefaultTokenizer(strings.NewReader(`
	<html class="blah"><body id="blee" class="sm:px-1 lg:w-10"><div class="w-1/2"></div>  </body></html>
`))

	var tokList []string
	for {
		tok, err := tz.NextToken()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			t.Fatal(err)
		}
		tokList = append(tokList, string(tok))
		// t.Logf("TOKEN(len=%d): %s", len(tok), tok)
	}

	tokMap := make(map[string]bool, len(tokList))
	for _, tok := range tokList {
		tokMap[tok] = true
	}
	if !tokMap["sm:px-1"] {
		t.Fail()
	}
	if !tokMap["lg:w-10"] {
		t.Fail()
	}
	if !tokMap["w-1/2"] {
		t.Fail()
	}
	if tokMap["class="] { // should not have trailing equal sign
		t.Fail()
	}

}
