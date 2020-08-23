package twpurge

import (
	"bufio"
	"bytes"
	"io"
)

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
