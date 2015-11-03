# parser
Parser Combinator Library

# Usage

```
import p "github.com/andyleap/parser"

func MakeParser() *p.Grammar {
  number := p.And(p.Mult(0, 1, p.Lit("-")), p.Mult(1, 0, p.Set("0-9")), p.Mult(0, 1, p.And(p.Lit("."), p.Mult(0, 0, p.Set("0-9")))))
  number.Node(func(m Match) (Match, error) {
		v, err := strconv.ParseFloat(String(m), 64)
		if err != nil {
			return nil, err
		}
		return v, nil
	})
  
  expr := p.And(number, p.Mult(0, 0, p.And(Set("+-"), number)))
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
	
	return expr
}

func main() {
  mathparser := MakeParser()
  res, _ := mathparser.ParseString("1+2")
  fmt.Println(res)
  res, _ = mathparser.ParseString("1+2-5+3.652+3")
  fmt.Println(res)
  res, _ = mathparser.ParseString("5--1")
  fmt.Println(res)
}
```
