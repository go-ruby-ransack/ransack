// Copyright (c) 2026, the go-ruby-ransack/ransack authors
// All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

package ransack

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

// toFloat coerces v to a float64 when it holds a numeric value (any Go integer
// or float kind, or a numeric string). The second result reports success.
func toFloat(v any) (float64, bool) {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(rv.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(rv.Uint()), true
	case reflect.Float32, reflect.Float64:
		return rv.Float(), true
	case reflect.String:
		f, err := strconv.ParseFloat(strings.TrimSpace(rv.String()), 64)
		if err != nil {
			return 0, false
		}
		return f, true
	default:
		return 0, false
	}
}

// toString renders v as a string, treating nil as the empty string.
func toString(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

// numCompare compares a and b numerically when both coerce to numbers. It
// returns -1, 0 or 1 and reports whether a numeric comparison was possible.
func numCompare(a, b any) (int, bool) {
	fa, ok1 := toFloat(a)
	fb, ok2 := toFloat(b)
	if !ok1 || !ok2 {
		return 0, false
	}
	switch {
	case fa < fb:
		return -1, true
	case fa > fb:
		return 1, true
	default:
		return 0, true
	}
}

// compareGeneric orders a and b, preferring a numeric comparison and falling
// back to lexical string comparison.
func compareGeneric(a, b any) int {
	if c, ok := numCompare(a, b); ok {
		return c
	}
	sa, sb := toString(a), toString(b)
	switch {
	case sa < sb:
		return -1
	case sa > sb:
		return 1
	default:
		return 0
	}
}

// equalValues reports whether a and b are equal, numerically when possible and
// lexically otherwise.
func equalValues(a, b any) bool {
	if c, ok := numCompare(a, b); ok {
		return c == 0
	}
	return toString(a) == toString(b)
}

// isNil reports whether v is nil.
func isNil(v any) bool {
	return v == nil
}

// isBlankVal mirrors Ruby's Object#blank? for the values this engine handles:
// nil, whitespace-only strings and empty slices are blank.
func isBlankVal(v any) bool {
	switch t := v.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(t) == ""
	case []any:
		return len(t) == 0
	default:
		return false
	}
}

// isPresent is the negation of isBlankVal.
func isPresent(v any) bool {
	return !isBlankVal(v)
}

// parseBool interprets v as a Ruby-style boolean flag. Recognised truthy string
// tokens follow ActiveRecord's TRUE_VALUES; numbers are truthy when non-zero.
func parseBool(v any) bool {
	switch t := v.(type) {
	case bool:
		return t
	case nil:
		return false
	case string:
		switch strings.ToLower(strings.TrimSpace(t)) {
		case "1", "true", "t", "yes", "y", "on":
			return true
		default:
			return false
		}
	default:
		if f, ok := toFloat(v); ok {
			return f != 0
		}
		return false
	}
}

// toSlice normalises v to a []any: existing slices are expanded element-wise and
// scalars are wrapped in a single-element slice.
func toSlice(v any) []any {
	rv := reflect.ValueOf(v)
	if rv.Kind() == reflect.Slice {
		out := make([]any, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			out[i] = rv.Index(i).Interface()
		}
		return out
	}
	return []any{v}
}

// filterBlanks drops blank elements from vs.
func filterBlanks(vs []any) []any {
	out := make([]any, 0, len(vs))
	for _, v := range vs {
		if !isBlankVal(v) {
			out = append(out, v)
		}
	}
	return out
}

// likeMatch evaluates a SQL LIKE pattern (val) against field. '%' matches any
// run of characters and '_' matches a single character; every other rune is
// matched literally. The match is anchored.
func likeMatch(field, val any) bool {
	pattern := toString(val)
	var b strings.Builder
	b.WriteString("^")
	for _, r := range pattern {
		switch r {
		case '%':
			b.WriteString(".*")
		case '_':
			b.WriteString(".")
		default:
			b.WriteString(regexp.QuoteMeta(string(r)))
		}
	}
	b.WriteString("$")
	re := regexp.MustCompile(b.String())
	return re.MatchString(toString(field))
}
