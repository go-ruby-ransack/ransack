// Copyright (c) 2026, the go-ruby-ransack/ransack authors
// All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

// Package ransack is a pure-Go (CGO=0), MRI-faithful reimplementation of the
// Ruby "ransack" gem: a search/filter query builder.
//
// A Ransack search is expressed as a "q" params hash whose keys encode an
// attribute and a predicate, for example:
//
//	q := map[string]any{
//		"name_cont":  "foo",   // name LIKE '%foo%'
//		"age_gteq":   18,       // age >= 18
//		"role_in":    []any{"admin", "editor"},
//		"s":          "name asc",
//	}
//	search := ransack.New(ctx, q)
//	results, _ := search.Result(&ransack.MemoryBackend{Records: rows})
//
// New parses the hash into a Search: a tree of conditions (Search.Root), a list
// of Sorts and a Distinct flag. Parsing strips the predicate suffix using the
// longest matching predicate, splits multi-attribute keys on the _or_ / _and_
// combinators, resolves association prefixes and validates every attribute
// against the Context allowlist (ActiveRecord's ransackable_attributes seam).
//
// Predicates include eq/not_eq, lt/lteq/gt/gteq, cont/not_cont/i_cont, start/
// end, matches/does_not_match, in/not_in, the boolean-flag predicates
// true/false/null/not_null/present/blank and the _any/_all compounds of every
// scalar predicate.
//
// Application is a seam. A Backend turns a Search into results; the rbgo/
// ActiveRecord binding implements it as SQL, while the built-in MemoryBackend
// (and Search.Match / Search.Apply) evaluate the search in memory so the engine
// is testable and usable without an ORM.
//
// The intended Ruby surface is Model.ransack(q).result, with predicate
// suffixes, s/sorts ordering, distinct and g[] grouped conditions all supported.
package ransack
