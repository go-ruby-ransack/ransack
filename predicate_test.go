// Copyright (c) 2026, the go-ruby-ransack/ransack authors
// All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

package ransack

import "testing"

// mp looks a predicate up from the registry and matches it.
func mp(t *testing.T, name string, field any, values ...any) bool {
	t.Helper()
	p, ok := registry[name]
	if !ok {
		t.Fatalf("unknown predicate %q", name)
	}
	return p.Match(field, values)
}

func TestScalarPredicates(t *testing.T) {
	if !mp(t, "eq", "x", "x") || mp(t, "eq", "x", "y") {
		t.Fatal("eq")
	}
	if !mp(t, "not_eq", "x", "y") || mp(t, "not_eq", "x", "x") {
		t.Fatal("not_eq")
	}
	if !mp(t, "lt", 1, 2) || mp(t, "lt", 2, 1) {
		t.Fatal("lt")
	}
	if !mp(t, "lteq", 2, 2) || mp(t, "lteq", 3, 2) {
		t.Fatal("lteq")
	}
	if !mp(t, "gt", 2, 1) || mp(t, "gt", 1, 2) {
		t.Fatal("gt")
	}
	if !mp(t, "gteq", 2, 2) || mp(t, "gteq", 1, 2) {
		t.Fatal("gteq")
	}
	if !mp(t, "cont", "foobar", "oob") || mp(t, "cont", "foobar", "zzz") {
		t.Fatal("cont")
	}
	if !mp(t, "not_cont", "foobar", "zzz") || mp(t, "not_cont", "foobar", "oob") {
		t.Fatal("not_cont")
	}
	if !mp(t, "i_cont", "FooBar", "oob") || mp(t, "i_cont", "FooBar", "zzz") {
		t.Fatal("i_cont")
	}
	if !mp(t, "not_i_cont", "FooBar", "zzz") || mp(t, "not_i_cont", "FooBar", "oob") {
		t.Fatal("not_i_cont")
	}
	if !mp(t, "start", "foobar", "foo") || mp(t, "start", "foobar", "bar") {
		t.Fatal("start")
	}
	if !mp(t, "not_start", "foobar", "bar") || mp(t, "not_start", "foobar", "foo") {
		t.Fatal("not_start")
	}
	if !mp(t, "end", "foobar", "bar") || mp(t, "end", "foobar", "foo") {
		t.Fatal("end")
	}
	if !mp(t, "not_end", "foobar", "foo") || mp(t, "not_end", "foobar", "bar") {
		t.Fatal("not_end")
	}
	if !mp(t, "matches", "foobar", "foo%") || mp(t, "matches", "foobar", "zzz%") {
		t.Fatal("matches")
	}
	if !mp(t, "does_not_match", "foobar", "zzz%") || mp(t, "does_not_match", "foobar", "foo%") {
		t.Fatal("does_not_match")
	}
}

func TestArrayPredicates(t *testing.T) {
	if !mp(t, "in", "b", "a", "b", "c") || mp(t, "in", "z", "a", "b") {
		t.Fatal("in")
	}
	if !mp(t, "not_in", "z", "a", "b") || mp(t, "not_in", "a", "a", "b") {
		t.Fatal("not_in")
	}
	if !mp(t, "cont_any", "foobar", "zzz", "oob") || mp(t, "cont_any", "foobar", "zzz", "qqq") {
		t.Fatal("cont_any")
	}
	if !mp(t, "cont_all", "foobar", "foo", "bar") || mp(t, "cont_all", "foobar", "foo", "zzz") {
		t.Fatal("cont_all")
	}
	if !mp(t, "eq_any", "b", "a", "b") || mp(t, "eq_any", "z", "a", "b") {
		t.Fatal("eq_any")
	}
	if !mp(t, "gteq_all", 10, 1, 5) || mp(t, "gteq_all", 3, 1, 5) {
		t.Fatal("gteq_all")
	}
}

func TestBooleanPredicates(t *testing.T) {
	// true: field truthiness must equal the flag.
	if !mp(t, "true", true, "1") || mp(t, "true", false, "1") {
		t.Fatal("true flag=1")
	}
	if !mp(t, "true", false, "0") {
		t.Fatal("true flag=0 selects falsy field")
	}
	if !mp(t, "not_true", false, "1") || mp(t, "not_true", true, "1") {
		t.Fatal("not_true")
	}
	if !mp(t, "false", false, "1") || mp(t, "false", true, "1") {
		t.Fatal("false")
	}
	if !mp(t, "not_false", true, "1") || mp(t, "not_false", false, "1") {
		t.Fatal("not_false")
	}
	if !mp(t, "null", nil, "1") || mp(t, "null", "x", "1") {
		t.Fatal("null flag=1")
	}
	if !mp(t, "null", "x", "0") {
		t.Fatal("null flag=0 selects non-nil")
	}
	if !mp(t, "not_null", "x", "1") || mp(t, "not_null", nil, "1") {
		t.Fatal("not_null")
	}
	if !mp(t, "present", "x", "1") || mp(t, "present", "", "1") {
		t.Fatal("present flag=1")
	}
	if !mp(t, "present", "", "0") {
		t.Fatal("present flag=0 selects blank")
	}
	if !mp(t, "blank", "", "1") || mp(t, "blank", "x", "1") {
		t.Fatal("blank")
	}
}

func TestPredicateOrdering(t *testing.T) {
	// The longest-first ordering must place does_not_match_any before shorter
	// predicates.
	if len(ordered) == 0 {
		t.Fatal("registry empty")
	}
	for i := 1; i < len(ordered); i++ {
		if len(ordered[i-1].Name) < len(ordered[i].Name) {
			t.Fatalf("ordering broken at %d", i)
		}
	}
}
