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
	Tag string
}

type Grammer struct {
	parse func(rs io.ReadSeeker) (Match, error)
}

func (g *Grammer) Set(ng *Grammer) {
	g.parse = ng.parse
}

func (g *Grammer) Node(node func(Match) (Match, error)) {
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

func (g *Grammer) Parse(rs io.ReadSeeker) (Match, error) {
	m, err := g.parse(rs)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (g *Grammer) ParseString(s string) (Match, error) {
	m, err := g.parse(strings.NewReader(s))
	if err != nil {
		return nil, err
	}
	return m, nil
}

func Set(set string) *Grammer {
	regset, _ := regexp.Compile(fmt.Sprintf("[%s]", set))
	return &Grammer{parse: func(rs io.ReadSeeker) (Match, error) {
		pos, _ := rs.Seek(0, 1)
		b := make([]byte, 1)
		c, _ := rs.Read(b)
		if c < 1 {
			rs.Seek(pos, 0)
			return nil, fmt.Errorf("Unexpected EOF")
		}
		if regset.Match(b) {
			m := MatchString(b)
			return m, nil
		}
		rs.Seek(pos, 0)
		return nil, fmt.Errorf("Expected %s, got %s", set, string(b))
	}}
}

func Lit(text string) *Grammer {
	return &Grammer{parse: func(rs io.ReadSeeker) (Match, error) {
		pos, _ := rs.Seek(0, 1)
		b := make([]byte, len(text))
		c, _ := rs.Read(b)
		if c < len(text) {
			rs.Seek(pos, 0)
			return nil, fmt.Errorf("Unexpected EOF")
		}
		if string(b) == text {
			m := MatchString(text)
			return m, nil
		}
		rs.Seek(pos, 0)
		return nil, fmt.Errorf("Expected %s, got %s", text, string(b))
	}}
}

func And(ps ...*Grammer) *Grammer {
	return &Grammer{parse: func(rs io.ReadSeeker) (Match, error) {
		pos, _ := rs.Seek(0, 1)
		matches := []Match{}
		for _, p := range ps {
			m, err := p.parse(rs)
			if err != nil {
				rs.Seek(pos, 0)
				return nil, err
			}
			matches = append(matches, m)
		}
		mt := MatchTree(matches)
		return mt, nil
	}}
}

func Or(ps ...*Grammer) *Grammer {
	return &Grammer{parse: func(rs io.ReadSeeker) (Match, error) {
		pos, _ := rs.Seek(0, 1)
		errs := []error{}
		for _, p := range ps {
			m, err := p.parse(rs)
			if err == nil {
				return m, nil
			} else {
				errs = append(errs, err)
			}
			rs.Seek(pos, 0)
		}
		return nil, fmt.Errorf("Or error, expected: (%v)", errs)
	}}
}

func Mult(n, m int, p *Grammer) *Grammer {
	if m == 0 {
		m = int(^uint(0) >> 1)
	}
	return &Grammer{parse: func(rs io.ReadSeeker) (Match, error) {
		pos, _ := rs.Seek(0, 1)
		ms := make(MatchTree, 0)
		for i := 0; i < m; i++ {
			match, err1 := p.parse(rs)
			if err1 != nil {
				if i < n {
					rs.Seek(pos, 0)
					return nil, fmt.Errorf("Error: not enough")
				}
				return ms, nil
			}
			ms = append(ms, match)
		}
		return ms, nil
	}}
}

func Tag(tag string, g *Grammer) *Grammer {
	return &Grammer{parse: func(rs io.ReadSeeker) (Match, error) {
		m, err := g.parse(rs)
		if err != nil {
			return nil, err
		}
		tm := TaggedMatch{
			Match: m,
			Tag: tag,
		}
		return tm, nil
	}}
}

func GetTag(m Match, tag string) *TaggedMatch {
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
			return &m
		}
		return GetTag(m.Match, tag)
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
	}
	return ""
}