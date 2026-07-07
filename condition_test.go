// Copyright (c) 2026, the go-ruby-ransack/ransack authors
// All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

package ransack

import "testing"

func TestGatherScalarIntermediate(t *testing.T) {
	// Descending through a scalar intermediate yields a nil candidate rather
	// than panicking (exercises descend's non-map branch).
	rec := map[string]any{"a": "scalar"}
	got := gather(rec, []string{"a", "b"}, "c")
	if len(got) != 1 || got[0] != nil {
		t.Fatalf("gather scalar intermediate = %v", got)
	}
}

func TestFlattenVariants(t *testing.T) {
	if got := flatten(nil); len(got) != 1 || got[0] != nil {
		t.Fatal("flatten nil")
	}
	if got := flatten([]any{1, 2}); len(got) != 2 {
		t.Fatal("flatten []any")
	}
	if got := flatten([]map[string]any{{"x": 1}}); len(got) != 1 {
		t.Fatal("flatten []map")
	}
	if got := flatten("scalar"); len(got) != 1 {
		t.Fatal("flatten default")
	}
}

func TestFirstFieldMissingAssociation(t *testing.T) {
	// A sort over a has-many whose collection is empty resolves to nil.
	rec := map[string]any{"articles": []any{}}
	if firstField(rec, Attr{Path: []string{"articles"}, Name: "title"}) != nil {
		t.Fatal("expected nil for empty association")
	}
}

func TestMatchAssocNoMatch(t *testing.T) {
	// Associations present but none prefix-matches the remainder.
	ctx := NewContext("id").Associate("articles", NewContext())
	a, ok := resolveAttr(ctx, "id")
	if !ok || len(a.Path) != 0 || a.Name != "id" {
		t.Fatalf("resolve id: %+v %v", a, ok)
	}
	if _, _, ok := ctx.matchAssoc("id"); ok {
		t.Fatal("matchAssoc should not match id")
	}
}

func TestGroupAndNestedGroups(t *testing.T) {
	// AND-combined nested groups (exercises the group AND-combine branch).
	s := New(nil, map[string]any{
		"m": "and",
		"g": []any{
			map[string]any{"age_gteq": 20},
			map[string]any{"role_eq": "admin"},
		},
	})
	got := s.Apply(people())
	if len(got) != 2 { // Alice(30,admin) and Carol(42,admin)
		t.Fatalf("AND nested groups %v", names(got))
	}
}

func TestMatchAssocSortsByLength(t *testing.T) {
	// Two associations force the length-sorting comparator to run; the longer
	// name must win so that "articles_comments_x" binds "articles_comments".
	ctx := NewContext().
		Associate("articles", NewContext()).
		Associate("articles_comments", NewContext())
	name, _, ok := ctx.matchAssoc("articles_comments_body")
	if !ok || name != "articles_comments" {
		t.Fatalf("expected longest association, got %q %v", name, ok)
	}
}

func TestConditionAndCombinatorEval(t *testing.T) {
	// first_and_last both contain "o".
	c, _ := parseCondition(nil2ctx(), "first_and_last_cont", "o")
	if c.Combinator != "and" {
		t.Fatalf("combinator %s", c.Combinator)
	}
	rec := map[string]any{"first": "Bob", "last": "Jones"}
	if !c.Match(rec) {
		t.Fatal("expected AND match")
	}
	rec2 := map[string]any{"first": "Bob", "last": "Smith"} // last has no "o"
	if c.Match(rec2) {
		t.Fatal("expected AND non-match")
	}
}
