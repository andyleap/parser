package parser

import (
	"testing"
	"strings"
	"strconv"
)

func test_main(t *testing.T) {

	number := And(Mult(0, 1, Lit("-")), Mult(1, 0, Set("0-9")), Mult(0, 1, And(Lit("."), Mult(0, 0, Set("0-9")))))
	number.Node(func(m Match) Match {
		v, _ := strconv.ParseFloat(String(m), 64)
		return v
	})

	expr := &Grammer{}
	
	parenexpr := And(Lit("("), Tag("expr", expr), Lit(")"))
	parenexpr.Node(func(m Match) Match {
		return GetTag(m, "expr").Match
	})

	factor := Or(number, parenexpr)

	term := And(factor, Mult(0, 0, And(Set("*/"), factor)))
	term.Node(func(m Match) Match {
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
		return val
	})

	expr.Set(And(term, Mult(0, 0, And(Set("+-"), term))))
	expr.Node(func(m Match) Match {
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
		return val
	})
	
	
	test := strings.NewReader("-12.5*4/(2+2)+(30*2)/3+6")
	m, err := expr.parse(test)

	if err != nil {
		t.Error(err)
	}
	if m.(float64) != 13.5 {
		t.Errorf("%v != 13.5", m)
	}
}