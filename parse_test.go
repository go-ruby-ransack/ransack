// Copyright (c) 2026, the go-ruby-ransack/ransack authors
// All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

package ransack

import "testing"

func TestParseConditionBasic(t *testing.T) {
	c, err := parseCondition(nil2ctx(), "name_cont", "foo")
	if err != nil || c == nil {
		t.Fatalf("parse: %v %v", c, err)
	}
	if c.Predicate.Name != "cont" || len(c.Attributes) != 1 || c.Attributes[0].Name != "name" {
		t.Fatalf("unexpected condition %+v", c)
	}
	if c.Key != "name_cont" || c.Values[0] != "foo" {
		t.Fatal("key/values")
	}
}

func nil2ctx() *Context { return NewContext() }

func TestParseConditionLongestPredicate(t *testing.T) {
	// name_not_cont must bind not_cont, not cont.
	c, _ := parseCondition(nil2ctx(), "name_not_cont", "x")
	if c.Predicate.Name != "not_cont" {
		t.Fatalf("got %s", c.Predicate.Name)
	}
}

func TestParseConditionArrayValue(t *testing.T) {
	c, _ := parseCondition(nil2ctx(), "role_in", []any{"a", "b"})
	if !c.Predicate.WantsArray || len(c.Values) != 2 {
		t.Fatalf("in values %+v", c.Values)
	}
}

func TestParseConditionBlankSkip(t *testing.T) {
	// Scalar blank value -> dropped (nil,nil).
	c, err := parseCondition(nil2ctx(), "name_cont", "")
	if c != nil || err != nil {
		t.Fatalf("blank scalar not skipped: %v %v", c, err)
	}
	// Array with only blank elements -> dropped.
	c, err = parseCondition(nil2ctx(), "role_in", []any{"", "  "})
	if c != nil || err != nil {
		t.Fatalf("blank array not skipped: %v %v", c, err)
	}
	// Boolean predicate with blank value -> NOT skipped.
	c, err = parseCondition(nil2ctx(), "active_null", "")
	if c == nil || err != nil {
		t.Fatalf("boolean blank wrongly skipped: %v %v", c, err)
	}
}

func TestParseConditionEmptyAttr(t *testing.T) {
	// Key equal to just a predicate suffix has an empty attribute -> unparseable.
	_, err := parseCondition(nil2ctx(), "_eq", "x")
	if err == nil {
		t.Fatal("expected error for empty attribute")
	}
}

func TestParseConditionNoPredicate(t *testing.T) {
	_, err := parseCondition(nil2ctx(), "bogus", "x")
	if err == nil {
		t.Fatal("expected error for missing predicate")
	}
}

func TestParseConditionAllowlistReject(t *testing.T) {
	ctx := NewContext("name") // age not allowed
	_, err := parseCondition(ctx, "age_gteq", 18)
	if err == nil {
		t.Fatal("expected allowlist rejection")
	}
	// Allowed attribute still parses.
	c, err := parseCondition(ctx, "name_eq", "z")
	if err != nil || c == nil {
		t.Fatalf("allowed attr rejected: %v", err)
	}
}

func TestParseAttrsCombinators(t *testing.T) {
	attrs, combi, ok := parseAttrs(nil2ctx(), "name_or_description")
	if !ok || combi != "or" || len(attrs) != 2 {
		t.Fatalf("or combinator: %v %v %v", attrs, combi, ok)
	}
	attrs, combi, ok = parseAttrs(nil2ctx(), "first_and_last")
	if !ok || combi != "and" || len(attrs) != 2 {
		t.Fatalf("and combinator: %v %v %v", attrs, combi, ok)
	}
	attrs, combi, ok = parseAttrs(nil2ctx(), "name")
	if !ok || combi != "" || len(attrs) != 1 {
		t.Fatalf("single: %v %v %v", attrs, combi, ok)
	}
	// One part not allowed -> whole thing rejected.
	ctx := NewContext("name")
	if _, _, ok := parseAttrs(ctx, "name_or_secret"); ok {
		t.Fatal("expected rejection when one combinator part is disallowed")
	}
}

func TestResolveAttrAssociation(t *testing.T) {
	ctx := NewContext("id").Associate("articles", NewContext("title"))
	a, ok := resolveAttr(ctx, "articles_title")
	if !ok || len(a.Path) != 1 || a.Path[0] != "articles" || a.Name != "title" {
		t.Fatalf("assoc resolve: %+v %v", a, ok)
	}
	// Column on the associated model not allowed.
	if _, ok := resolveAttr(ctx, "articles_body"); ok {
		t.Fatal("expected disallowed association column")
	}
	// Nested association.
	ctx2 := NewContext().Associate("articles", NewContext().Associate("comments", NewContext()))
	a, ok = resolveAttr(ctx2, "articles_comments_body")
	if !ok || len(a.Path) != 2 || a.Name != "body" {
		t.Fatalf("nested assoc: %+v %v", a, ok)
	}
}

func TestCombinatorOf(t *testing.T) {
	if combinatorOf(map[string]any{"m": "or"}) != "or" {
		t.Fatal("or")
	}
	if combinatorOf(map[string]any{"m": "and"}) != "and" {
		t.Fatal("and explicit")
	}
	if combinatorOf(map[string]any{}) != "and" {
		t.Fatal("default and")
	}
}

func TestParseGroupNested(t *testing.T) {
	params := map[string]any{
		"m":         "or",
		"name_cont": "foo",
		"g": []any{
			map[string]any{"m": "and", "age_gteq": 18, "age_lteq": 30},
			"not-a-map", // skipped
		},
	}
	g, errs := parseGroup(nil2ctx(), params)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors %v", errs)
	}
	if g.Combinator != "or" || len(g.Conditions) != 1 || len(g.Groups) != 1 {
		t.Fatalf("group shape %+v", g)
	}
	if g.Groups[0].Combinator != "and" || len(g.Groups[0].Conditions) != 2 {
		t.Fatalf("nested group %+v", g.Groups[0])
	}
}

func TestParseGroupCollectsErrors(t *testing.T) {
	_, errs := parseGroup(nil2ctx(), map[string]any{"bogus": "x"})
	if len(errs) != 1 {
		t.Fatalf("want 1 error, got %v", errs)
	}
}

func TestParseSorts(t *testing.T) {
	sorts, errs := parseSorts(nil2ctx(), []any{"name asc", "age desc", "  ", "id"})
	if len(errs) != 0 {
		t.Fatalf("errors %v", errs)
	}
	if len(sorts) != 3 {
		t.Fatalf("want 3 sorts, got %d", len(sorts))
	}
	if sorts[0].Dir != "asc" || sorts[1].Dir != "desc" || sorts[2].Dir != "asc" {
		t.Fatalf("dirs %+v", sorts)
	}
	// String form.
	sorts, _ = parseSorts(nil2ctx(), "name desc")
	if len(sorts) != 1 || sorts[0].Dir != "desc" {
		t.Fatalf("string sort %+v", sorts)
	}
	// Disallowed sort attribute.
	_, errs = parseSorts(NewContext("name"), "secret asc")
	if len(errs) != 1 {
		t.Fatalf("expected sort rejection, got %v", errs)
	}
}
