package tools

import (
	"context"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math"
	"strconv"
)

type CalculatorArgs struct {
	Expression string `tool:"expression" desc:"math expression to evaluate"`
}

func RegisterCalculator(r *Registry) {
	Register(r, "calculator",
		"Evaluate a math expression and return the numeric result. Supports + - * / % **.",
		func(_ context.Context, a CalculatorArgs) (string, []Artifact, error) {
			// Translate ** → Go-incompatible; convert before parsing.
			expr := rewritePow(a.Expression)
			tree, err := parser.ParseExpr(expr)
			if err != nil {
				return "", nil, fmt.Errorf("parse: %w", err)
			}
			val, err := evalExpr(tree)
			if err != nil {
				return "", nil, err
			}
			return strconv.FormatFloat(val, 'f', -1, 64), nil, nil
		})
}

// rewritePow turns "a ** b" into "pow(a,b)" sentinels we handle in evalExpr.
// Simple textual pass — good enough for the demo expressions the LLM will emit.
func rewritePow(s string) string {
	out := []byte{}
	for i := 0; i < len(s); i++ {
		if i+1 < len(s) && s[i] == '*' && s[i+1] == '*' {
			out = append(out, '^')
			i++
			continue
		}
		out = append(out, s[i])
	}
	return string(out)
}

func evalExpr(n ast.Expr) (float64, error) {
	switch v := n.(type) {
	case *ast.BasicLit:
		if v.Kind == token.INT || v.Kind == token.FLOAT {
			return strconv.ParseFloat(v.Value, 64)
		}
		return 0, fmt.Errorf("unsupported literal: %s", v.Value)
	case *ast.ParenExpr:
		return evalExpr(v.X)
	case *ast.UnaryExpr:
		x, err := evalExpr(v.X)
		if err != nil {
			return 0, err
		}
		switch v.Op {
		case token.SUB:
			return -x, nil
		case token.ADD:
			return x, nil
		}
		return 0, fmt.Errorf("unsupported unary op: %s", v.Op)
	case *ast.BinaryExpr:
		l, err := evalExpr(v.X)
		if err != nil {
			return 0, err
		}
		r, err := evalExpr(v.Y)
		if err != nil {
			return 0, err
		}
		switch v.Op {
		case token.ADD:
			return l + r, nil
		case token.SUB:
			return l - r, nil
		case token.MUL:
			return l * r, nil
		case token.QUO:
			return l / r, nil
		case token.REM:
			return math.Mod(l, r), nil
		case token.XOR: // our stand-in for **
			return math.Pow(l, r), nil
		}
		return 0, fmt.Errorf("unsupported binary op: %s", v.Op)
	}
	return 0, fmt.Errorf("unsupported node: %T", n)
}
