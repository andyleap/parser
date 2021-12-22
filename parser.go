package parser

import (
	"fmt"
	"io"
	"unicode/utf8"
)

//type Parser func[T any](rs io.ReadSeeker) (T, error)

type fatalError struct {
	err error
}

func (fe fatalError) Error() string {
	return fmt.Sprintf("Fatal match error: %s", fe.err)
}

type StatefulReader interface {
	io.Reader
	State() any
	Restore(any)
}

type SimpleReader struct {
	r io.ReadSeeker
}

func (sr SimpleReader) Read(p []byte) (n int, err error) {
	return sr.r.Read(p)
}

func (sr SimpleReader) State() any {
	s, _ := sr.r.Seek(0, 1)
	return s
}

func (sr SimpleReader) Restore(s any) {
	sr.r.Seek(s.(int64), 0)
}

func Lit(text string) func(sr StatefulReader) (string, error) {
	return func(sr StatefulReader) (string, error) {
		s := sr.State()
		b := make([]byte, len(text))
		c, _ := io.ReadFull(sr, b)
		if c < len(text) {
			sr.Restore(s)
			return "", fmt.Errorf("Unexpected EOF")
		}
		if string(b) == text {
			return text, nil
		}
		sr.Restore(s)
		return "", fmt.Errorf("Expected %q, got %q", text, string(b))
	}
}

func readRune(sr StatefulReader) (rune, error) {
	b := make([]byte, 1, 4)
	_, err := sr.Read(b)
	for !utf8.FullRune(b) && err != nil {
		b = b[:len(b)+1]
		_, err = sr.Read(b[len(b)-1:])
	}
	r, _ := utf8.DecodeRune(b)
	return r, err
}

func Set(text string) func(sr StatefulReader) (string, error) {
	//expand 0-9 to 0123456789
	final := []rune{}
	rawtext := []rune(text)
	for i := range rawtext {
		if rawtext[i] == '-' && i > 0 && i < len(rawtext)-1 {
			for j := rawtext[i-1] + 1; j < rawtext[i+1]; j++ {
				final = append(final, j)
			}
		} else {
			final = append(final, rawtext[i])
		}
	}

	return func(sr StatefulReader) (string, error) {
		s := sr.State()
		r, err := readRune(sr)
		if err != nil {
			sr.Restore(s)
			return "", err
		}
		for _, tr := range final {
			if r == tr {
				return string(r), nil
			}
		}
		sr.Restore(s)
		return "", fmt.Errorf("Expected %q, got %q", text, string(r))
	}
}

func Or[T any](ps ...func(sr StatefulReader) (T, error)) func(sr StatefulReader) (T, error) {
	return func(sr StatefulReader) (T, error) {
		s := sr.State()
		for _, p := range ps {
			v, err := p(sr)
			if err == nil {
				return v, nil
			}
			sr.Restore(s)
		}
		var t T
		return t, fmt.Errorf("No match")
	}
}

func And[T any](ps ...func(sr StatefulReader) (T, error)) func(sr StatefulReader) ([]T, error) {
	return func(sr StatefulReader) ([]T, error) {
		vs := []T{}
		s := sr.State()
		for _, p := range ps {
			v, err := p(sr)
			if err != nil {
				sr.Restore(s)
				return nil, err
			}
			vs = append(vs, v)
		}
		return vs, nil
	}
}

func Optional[T any](p func(sr StatefulReader) (T, error)) func(sr StatefulReader) (T, error) {
	return func(sr StatefulReader) (T, error) {
		s := sr.State()
		p, err := p(sr)
		if err != nil {
			sr.Restore(s)
		}
		if _, isFE := err.(fatalError); isFE {
			return p, err
		}
		return p, nil
	}
}

func Mult[T any](n, m int, p func(sr StatefulReader) (T, error)) func(sr StatefulReader) ([]T, error) {
	if m == 0 {
		m = int(^uint(0) >> 1)
	}
	return func(sr StatefulReader) ([]T, error) {
		s := sr.State()
		ms := []T{}
		for i := 0; i < m; i++ {
			match, err := p(sr)
			if err != nil {
				if _, isFE := err.(fatalError); isFE {
					return nil, err
				}
				if i < n {
					sr.Restore(s)
					return nil, err
				}
				return ms, nil
			}
			ms = append(ms, match)
		}
		return ms, nil
	}
}

func Convert[T, U any](p func(sr StatefulReader) (T, error), f func(T) (U, error)) func(sr StatefulReader) (U, error) {
	return func(sr StatefulReader) (U, error) {
		v, err := p(sr)
		if err != nil {
			var u U
			return u, err
		}
		return f(v)
	}
}
