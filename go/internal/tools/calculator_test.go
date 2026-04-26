package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func runCalc(t *testing.T, expr string) (string, error) {
	t.Helper()
	tool := NewCalculator()
	in, _ := json.Marshal(map[string]any{"expression": expr})
	return tool.Run(context.Background(), in)
}

func TestCalculatorBasicAddition(t *testing.T) {
	out, err := runCalc(t, "1+2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(out, "= 3") {
		t.Errorf("expected '= 3' suffix, got %q", out)
	}
}

func TestCalculatorParenthesesAndPrecedence(t *testing.T) {
	out, err := runCalc(t, "2*(3+4)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(out, "= 14") {
		t.Errorf("expected '= 14' suffix, got %q", out)
	}
}

func TestCalculatorExponent(t *testing.T) {
	out, err := runCalc(t, "2^10")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(out, "= 1024") {
		t.Errorf("expected '= 1024' suffix, got %q", out)
	}
}

func TestCalculatorRightAssociativeExponent(t *testing.T) {
	// 2^3^2 should be 2^(3^2) = 2^9 = 512.
	out, err := runCalc(t, "2^3^2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(out, "= 512") {
		t.Errorf("expected '= 512' suffix, got %q", out)
	}
}

func TestCalculatorDivideByZeroErrors(t *testing.T) {
	_, err := runCalc(t, "1/0")
	if err == nil {
		t.Error("expected error for division by zero")
	}
}

func TestCalculatorModuloByZeroErrors(t *testing.T) {
	_, err := runCalc(t, "5%0")
	if err == nil {
		t.Error("expected error for modulo by zero")
	}
}

func TestCalculatorEmptyExpressionErrors(t *testing.T) {
	_, err := runCalc(t, "   ")
	if err == nil {
		t.Error("expected error for empty expression")
	}
}

func TestCalculatorUnaryMinus(t *testing.T) {
	out, err := runCalc(t, "-3+5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasSuffix(out, "= 2") {
		t.Errorf("expected '= 2' suffix, got %q", out)
	}
}

func TestCalculatorMismatchedParen(t *testing.T) {
	if _, err := runCalc(t, "(1+2"); err == nil {
		t.Error("expected error for mismatched (")
	}
	if _, err := runCalc(t, "1+2)"); err == nil {
		t.Error("expected error for mismatched )")
	}
}
