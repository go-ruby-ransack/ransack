// Copyright (c) 2026, the go-ruby-ransack/ransack authors
// All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

package ransack

// Attr is a searchable attribute reference. Path holds the association segments
// (empty for a direct attribute) and Name is the final column.
type Attr struct {
	Path []string
	Name string
}

// Condition is a parsed Ransack condition: one or more attributes tested with a
// single predicate against one or more values. When more than one attribute is
// present (the a_or_b / a_and_b grammar) Combinator ("or"/"and") joins them.
type Condition struct {
	// Key is the original params key that produced this condition.
	Key string
	// Attributes are the attributes tested by this condition.
	Attributes []Attr
	// Combinator joins multiple attributes ("or"/"and"); empty when there is
	// a single attribute.
	Combinator string
	// Predicate is the test applied to each attribute.
	Predicate *Predicate
	// Values are the (blank-filtered) condition values.
	Values []any
}

// Group is a tree node combining conditions and nested groups with a boolean
// combinator ("and"/"or").
type Group struct {
	Combinator string
	Conditions []*Condition
	Groups     []*Group
}

// Match evaluates the condition against a record.
func (c *Condition) Match(rec map[string]any) bool {
	if len(c.Attributes) == 1 {
		return c.matchAttr(rec, c.Attributes[0])
	}
	or := c.Combinator == "or"
	result := !or
	for _, a := range c.Attributes {
		m := c.matchAttr(rec, a)
		if or {
			result = result || m
		} else {
			result = result && m
		}
	}
	return result
}

// matchAttr resolves the candidate field values for attr (traversing
// associations, which may fan out over has-many collections) and reports
// whether any candidate satisfies the predicate — the in-memory analogue of an
// EXISTS join.
func (c *Condition) matchAttr(rec map[string]any, attr Attr) bool {
	candidates := gather(rec, attr.Path, attr.Name)
	if len(candidates) == 0 {
		return false
	}
	for _, cand := range candidates {
		if c.Predicate.Match(cand, c.Values) {
			return true
		}
	}
	return false
}

// Match evaluates the group against a record. An empty group matches every
// record.
func (g *Group) Match(rec map[string]any) bool {
	or := g.Combinator == "or"
	result := !or
	items := 0
	for _, c := range g.Conditions {
		items++
		m := c.Match(rec)
		if or {
			result = result || m
		} else {
			result = result && m
		}
	}
	for _, sub := range g.Groups {
		items++
		m := sub.Match(rec)
		if or {
			result = result || m
		} else {
			result = result && m
		}
	}
	if items == 0 {
		return true
	}
	return result
}

// gather returns every field value reachable by following the association path
// segs and reading the column name. Traversing a has-many collection fans the
// result out across its elements.
func gather(cur any, segs []string, name string) []any {
	if len(segs) == 0 {
		if m, ok := cur.(map[string]any); ok {
			return []any{m[name]}
		}
		return []any{nil}
	}
	var out []any
	for _, nxt := range descend(cur, segs[0]) {
		out = append(out, gather(nxt, segs[1:], name)...)
	}
	return out
}

// descend reads key from cur and flattens the result to a list of records.
func descend(cur any, key string) []any {
	m, ok := cur.(map[string]any)
	if !ok {
		return []any{nil}
	}
	return flatten(m[key])
}

// flatten normalises an association value to a list of associated records.
func flatten(v any) []any {
	switch t := v.(type) {
	case nil:
		return []any{nil}
	case []any:
		return t
	case []map[string]any:
		out := make([]any, len(t))
		for i := range t {
			out[i] = t[i]
		}
		return out
	default:
		return []any{v}
	}
}
