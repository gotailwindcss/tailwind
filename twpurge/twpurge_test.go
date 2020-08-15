package twpurge

import (
	"errors"
	"io"
	"reflect"
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

func TestPurgeKeysFromReader(t *testing.T) {

	pk, err := PurgeKeysFromReader(strings.NewReader(`
  .md\:bg-purple-500 {
    --bg-opacity: 1;
    background-color: #9f7aea;
    background-color: rgba(159, 122, 234, var(--bg-opacity))
  }
  .space-x-0 > :not(template) ~ :not(template) {
	--space-x-reverse: 0;
	margin-right: calc(0px * var(--space-x-reverse));
	margin-left: calc(0px * calc(1 - var(--space-x-reverse)))
  }
  .xl\:w-auto {
    width: auto
  }
  .focus\:placeholder-gray-200:focus:-ms-input-placeholder {
	--placeholder-opacity: 1;
	color: #edf2f7;
	color: rgba(237, 242, 247, var(--placeholder-opacity))
  }
  .placeholder-indigo-800::placeholder {
	--placeholder-opacity: 1;
	color: #434190;
	color: rgba(67, 65, 144, var(--placeholder-opacity))
  }
  .-my-56 {
	margin-top: -14rem;
	margin-bottom: -14rem
  }
  @media (min-width: 640px) {
	.sm\:space-y-0 > :not(template) ~ :not(template) {
	  --space-y-reverse: 0;
	  margin-top: calc(0px * calc(1 - var(--space-y-reverse)));
	  margin-bottom: calc(0px * var(--space-y-reverse))
	}
  }			
  .scale-y-125 {
	--transform-scale-y: 1.25
  }
`))
	if err != nil {
		t.Fatal(err)
	}

	v := struct{}{}
	if !reflect.DeepEqual(pk, map[string]struct{}{
		"md:bg-purple-500":           v,
		"space-x-0":                  v,
		"xl:w-auto":                  v,
		"focus:placeholder-gray-200": v,
		"placeholder-indigo-800":     v,
		"-my-56":                     v,
		"sm:space-y-0":               v,
		"scale-y-125":                v,
	}) {
		t.Errorf("unexpected result: %+v", pk)
	}

}
