// Package personalization implements the tiny condition DSL Atomic Site
// uses to gate per-block visibility on per-visitor metadata pushed back
// from a CRM. The grammar is intentionally minimal so the validator (Go,
// here) and the evaluator (JS, inlined in internal/builder/personalization.go)
// can stay in lock-step:
//
//	condition := key " " op " " value
//	op        := "==" | "!=" | ">" | ">=" | "<" | "<=" | "in" | "present"
//	key       := [A-Za-z_][A-Za-z0-9_]*
//	value     := bare_word | quoted_string | comma_list
//
// "present" takes no value: the condition matches if the key exists in
// the visitor map and is non-empty. "in" takes a comma-separated list:
// "lifecycle in MQL,SQL,Customer". Everything else is a binary op.
//
// One condition per block. AND is achieved by stacking multiple
// conditional blocks; OR is not supported (use multiple stacked blocks
// or duplicate content). This is deliberate: an expression DSL with
// nesting, parens, and short-circuit semantics is a tarpit, and we
// have ~4 customers for personalization at this stage.
package personalization

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	keyRE     = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	presentRE = regexp.MustCompile(`^\s*([A-Za-z_][A-Za-z0-9_]*)\s+(present)\s*$`)
	binaryRE  = regexp.MustCompile(`^\s*([A-Za-z_][A-Za-z0-9_]*)\s+(==|!=|>=|<=|>|<|in)\s+(.+?)\s*$`)
)

// Validate returns nil when expr is a syntactically valid condition or an
// error describing the parse failure. Rejected: unknown operators, bare
// values without a key, semicolon-separated chains, JS expressions, etc.
func Validate(expr string) error {
	if strings.TrimSpace(expr) == "" {
		return errors.New("empty condition")
	}
	// Reject obvious script attempts up front. These would still parse as
	// "key op value" with the rest going into value, but it's cheaper to
	// reject early than to chase the symptom.
	for _, bad := range []string{";", "{", "}", "()", "/*", "*/", "<script", "javascript:"} {
		if strings.Contains(strings.ToLower(expr), bad) {
			return fmt.Errorf("forbidden token in condition: %q", bad)
		}
	}
	if presentRE.MatchString(expr) {
		return nil
	}
	m := binaryRE.FindStringSubmatch(expr)
	if m == nil {
		return errors.New("expected: key op value (or: key present)")
	}
	if !keyRE.MatchString(m[1]) {
		return fmt.Errorf("invalid key: %q", m[1])
	}
	value := strings.TrimSpace(m[3])
	if value == "" {
		return errors.New("missing value")
	}
	return nil
}

// Eval returns true when the condition matches the given variables.
// Validation errors propagate; missing keys evaluate to false (except
// "present" which is the explicit existence test).
func Eval(expr string, vars map[string]any) (bool, error) {
	if err := Validate(expr); err != nil {
		return false, err
	}
	if m := presentRE.FindStringSubmatch(expr); m != nil {
		v, ok := vars[m[1]]
		if !ok {
			return false, nil
		}
		return !isEmpty(v), nil
	}
	m := binaryRE.FindStringSubmatch(expr)
	if m == nil {
		return false, errors.New("unparsable condition") // unreachable: Validate caught it
	}
	key, op, raw := m[1], m[2], strings.TrimSpace(m[3])
	if len(raw) >= 2 && raw[0] == '"' && raw[len(raw)-1] == '"' {
		raw = raw[1 : len(raw)-1]
	}
	lhs, present := vars[key]
	if !present {
		return false, nil
	}
	if op == "in" {
		needle := toString(lhs)
		for _, candidate := range strings.Split(raw, ",") {
			if strings.TrimSpace(candidate) == needle {
				return true, nil
			}
		}
		return false, nil
	}
	ln, lOK := toNumber(lhs)
	rn, rOK := toNumber(raw)
	numeric := lOK && rOK
	switch op {
	case "==":
		if numeric {
			return ln == rn, nil
		}
		return toString(lhs) == raw, nil
	case "!=":
		if numeric {
			return ln != rn, nil
		}
		return toString(lhs) != raw, nil
	case ">":
		return numeric && ln > rn, nil
	case ">=":
		return numeric && ln >= rn, nil
	case "<":
		return numeric && ln < rn, nil
	case "<=":
		return numeric && ln <= rn, nil
	}
	return false, fmt.Errorf("unknown op %q", op)
}

func isEmpty(v any) bool {
	if v == nil {
		return true
	}
	switch x := v.(type) {
	case string:
		return x == ""
	case bool:
		return !x
	case float64:
		return x == 0
	}
	return false
}

func toString(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	case bool:
		if x {
			return "true"
		}
		return "false"
	case nil:
		return ""
	}
	return fmt.Sprintf("%v", v)
}

func toNumber(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case string:
		n, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
		if err != nil {
			return 0, false
		}
		return n, true
	}
	return 0, false
}
