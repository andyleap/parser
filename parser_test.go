package parser

import (
	"strconv"
	"testing"
)

func TestMath(t *testing.T) {

	number := And(Mult(0, 1, Lit("-")), Mult(1, 0, Set("0-9")), Mult(0, 1, And(Lit("."), Mult(0, 0, Set("0-9")))))
	number.Node(func(m Match) (Match, error) {
		v, err := strconv.ParseFloat(String(m), 64)
		if err == nil {
			return v, nil
		}
		return nil, err
	})

	expr := &Grammar{}

	parenexpr := And(Lit("("), Tag("expr", expr), Lit(")"))
	parenexpr.Node(func(m Match) (Match, error) {
		return GetTag(m, "expr"), nil
	})

	factor := Or(number, parenexpr)

	term := And(factor, Mult(0, 0, And(Set("*/"), factor)))
	term.Node(func(m Match) (Match, error) {
		mt := m.(MatchTree)
		val := mt[0].(float64)
		for _, op := range mt[1].(MatchTree) {
			switch op.(MatchTree)[0].(MatchString) {
			case "*":
				val = val * op.(MatchTree)[1].(float64)
			case "/":
				val = val / op.(MatchTree)[1].(float64)
			}
		}
		return val, nil
	})

	expr.Set(And(term, Mult(0, 0, And(Set("+-"), term))))
	expr.Node(func(m Match) (Match, error) {
		mt := m.(MatchTree)
		val := mt[0].(float64)
		for _, op := range mt[1].(MatchTree) {
			switch op.(MatchTree)[0].(MatchString) {
			case "+":
				val = val + op.(MatchTree)[1].(float64)
			case "-":
				val = val - op.(MatchTree)[1].(float64)
			}
		}
		return val, nil
	})

	m, err := expr.ParseString("-12.5*4/(2+2)+(30*2)/3+6")

	if err != nil {
		t.Error(err)
	}
	if m.(float64) != 13.5 {
		t.Errorf("%v != 13.5", m)
	}

	m, err = expr.ParseString("5-1")

	if err != nil {
		t.Error(err)
	}
	if m.(float64) != 4 {
		t.Errorf("%v != 4", m)
	}
}

func BenchmarkMath(b *testing.B) {

	number := And(Mult(1, 0, Set("0-9")), Mult(0, 1, And(Lit("."), Mult(0, 0, Set("0-9")))))
	number.Node(func(m Match) (Match, error) {
		v, err := strconv.ParseFloat(String(m), 64)
		if err != nil {
			return nil, err
		}
		return v, nil
	})

	expr := &Grammar{}

	parenexpr := And(Lit("("), Tag("expr", expr), Lit(")"))
	parenexpr.Node(func(m Match) (Match, error) {
		return GetTag(m, "expr"), nil
	})

	factor := Or(number, parenexpr)

	term := And(factor, Mult(0, 0, And(Set("*/"), factor)))
	term.Node(func(m Match) (Match, error) {
		mt := m.(MatchTree)
		val := mt[0].(float64)
		for _, op := range mt[1].(MatchTree) {
			switch op.(MatchTree)[0].(MatchString) {
			case "*":
				val = val * op.(MatchTree)[1].(float64)
			case "/":
				val = val / op.(MatchTree)[1].(float64)
			}
		}
		return val, nil
	})

	expr.Set(And(term, Mult(0, 0, And(Set("+-"), term))))
	expr.Node(func(m Match) (Match, error) {
		mt := m.(MatchTree)
		val := mt[0].(float64)
		for _, op := range mt[1].(MatchTree) {
			switch op.(MatchTree)[0].(MatchString) {
			case "+":
				val = val + op.(MatchTree)[1].(float64)
			case "-":
				val = val - op.(MatchTree)[1].(float64)
			}
		}
		return val, nil
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		expr.ParseString("5-1")
	}
}
