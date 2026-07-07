// Copyright (c) 2026, the go-ruby-ransack/ransack authors
// All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

package ransack

import (
	"sort"
	"strings"
)

// Predicate describes a Ransack search predicate, such as eq, cont or gteq. The
// exported fields let a Backend translate a condition to its own query language;
// Match evaluates the predicate in memory.
type Predicate struct {
	// Name is the predicate suffix, e.g. "cont", "not_eq" or "gteq_any".
	Name string
	// WantsArray reports whether the predicate consumes a list of values
	// (in/not_in and the _any/_all compounds).
	WantsArray bool

	base     string // underlying scalar matcher key
	compound string // "", "any" or "all"
	boolean  bool   // boolean-flag predicate (true/null/present/...)
}

// scalarBases are the predicates that operate on a single value and that also
// gain _any / _all compound variants.
var scalarBases = []string{
	"eq", "not_eq",
	"lt", "lteq", "gt", "gteq",
	"cont", "not_cont", "i_cont", "not_i_cont",
	"start", "not_start", "end", "not_end",
	"matches", "does_not_match",
}

// booleanBases are the predicates whose value is a boolean flag controlling the
// direction of the test.
var booleanBases = []string{
	"true", "not_true", "false", "not_false",
	"present", "blank", "null", "not_null",
}

// baseMatchers maps a scalar base predicate to its in-memory matcher.
var baseMatchers = map[string]func(field, val any) bool{
	"eq":             func(f, v any) bool { return equalValues(f, v) },
	"not_eq":         func(f, v any) bool { return !equalValues(f, v) },
	"lt":             func(f, v any) bool { return compareGeneric(f, v) < 0 },
	"lteq":           func(f, v any) bool { return compareGeneric(f, v) <= 0 },
	"gt":             func(f, v any) bool { return compareGeneric(f, v) > 0 },
	"gteq":           func(f, v any) bool { return compareGeneric(f, v) >= 0 },
	"cont":           func(f, v any) bool { return strings.Contains(toString(f), toString(v)) },
	"not_cont":       func(f, v any) bool { return !strings.Contains(toString(f), toString(v)) },
	"i_cont":         func(f, v any) bool { return strings.Contains(lower(f), lower(v)) },
	"not_i_cont":     func(f, v any) bool { return !strings.Contains(lower(f), lower(v)) },
	"start":          func(f, v any) bool { return strings.HasPrefix(toString(f), toString(v)) },
	"not_start":      func(f, v any) bool { return !strings.HasPrefix(toString(f), toString(v)) },
	"end":            func(f, v any) bool { return strings.HasSuffix(toString(f), toString(v)) },
	"not_end":        func(f, v any) bool { return !strings.HasSuffix(toString(f), toString(v)) },
	"matches":        func(f, v any) bool { return likeMatch(f, v) },
	"does_not_match": func(f, v any) bool { return !likeMatch(f, v) },
}

func lower(v any) string { return strings.ToLower(toString(v)) }

var (
	// registry maps a predicate name to its definition.
	registry = map[string]*Predicate{}
	// ordered lists predicates by descending name length so that the longest
	// suffix wins during detection.
	ordered []*Predicate
)

func register(p *Predicate) {
	registry[p.Name] = p
	ordered = append(ordered, p)
}

func init() {
	for _, base := range scalarBases {
		register(&Predicate{Name: base, base: base})
		register(&Predicate{Name: base + "_any", base: base, compound: "any", WantsArray: true})
		register(&Predicate{Name: base + "_all", base: base, compound: "all", WantsArray: true})
	}
	register(&Predicate{Name: "in", base: "in", WantsArray: true})
	register(&Predicate{Name: "not_in", base: "not_in", WantsArray: true})
	for _, base := range booleanBases {
		register(&Predicate{Name: base, base: base, boolean: true})
	}
	sort.SliceStable(ordered, func(i, j int) bool {
		return len(ordered[i].Name) > len(ordered[j].Name)
	})
}

// inList reports whether field equals any element of values.
func inList(field any, values []any) bool {
	for _, v := range values {
		if equalValues(field, v) {
			return true
		}
	}
	return false
}

// boolMatch evaluates a boolean-flag predicate. flag is the parsed truthiness of
// the supplied value and controls the direction of the test.
func boolMatch(base string, field any, flag bool) bool {
	switch base {
	case "true", "not_false":
		return parseBool(field) == flag
	case "false", "not_true":
		return parseBool(field) != flag
	case "null":
		return isNil(field) == flag
	case "not_null":
		return isNil(field) != flag
	case "present":
		return isPresent(field) == flag
	default: // "blank"
		return isBlankVal(field) == flag
	}
}

// Match evaluates the predicate against a single field value and its condition
// values. For array predicates every value participates; for boolean predicates
// only the first value (a flag) is consulted.
func (p *Predicate) Match(field any, values []any) bool {
	if p.boolean {
		return boolMatch(p.base, field, parseBool(values[0]))
	}
	switch p.base {
	case "in":
		return inList(field, values)
	case "not_in":
		return !inList(field, values)
	}
	f := baseMatchers[p.base]
	switch p.compound {
	case "any":
		for _, v := range values {
			if f(field, v) {
				return true
			}
		}
		return false
	case "all":
		for _, v := range values {
			if !f(field, v) {
				return false
			}
		}
		return true
	default:
		return f(field, values[0])
	}
}
