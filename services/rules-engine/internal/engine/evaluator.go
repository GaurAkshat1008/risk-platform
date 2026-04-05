// Package engine implements a JSON-condition-tree rule evaluator.
//
// Expression grammar:
//
//	Leaf:       { "field": "amount",   "op": "gt",  "value": 10000 }
//	Combinator: { "and": [ cond, ... ] }
//	            { "or":  [ cond, ... ] }
//	            { "not": cond          }
//
// Supported fields: amount, currency, source, destination, metadata.<key>
// Supported ops:    eq, neq, gt, gte, lt, lte, in, not_in, contains
package engine

import (
    "fmt"
    "strconv"
    "strings"
)

// PaymentContext is the data the evaluator reasons over.
type PaymentContext struct {
    PaymentEventID string
    TenantID       string
    Amount         int64
    Currency       string
    Source         string
    Destination    string
    Metadata       map[string]string
}

// EvalResult is the outcome of evaluating one rule against a context.
type EvalResult struct {
    Matched bool
    Reason  string
}

// Evaluate runs the expression tree against ctx and returns the result.
func Evaluate(expression map[string]any, ctx PaymentContext) (EvalResult, error) {
    matched, reason, err := evalNode(expression, ctx)
    if err != nil {
        return EvalResult{}, err
    }
    return EvalResult{Matched: matched, Reason: reason}, nil
}

func evalNode(node map[string]any, ctx PaymentContext) (bool, string, error) {
    if andRaw, ok := node["and"]; ok {
        return evalAnd(andRaw, ctx)
    }
    if orRaw, ok := node["or"]; ok {
        return evalOr(orRaw, ctx)
    }
    if notRaw, ok := node["not"]; ok {
        return evalNot(notRaw, ctx)
    }
    return evalLeaf(node, ctx)
}

func evalAnd(raw any, ctx PaymentContext) (bool, string, error) {
    conds, err := toNodeSlice(raw)
    if err != nil {
        return false, "", fmt.Errorf("and: %w", err)
    }
    for _, c := range conds {
        matched, reason, err := evalNode(c, ctx)
        if err != nil {
            return false, "", err
        }
        if !matched {
            return false, "and: failed at: " + reason, nil
        }
    }
    return true, "all conditions matched", nil
}

func evalOr(raw any, ctx PaymentContext) (bool, string, error) {
    conds, err := toNodeSlice(raw)
    if err != nil {
        return false, "", fmt.Errorf("or: %w", err)
    }
    for _, c := range conds {
        matched, reason, err := evalNode(c, ctx)
        if err != nil {
            return false, "", err
        }
        if matched {
            return true, reason, nil
        }
    }
    return false, "or: no conditions matched", nil
}

func evalNot(raw any, ctx PaymentContext) (bool, string, error) {
    condMap, ok := raw.(map[string]any)
    if !ok {
        return false, "", fmt.Errorf("not: expected object, got %T", raw)
    }
    matched, reason, err := evalNode(condMap, ctx)
    if err != nil {
        return false, "", err
    }
    return !matched, "not(" + reason + ")", nil
}

func toNodeSlice(raw any) ([]map[string]any, error) {
    slice, ok := raw.([]any)
    if !ok {
        return nil, fmt.Errorf("expected array, got %T", raw)
    }
    out := make([]map[string]any, 0, len(slice))
    for i, item := range slice {
        m, ok := item.(map[string]any)
        if !ok {
            return nil, fmt.Errorf("element %d: expected object, got %T", i, item)
        }
        out = append(out, m)
    }
    return out, nil
}

func evalLeaf(node map[string]any, ctx PaymentContext) (bool, string, error) {
    fieldRaw, ok := node["field"]
    if !ok {
        return false, "", fmt.Errorf("leaf missing 'field'")
    }
    field, ok := fieldRaw.(string)
    if !ok {
        return false, "", fmt.Errorf("'field' must be a string")
    }

    opRaw, ok := node["op"]
    if !ok {
        return false, "", fmt.Errorf("leaf missing 'op'")
    }
    op, ok := opRaw.(string)
    if !ok {
        return false, "", fmt.Errorf("'op' must be a string")
    }

    expected := node["value"]
    actual := resolveField(field, ctx)

    matched, err := applyOp(op, actual, expected)
    if err != nil {
        return false, "", fmt.Errorf("field %q op %q: %w", field, op, err)
    }

    reason := fmt.Sprintf("%s %s %v", field, op, expected)
    return matched, reason, nil
}

// resolveField returns the context value for a field path.
// Supports: amount, currency, source, destination, metadata.<key>
func resolveField(field string, ctx PaymentContext) any {
    if strings.HasPrefix(field, "metadata.") {
        key := strings.TrimPrefix(field, "metadata.")
        if ctx.Metadata != nil {
            return ctx.Metadata[key]
        }
        return ""
    }
    switch field {
    case "amount":
        return ctx.Amount
    case "currency":
        return ctx.Currency
    case "source":
        return ctx.Source
    case "destination":
        return ctx.Destination
    default:
        return nil
    }
}

func applyOp(op string, actual, expected any) (bool, error) {
    switch op {
    case "eq":
        return anyStr(actual) == anyStr(expected), nil
    case "neq":
        return anyStr(actual) != anyStr(expected), nil
    case "gt":
        a, e, err := toFloats(actual, expected)
        if err != nil {
            return false, err
        }
        return a > e, nil
    case "gte":
        a, e, err := toFloats(actual, expected)
        if err != nil {
            return false, err
        }
        return a >= e, nil
    case "lt":
        a, e, err := toFloats(actual, expected)
        if err != nil {
            return false, err
        }
        return a < e, nil
    case "lte":
        a, e, err := toFloats(actual, expected)
        if err != nil {
            return false, err
        }
        return a <= e, nil
    case "in":
        return inSlice(actual, expected)
    case "not_in":
        ok, err := inSlice(actual, expected)
        return !ok, err
    case "contains":
        return strings.Contains(
            strings.ToLower(anyStr(actual)),
            strings.ToLower(anyStr(expected)),
        ), nil
    default:
        return false, fmt.Errorf("unknown op %q", op)
    }
}

func anyStr(v any) string {
    if v == nil {
        return ""
    }
    switch t := v.(type) {
    case string:
        return t
    case float64:
        return strconv.FormatFloat(t, 'f', -1, 64)
    case int64:
        return strconv.FormatInt(t, 10)
    case bool:
        return strconv.FormatBool(t)
    default:
        return fmt.Sprintf("%v", v)
    }
}

func toFloat64(v any) (float64, error) {
    switch t := v.(type) {
    case float64:
        return t, nil
    case int64:
        return float64(t), nil
    case int:
        return float64(t), nil
    case string:
        f, err := strconv.ParseFloat(t, 64)
        if err != nil {
            return 0, fmt.Errorf("cannot parse %q as number", t)
        }
        return f, nil
    default:
        return 0, fmt.Errorf("cannot convert %T to float64", v)
    }
}

func toFloats(actual, expected any) (float64, float64, error) {
    a, err := toFloat64(actual)
    if err != nil {
        return 0, 0, fmt.Errorf("actual: %w", err)
    }
    e, err := toFloat64(expected)
    if err != nil {
        return 0, 0, fmt.Errorf("expected: %w", err)
    }
    return a, e, nil
}

func inSlice(actual, expected any) (bool, error) {
    slice, ok := expected.([]any)
    if !ok {
        return false, fmt.Errorf("in/not_in requires array value, got %T", expected)
    }
    actualStr := anyStr(actual)
    for _, item := range slice {
        if anyStr(item) == actualStr {
            return true, nil
        }
    }
    return false, nil
}