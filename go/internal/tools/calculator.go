package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"claudecode/internal/core"
)

type calculatorTool struct{}

type calculatorInput struct {
	Expression string `json:"expression"`
}

func NewCalculator() core.Tool { return &calculatorTool{} }

func (calculatorTool) Name() string { return "Calculator" }

func (calculatorTool) Description() string {
	return "Evaluate an arithmetic expression with + - * / % ^ and parentheses. Returns '<expr> = <result>'."
}

func (calculatorTool) Schema() json.RawMessage {
	return json.RawMessage(`{
  "type": "object",
  "properties": {
    "expression": {"type": "string"}
  },
  "required": ["expression"],
  "additionalProperties": false
}`)
}

func (calculatorTool) Run(ctx context.Context, input json.RawMessage) (string, error) {
	var in calculatorInput
	if err := json.Unmarshal(input, &in); err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}
	expr := strings.TrimSpace(in.Expression)
	if expr == "" {
		return "", fmt.Errorf("expression is required")
	}
	v, err := evalExpr(expr)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s = %s", expr, formatFloat(v)), nil
}

func formatFloat(v float64) string {
	if math.IsNaN(v) {
		return "NaN"
	}
	if math.IsInf(v, 1) {
		return "+Inf"
	}
	if math.IsInf(v, -1) {
		return "-Inf"
	}
	if v == math.Trunc(v) && math.Abs(v) < 1e15 {
		return strconv.FormatFloat(v, 'f', 0, 64)
	}
	return strconv.FormatFloat(v, 'g', -1, 64)
}

type calcToken struct {
	kind byte // 'n' number, 'o' operator, '(' or ')'
	num  float64
	op   byte
}

func tokenizeExpr(s string) ([]calcToken, error) {
	var out []calcToken
	i := 0
	prevValue := false
	for i < len(s) {
		c := s[i]
		if c == ' ' || c == '\t' {
			i++
			continue
		}
		if (c >= '0' && c <= '9') || c == '.' {
			j := i
			for j < len(s) && ((s[j] >= '0' && s[j] <= '9') || s[j] == '.') {
				j++
			}
			n, err := strconv.ParseFloat(s[i:j], 64)
			if err != nil {
				return nil, fmt.Errorf("bad number %q", s[i:j])
			}
			out = append(out, calcToken{kind: 'n', num: n})
			i = j
			prevValue = true
			continue
		}
		switch c {
		case '+', '-', '*', '/', '%', '^':
			op := c
			if (op == '+' || op == '-') && !prevValue {
				// unary +/-: treat by emitting 0 before it
				out = append(out, calcToken{kind: 'n', num: 0})
			}
			out = append(out, calcToken{kind: 'o', op: op})
			i++
			prevValue = false
		case '(':
			out = append(out, calcToken{kind: '('})
			i++
			prevValue = false
		case ')':
			out = append(out, calcToken{kind: ')'})
			i++
			prevValue = true
		default:
			return nil, fmt.Errorf("unexpected character %q", string(c))
		}
	}
	return out, nil
}

func opPrec(op byte) (int, bool) {
	switch op {
	case '+', '-':
		return 1, true
	case '*', '/', '%':
		return 2, true
	case '^':
		return 3, false // right-associative
	}
	return 0, true
}

func applyOp(op byte, a, b float64) (float64, error) {
	switch op {
	case '+':
		return a + b, nil
	case '-':
		return a - b, nil
	case '*':
		return a * b, nil
	case '/':
		if b == 0 {
			return 0, errors.New("division by zero")
		}
		return a / b, nil
	case '%':
		if b == 0 {
			return 0, errors.New("modulo by zero")
		}
		return math.Mod(a, b), nil
	case '^':
		return math.Pow(a, b), nil
	}
	return 0, fmt.Errorf("unknown operator %q", string(op))
}

func evalExpr(s string) (float64, error) {
	toks, err := tokenizeExpr(s)
	if err != nil {
		return 0, err
	}
	var values []float64
	var ops []byte // operators and '('

	popApply := func() error {
		if len(values) < 2 || len(ops) == 0 {
			return errors.New("malformed expression")
		}
		op := ops[len(ops)-1]
		ops = ops[:len(ops)-1]
		b := values[len(values)-1]
		a := values[len(values)-2]
		values = values[:len(values)-2]
		v, err := applyOp(op, a, b)
		if err != nil {
			return err
		}
		values = append(values, v)
		return nil
	}

	for _, t := range toks {
		switch t.kind {
		case 'n':
			values = append(values, t.num)
		case '(':
			ops = append(ops, '(')
		case ')':
			for len(ops) > 0 && ops[len(ops)-1] != '(' {
				if err := popApply(); err != nil {
					return 0, err
				}
			}
			if len(ops) == 0 {
				return 0, errors.New("mismatched )")
			}
			ops = ops[:len(ops)-1]
		case 'o':
			p1, leftAssoc := opPrec(t.op)
			for len(ops) > 0 {
				top := ops[len(ops)-1]
				if top == '(' {
					break
				}
				p2, _ := opPrec(top)
				if p2 > p1 || (p2 == p1 && leftAssoc) {
					if err := popApply(); err != nil {
						return 0, err
					}
					continue
				}
				break
			}
			ops = append(ops, t.op)
		}
	}
	for len(ops) > 0 {
		if ops[len(ops)-1] == '(' {
			return 0, errors.New("mismatched (")
		}
		if err := popApply(); err != nil {
			return 0, err
		}
	}
	if len(values) != 1 {
		return 0, errors.New("malformed expression")
	}
	return values[0], nil
}
