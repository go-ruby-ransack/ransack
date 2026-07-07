// Copyright (c) 2026, the go-ruby-ransack/ransack authors
// All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

package ransack

import (
	"fmt"
	"sort"
	"strings"
)

// reserved keys carry search structure rather than conditions.
var reserved = map[string]bool{
	"m": true, "g": true, "c": true,
	"s": true, "sorts": true,
	"distinct": true, "d": true,
}

// parseCondition parses a single "attribute_predicate" params entry. It returns
// (nil, nil) when the condition is dropped because its value is blank, a non-nil
// condition on success, or an error when no predicate/attribute combination is
// valid for ctx.
func parseCondition(ctx *Context, key string, raw any) (*Condition, error) {
	for _, p := range ordered {
		suffix := "_" + p.Name
		if !strings.HasSuffix(key, suffix) {
			continue
		}
		attrExpr := key[:len(key)-len(suffix)]
		if attrExpr == "" {
			continue
		}
		attrs, combi, ok := parseAttrs(ctx, attrExpr)
		if !ok {
			continue
		}
		values, skip := buildValues(p, raw)
		if skip {
			return nil, nil
		}
		return &Condition{
			Key:        key,
			Attributes: attrs,
			Combinator: combi,
			Predicate:  p,
			Values:     values,
		}, nil
	}
	return nil, fmt.Errorf("ransack: could not derive a predicate/attribute from %q", key)
}

// parseAttrs splits an attribute expression on the _or_ / _and_ combinator and
// resolves each part. It reports ok=false when any part is not allowed by ctx.
func parseAttrs(ctx *Context, expr string) ([]Attr, string, bool) {
	var parts []string
	var combi string
	switch {
	case strings.Contains(expr, "_or_"):
		parts = strings.Split(expr, "_or_")
		combi = "or"
	case strings.Contains(expr, "_and_"):
		parts = strings.Split(expr, "_and_")
		combi = "and"
	default:
		parts = []string{expr}
	}
	attrs := make([]Attr, 0, len(parts))
	for _, part := range parts {
		a, ok := resolveAttr(ctx, part)
		if !ok {
			return nil, "", false
		}
		attrs = append(attrs, a)
	}
	return attrs, combi, true
}

// resolveAttr walks association prefixes of part against ctx and validates the
// final column against the deepest context's allowlist.
func resolveAttr(ctx *Context, part string) (Attr, bool) {
	var path []string
	cur := ctx
	rem := part
	for {
		name, sub, ok := cur.matchAssoc(rem)
		if !ok {
			break
		}
		path = append(path, name)
		cur = sub
		rem = rem[len(name)+1:]
	}
	if !cur.allows(rem) {
		return Attr{}, false
	}
	return Attr{Path: path, Name: rem}, true
}

// buildValues derives a condition's value list from a raw params value and
// reports whether the condition should be skipped (dropped) because its value is
// blank. Boolean-flag predicates are never skipped.
func buildValues(p *Predicate, raw any) ([]any, bool) {
	if p.boolean {
		return []any{raw}, false
	}
	if p.WantsArray {
		list := filterBlanks(toSlice(raw))
		if len(list) == 0 {
			return nil, true
		}
		return list, false
	}
	if isBlankVal(raw) {
		return nil, true
	}
	return []any{raw}, false
}

// combinatorOf reads a group's "m" combinator, defaulting to "and".
func combinatorOf(m map[string]any) string {
	if v, ok := m["m"]; ok {
		if strings.ToLower(toString(v)) == "or" {
			return "or"
		}
	}
	return "and"
}

// asMap type-asserts a params value to a string-keyed map.
func asMap(v any) (map[string]any, bool) {
	m, ok := v.(map[string]any)
	return m, ok
}

// sortedKeys returns the keys of m in deterministic order.
func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// parseGroup builds a Group from a params map: non-reserved keys become
// conditions and the "g" key yields nested groups. Parse errors are collected
// and returned alongside the group.
func parseGroup(ctx *Context, m map[string]any) (*Group, []string) {
	g := &Group{Combinator: combinatorOf(m)}
	var errs []string
	for _, k := range sortedKeys(m) {
		if reserved[k] {
			continue
		}
		cond, err := parseCondition(ctx, k, m[k])
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		if cond != nil {
			g.Conditions = append(g.Conditions, cond)
		}
	}
	if gv, ok := m["g"]; ok {
		for _, sub := range toSlice(gv) {
			sm, ok := asMap(sub)
			if !ok {
				continue
			}
			sg, e := parseGroup(ctx, sm)
			g.Groups = append(g.Groups, sg)
			errs = append(errs, e...)
		}
	}
	return g, errs
}

// parseSorts parses an "s"/"sorts" value (a string or list of "column dir"
// strings) into Sort entries, validating each column against ctx.
func parseSorts(ctx *Context, raw any) ([]Sort, []string) {
	var sorts []Sort
	var errs []string
	for _, e := range toSlice(raw) {
		spec := strings.TrimSpace(toString(e))
		if spec == "" {
			continue
		}
		fields := strings.Fields(spec)
		name := fields[0]
		dir := "asc"
		if len(fields) > 1 && strings.ToLower(fields[1]) == "desc" {
			dir = "desc"
		}
		attr, ok := resolveAttr(ctx, name)
		if !ok {
			errs = append(errs, fmt.Sprintf("ransack: %q is not a sortable attribute", name))
			continue
		}
		sorts = append(sorts, Sort{Attr: attr, Name: name, Dir: dir})
	}
	return sorts, errs
}
