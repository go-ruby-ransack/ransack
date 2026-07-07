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

// Sort is a single ordering directive.
type Sort struct {
	// Attr is the resolved attribute to order by.
	Attr Attr
	// Name is the original column name from the params.
	Name string
	// Dir is "asc" or "desc".
	Dir string
}

// Search is the parsed form of a Ransack "q" params hash: a tree of conditions
// (Root), an ordered list of sorts, a distinct flag and any parse errors. It is
// the value the Ruby-facing Model.ransack(q) returns; .result is obtained by
// handing the Search to a Backend.
type Search struct {
	Root     *Group
	Sorts    []Sort
	Distinct bool
	Errors   []string
}

// New parses a params hash against ctx. A nil ctx is treated as an open context
// (every attribute allowed). Recognised structural keys are "m" (combinator),
// "g" (nested groups), "s"/"sorts" (ordering) and "distinct"/"d".
func New(ctx *Context, params map[string]any) *Search {
	if ctx == nil {
		ctx = NewContext()
	}
	s := &Search{}
	s.Root, s.Errors = parseGroup(ctx, params)
	for _, k := range []string{"s", "sorts"} {
		if v, ok := params[k]; ok {
			sorts, errs := parseSorts(ctx, v)
			s.Sorts = append(s.Sorts, sorts...)
			s.Errors = append(s.Errors, errs...)
		}
	}
	if v, ok := params["distinct"]; ok {
		s.Distinct = parseBool(v)
	} else if v, ok := params["d"]; ok {
		s.Distinct = parseBool(v)
	}
	return s
}

// Match reports whether a single record satisfies the search's condition tree.
func (s *Search) Match(rec map[string]any) bool {
	return s.Root.Match(rec)
}

// Apply is the built-in, ORM-free evaluator: it filters records by the condition
// tree, orders them by the sorts and, when Distinct is set, removes duplicate
// rows.
func (s *Search) Apply(records []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(records))
	for _, r := range records {
		if s.Root.Match(r) {
			out = append(out, r)
		}
	}
	out = s.sortRecords(out)
	if s.Distinct {
		out = distinctRecords(out)
	}
	return out
}

// Result hands the search to a Backend and returns whatever the backend
// produces. This is the seam the rbgo/ActiveRecord binding implements to build a
// real SQL relation; the in-memory MemoryBackend is the ORM-free default.
func (s *Search) Result(b Backend) (any, error) {
	return b.Result(s)
}

// sortRecords orders records by the search's sorts using a stable sort.
func (s *Search) sortRecords(records []map[string]any) []map[string]any {
	if len(s.Sorts) == 0 {
		return records
	}
	sort.SliceStable(records, func(i, j int) bool {
		for _, so := range s.Sorts {
			c := compareGeneric(firstField(records[i], so.Attr), firstField(records[j], so.Attr))
			if c == 0 {
				continue
			}
			if so.Dir == "desc" {
				return c > 0
			}
			return c < 0
		}
		return false
	})
	return records
}

// firstField returns the first candidate value for attr within rec.
func firstField(rec map[string]any, attr Attr) any {
	cands := gather(rec, attr.Path, attr.Name)
	if len(cands) > 0 {
		return cands[0]
	}
	return nil
}

// distinctRecords removes duplicate records, preserving order.
func distinctRecords(records []map[string]any) []map[string]any {
	seen := map[string]bool{}
	out := make([]map[string]any, 0, len(records))
	for _, r := range records {
		key := canonicalKey(r)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, r)
	}
	return out
}

// canonicalKey builds a deterministic identity string for a record.
func canonicalKey(rec map[string]any) string {
	keys := make([]string, 0, len(rec))
	for k := range rec {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "%s=%v;", k, rec[k])
	}
	return b.String()
}
