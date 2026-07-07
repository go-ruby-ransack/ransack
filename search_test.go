// Copyright (c) 2026, the go-ruby-ransack/ransack authors
// All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

package ransack

import (
	"errors"
	"testing"
)

func people() []map[string]any {
	return []map[string]any{
		{"name": "Alice", "age": 30, "role": "admin", "active": true},
		{"name": "Bob", "age": 18, "role": "editor", "active": false},
		{"name": "Carol", "age": 42, "role": "admin", "active": true},
	}
}

func names(rs []map[string]any) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r["name"].(string)
	}
	return out
}

func eq(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}

func TestSearchContAndGteq(t *testing.T) {
	s := New(nil, map[string]any{"name_cont": "a", "age_gteq": 20})
	eq(t, names(s.Apply(people())), []string{"Carol"})
}

func TestSearchNilContextOpen(t *testing.T) {
	s := New(nil, map[string]any{"role_eq": "admin"})
	if len(s.Errors) != 0 {
		t.Fatalf("errors %v", s.Errors)
	}
	eq(t, names(s.Apply(people())), []string{"Alice", "Carol"})
}

func TestSearchSort(t *testing.T) {
	s := New(nil, map[string]any{"s": "age desc"})
	eq(t, names(s.Apply(people())), []string{"Carol", "Alice", "Bob"})
	s = New(nil, map[string]any{"sorts": []any{"age asc"}})
	eq(t, names(s.Apply(people())), []string{"Bob", "Alice", "Carol"})
}

func TestSearchSortStableTiebreak(t *testing.T) {
	rows := []map[string]any{
		{"name": "A", "role": "x"},
		{"name": "B", "role": "x"},
	}
	// Equal sort keys -> stable order preserved (exercises the c==0 continue and
	// the trailing false return).
	s := New(nil, map[string]any{"s": "role asc"})
	eq(t, names(s.Apply(rows)), []string{"A", "B"})
}

func TestSearchOrGroupCombinator(t *testing.T) {
	s := New(nil, map[string]any{"m": "or", "name_eq": "Bob", "role_eq": "admin"})
	eq(t, names(s.Apply(people())), []string{"Alice", "Bob", "Carol"})
}

func TestSearchGroupedConditions(t *testing.T) {
	// (age >= 40) OR (role == editor)
	s := New(nil, map[string]any{
		"m": "or",
		"g": []any{
			map[string]any{"age_gteq": 40},
			map[string]any{"role_eq": "editor"},
		},
	})
	eq(t, names(s.Apply(people())), []string{"Bob", "Carol"})
}

func TestSearchAttributeCombinator(t *testing.T) {
	// name OR role contains "edi" -> matches Bob (role editor).
	s := New(nil, map[string]any{"name_or_role_cont": "edi"})
	eq(t, names(s.Apply(people())), []string{"Bob"})
	// name AND role contain "a" -> Alice(name) has role admin (both contain a).
	s = New(nil, map[string]any{"name_and_role_cont": "a"})
	got := names(s.Apply(people()))
	// Alice: name has 'a'? "Alice" lower not relevant (cont is case sensitive),
	// "Alice" contains "a"? no (capital A). role "admin" contains "a" yes.
	// Carol: "Carol" contains "a" yes, "admin" yes -> match.
	eq(t, got, []string{"Carol"})
}

func TestSearchBooleanPredicate(t *testing.T) {
	s := New(nil, map[string]any{"active_true": "1"})
	eq(t, names(s.Apply(people())), []string{"Alice", "Carol"})
}

func TestSearchInPredicate(t *testing.T) {
	s := New(nil, map[string]any{"role_in": []any{"editor", "admin"}})
	eq(t, names(s.Apply(people())), []string{"Alice", "Bob", "Carol"})
}

func TestSearchDistinct(t *testing.T) {
	rows := []map[string]any{
		{"name": "A", "role": "x"},
		{"name": "A", "role": "x"},
		{"name": "B", "role": "y"},
	}
	s := New(nil, map[string]any{"distinct": true})
	eq(t, names(s.Apply(rows)), []string{"A", "B"})
	// "d" alias.
	s = New(nil, map[string]any{"d": "1"})
	eq(t, names(s.Apply(rows)), []string{"A", "B"})
}

func TestSearchMatch(t *testing.T) {
	s := New(nil, map[string]any{"age_gteq": 40})
	if !s.Match(map[string]any{"age": 41}) || s.Match(map[string]any{"age": 10}) {
		t.Fatal("Match")
	}
}

func TestSearchEmptyMatchesAll(t *testing.T) {
	s := New(nil, map[string]any{})
	eq(t, names(s.Apply(people())), []string{"Alice", "Bob", "Carol"})
}

func TestSearchAssociation(t *testing.T) {
	ctx := NewContext("name").Associate("articles", NewContext("title"))
	rows := []map[string]any{
		{"name": "Alice", "articles": []map[string]any{{"title": "Go rocks"}, {"title": "Ruby"}}},
		{"name": "Bob", "articles": []map[string]any{{"title": "Rust"}}},
		{"name": "Carol", "articles": []any{}}, // empty has-many
	}
	s := New(ctx, map[string]any{"articles_title_cont": "Ruby"})
	eq(t, names(s.Apply(rows)), []string{"Alice"})
	// Empty association yields no candidate -> no match.
	s = New(ctx, map[string]any{"articles_title_cont": "x"})
	eq(t, names(s.Apply(rows)), []string{})
}

func TestSearchBelongsToAssociation(t *testing.T) {
	ctx := NewContext("id").Associate("company", NewContext("name"))
	rows := []map[string]any{
		{"id": 1, "company": map[string]any{"name": "Acme"}},
		{"id": 2, "company": nil},
	}
	s := New(ctx, map[string]any{"company_name_eq": "Acme"})
	got := s.Apply(rows)
	if len(got) != 1 || got[0]["id"] != 1 {
		t.Fatalf("belongs-to %v", got)
	}
}

// errBackend is a Backend that always fails, exercising Result error
// propagation.
type errBackend struct{}

func (errBackend) Result(*Search) (any, error) { return nil, errors.New("boom") }

func TestResultBackendSeam(t *testing.T) {
	s := New(nil, map[string]any{"age_gteq": 20})
	res, err := s.Result(&MemoryBackend{Records: people()})
	if err != nil {
		t.Fatalf("memory backend err %v", err)
	}
	rows := res.([]map[string]any)
	eq(t, names(rows), []string{"Alice", "Carol"})

	if _, err := s.Result(errBackend{}); err == nil {
		t.Fatal("expected backend error")
	}
}

func TestSearchErrorsSurface(t *testing.T) {
	s := New(NewContext("name"), map[string]any{"secret_eq": "x", "s": "secret asc"})
	if len(s.Errors) != 2 {
		t.Fatalf("expected 2 errors, got %v", s.Errors)
	}
}
