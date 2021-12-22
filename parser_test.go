package parser

import (
	"math"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

type Node interface {
	Value() int
}

type BinOp struct {
	Op1 Node
	Op  string
	Op2 Node
}

func (bo BinOp) Value() int {
	switch bo.Op {
	case "+":
		return bo.Op1.Value() + bo.Op2.Value()
	case "-":
		return bo.Op1.Value() - bo.Op2.Value()
	case "*":
		return bo.Op1.Value() * bo.Op2.Value()
	case "/":
		return bo.Op1.Value() / bo.Op2.Value()
	case "^":
		return int(math.Pow(float64(bo.Op1.Value()), float64(bo.Op2.Value())))
	}
	panic("Unknown operator")
}

type Num int

func (n Num) Value() int {
	return int(n)
}

func ParseParen(sr StatefulReader) (Node, error) {
	_, err := Lit("(")(sr)
	if err != nil {
		return nil, err
	}
	n, err := ParseExpr(sr)
	if err != nil {
		return nil, err
	}
	_, err = Lit(")")(sr)
	if err != nil {
		return nil, err
	}
	return n, err
}

func ParseExpr(sr StatefulReader) (Node, error) {
	return ParseBinOp(
		ParseBinOp(
			ParseBinOp(
				Or(ParseParen, ParseNum),
				"^",
			),
			"*", "/",
		),
		"+", "-",
	)(sr)
}

func ParseBinOp(opType func(StatefulReader) (Node, error), ops ...string) func(StatefulReader) (Node, error) {
	parseOps := []func(StatefulReader) (string, error){}
	for _, op := range ops {
		parseOps = append(parseOps, Lit(op))
	}
	return func(sr StatefulReader) (Node, error) {
		n, err := opType(sr)
		if err != nil {
			return nil, err
		}
		for {

			s := sr.State()
			op, err := Or(parseOps...)(sr)
			if err != nil {
				sr.Restore(s)
				break
			}
			n2, err := opType(sr)
			if err != nil {
				sr.Restore(s)
				break
			}
			n = BinOp{Op1: n, Op: op, Op2: n2}
		}
		return n, nil
	}
}

var ParseNum = Convert(And(Optional(Lit("-")), Convert(Mult(1, 0, Set("0-9")), func(s []string) (string, error) {
	return strings.Join(s, ""), nil
})), func(s []string) (Node, error) {
	n, err := strconv.Atoi(strings.Join(s, ""))
	if err != nil {
		return nil, err
	}
	return Num(n), nil
})

func TestLit(t *testing.T) {
	t.Parallel()
	p := Lit("foo")
	out, err := parse("foo", p)
	if err != nil {
		t.Error(err)
	}
	if out != "foo" {
		t.Errorf("Expected %s, got %s", "foo", out)
	}
}

func TestMult(t *testing.T) {
	t.Parallel()
	p := Mult(0, 3, Lit("foo"))
	out, err := parse("foo", p)
	if err != nil {
		t.Error(err)
	}
	assert(t, out, []string{"foo"})
	out, err = parse("foofoofoo", p)
	if err != nil {
		t.Error(err)
	}
	assert(t, out, []string{"foo", "foo", "foo"})
}

func TestMultOr(t *testing.T) {
	t.Parallel()
	p := Mult(0, 3, Or(Lit("bar"), Lit("foo")))
	out, err := parse("foo", p)
	if err != nil {
		t.Error(err)
	}
	assert(t, out, []string{"foo"})
	out, err = parse("foobarfoo", p)
	if err != nil {
		t.Error(err)
	}
	assert(t, out, []string{"foo", "bar", "foo"})
}

func TestExpr(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in  string
		out int
	}{
		{"1+2", 3},
		{"1+2*3", 7},
		{"1+2*3-4", 3},
		{"1+2*3-4*5", -13},
		{"2-1", 1},
		{"(1+2)*3", 9},
		{"(1+2)*(3-4)", -3},
		{"2^3", 8},
		{"1+2^2", 5},
		{"1+2^2+1", 6},
		{"(1+2)^2", 9},
	}
	for _, test := range tests {
		out, err := parse(test.in, ParseExpr)
		if err != nil {
			t.Error(err)
		}
		assertSrc(t, out, out.Value(), test.out)
	}
}

func assert[T any](t *testing.T, got, expected T) {
	t.Helper()
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Expected %v, got %v", expected, got)
	}
}

func assertSrc[T any](t *testing.T, src any, got, expected T) {
	t.Helper()
	if !reflect.DeepEqual(got, expected) {
		t.Errorf("Expected (%v) %v, got %v", src, expected, got)
	}
}

func parse[T any](s string, p func(StatefulReader) (T, error)) (T, error) {
	return p(SimpleReader{strings.NewReader(s)})
}
