package parser

import (
	"fmt"
	"io"
	"regexp"
	"strings"
)

type Match interface{}

type MatchString string

type MatchTree []Match

type TaggedMatch struct {
	Match Match
	Tag   string
}

type fatalError struct {
	err error
}

func (fe fatalError) Error() string {
	return fmt.Sprintf("Fatal match error: %s", fe.err)
}

type Grammar struct {
	parse func(rs io.ReadSeeker) (Match, error)
}

func (g *Grammar) Set(ng *Grammar) {
	g.parse = ng.parse
}

func (g *Grammar) Node(node func(Match) (Match, error)) {
	oldp := g.parse
	g.parse = func(rs io.ReadSeeker) (Match, error) {
		m, err := oldp(rs)
		if err == nil {
			m, err = node(m)
			if err == nil {
				return m, nil
			}
		}
		return nil, err
	}
}

func (g *Grammar) Parse(rs io.ReadSeeker) (Match, error) {
	m, err := g.parse(rs)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (g *Grammar) ParseString(s string) (Match, error) {
	m, err := g.parse(strings.NewReader(s))
	if err != nil {
		return nil, err
	}
	return m, nil
}

func Set(set string) *Grammar {
	regset, _ := regexp.Compile(fmt.Sprintf("[%s]", set))
	return &Grammar{parse: func(rs io.ReadSeeker) (Match, error) {
		pos, _ := rs.Seek(0, 1)
		b := make([]byte, 1)
		c, _ := io.ReadFull(rs, b)
		if c < 1 {
			rs.Seek(pos, 0)
			return nil, fmt.Errorf("Unexpected EOF")
		}
		if regset.Match(b) {
			m := MatchString(b)
			return m, nil
		}
		rs.Seek(pos, 0)
		return nil, fmt.Errorf("Expected %q, got %q", set, string(b))
	}}
}

func Lit(text string) *Grammar {
	return &Grammar{parse: func(rs io.ReadSeeker) (Match, error) {
		pos, _ := rs.Seek(0, 1)
		b := make([]byte, len(text))
		c, _ := io.ReadFull(rs, b)
		if c < len(text) {
			rs.Seek(pos, 0)
			return nil, fmt.Errorf("Unexpected EOF")
		}
		if string(b) == text {
			m := MatchString(text)
			return m, nil
		}
		rs.Seek(pos, 0)
		return nil, fmt.Errorf("Expected %q, got %q", text, string(b))
	}}
}

func And(ps ...*Grammar) *Grammar {
	return &Grammar{parse: func(rs io.ReadSeeker) (Match, error) {
		pos, _ := rs.Seek(0, 1)
		matches := []Match{}
		for _, p := range ps {
			m, err := p.parse(rs)
			if err != nil {
				rs.Seek(pos, 0)
				return nil, err
			}
			if m != nil {
				matches = append(matches, m)
			}
		}
		mt := MatchTree(matches)
		return mt, nil
	}}
}

func Or(ps ...*Grammar) *Grammar {
	return &Grammar{parse: func(rs io.ReadSeeker) (Match, error) {
		pos, _ := rs.Seek(0, 1)
		errs := []error{}
		for _, p := range ps {
			m, err := p.parse(rs)
			if err == nil {
				return m, nil
			} else {
				if _, isFE := err.(fatalError); isFE {
					return nil, err
				}
				errs = append(errs, err)
			}
			rs.Seek(pos, 0)
		}
		return nil, fmt.Errorf("Or error, expected: (%v)", errs)
	}}
}

func Mult(n, m int, p *Grammar) *Grammar {
	if m == 0 {
		m = int(^uint(0) >> 1)
	}
	return &Grammar{parse: func(rs io.ReadSeeker) (Match, error) {
		pos, _ := rs.Seek(0, 1)
		ms := make(MatchTree, 0)
		for i := 0; i < m; i++ {
			match, err := p.parse(rs)
			if err != nil {
				if _, isFE := err.(fatalError); isFE {
					return nil, err
				}
				if i < n {
					rs.Seek(pos, 0)
					return nil, err
				}
				return ms, nil
			}
			ms = append(ms, match)
		}
		return ms, nil
	}}
}

func Optional(g *Grammar) *Grammar {
	return Mult(0, 1, g)
}

func Ignore(g *Grammar) *Grammar {
	return &Grammar{parse: func(rs io.ReadSeeker) (Match, error) {
		_, err := g.parse(rs)
		if err != nil {
			return nil, err
		}
		return nil, nil
	}}
}

func Require(ps ...*Grammar) *Grammar {
	return &Grammar{parse: func(rs io.ReadSeeker) (Match, error) {
		pos, _ := rs.Seek(0, 1)
		matches := []Match{}
		for _, p := range ps {
			m, err := p.parse(rs)
			if err != nil {
				rs.Seek(pos, 0)
				return nil, fatalError{err}
			}
			if m != nil {
				matches = append(matches, m)
			}
		}
		mt := MatchTree(matches)
		return mt, nil
	}}
}

func Tag(tag string, g *Grammar) *Grammar {
	return &Grammar{parse: func(rs io.ReadSeeker) (Match, error) {
		m, err := g.parse(rs)
		if err != nil {
			return nil, err
		}
		tm := TaggedMatch{
			Match: m,
			Tag:   tag,
		}
		return tm, nil
	}}
}

func TagMatch(tag string, match Match) Match {
	return TaggedMatch{
		Match: match,
		Tag:   tag,
	}
}

func GetTag(m Match, tag string) Match {
	switch m := m.(type) {
	case MatchTree:
		for _, mi := range m {
			tm := GetTag(mi, tag)
			if tm != nil {
				return tm
			}
		}
		return nil
	case MatchString:
		return nil
	case TaggedMatch:
		if tag == m.Tag {
			return m.Match
		}
		return GetTag(m.Match, tag)
	}
	return nil
}

func GetTags(m Match, tag string) []Match {
	switch m := m.(type) {
	case MatchTree:
		tms := []Match{}
		for _, mi := range m {
			tm := GetTags(mi, tag)
			if tm != nil {
				tms = append(tms, tm...)
			}
		}
		return tms
	case MatchString:
		return nil
	case TaggedMatch:
		if tag == m.Tag {
			return append(GetTags(m.Match, tag), m.Match)
		}
		return GetTags(m.Match, tag)
	}
	return nil
}

func String(m Match) string {
	switch m := m.(type) {
	case MatchTree:
		ss := make([]string, len(m))
		for i, mi := range m {
			ss[i] = String(mi)
		}
		return strings.Join(ss, "")
	case MatchString:
		return string(m)
	case string:
		return m
	}
	return ""
}
