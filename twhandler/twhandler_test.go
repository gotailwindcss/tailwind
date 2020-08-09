package twhandler_test

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gotailwindcss/tailwind/twembed"
	"github.com/gotailwindcss/tailwind/twhandler"
)

func TestHandler(t *testing.T) {

	td, _ := filepath.Abs("testdata")
	h := twhandler.New(twembed.New(), http.Dir(td), "/td1")

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/td1/demo1.css", nil)
	h.ServeHTTP(w, r)
	res := w.Result()
	defer res.Body.Close()
	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	bs := string(b)
	if !strings.Contains(bs, `b,strong{`) {
		t.Errorf("missing expected string")
	}
	if !strings.Contains(bs, `.test1{padding-left:0.25rem;`) {
		t.Errorf("didn't match .test1")
	}
	if t.Failed() {
		t.Logf("b = %s", b)
	}

	// TODO: table test with cases for compressor, 304, mod time of file changes, multiple files, cache disabled, etc.

}
