// Copyright (c) 2026, the go-ruby-ransack/ransack authors
// All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

package ransack

// Backend is the application seam. A Search describes what to fetch; a Backend
// turns that description into results. The rbgo/ActiveRecord binding implements
// Backend by translating the condition tree into SQL WHERE/JOIN clauses and the
// sorts into ORDER BY. MemoryBackend is the built-in, ORM-free implementation.
type Backend interface {
	// Result evaluates the search and returns its result set.
	Result(s *Search) (any, error)
}

// MemoryBackend evaluates a Search over an in-memory slice of records using the
// built-in evaluator. Each record is a column-name -> value map; associations
// are nested maps (belongs-to) or slices of maps (has-many).
type MemoryBackend struct {
	Records []map[string]any
}

// Result filters, sorts and de-duplicates the backing records per the search.
func (b *MemoryBackend) Result(s *Search) (any, error) {
	return s.Apply(b.Records), nil
}
