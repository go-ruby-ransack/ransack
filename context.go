// Copyright (c) 2026, the go-ruby-ransack/ransack authors
// All rights reserved.
//
// SPDX-License-Identifier: BSD-3-Clause

package ransack

import (
	"sort"
	"strings"
)

// Context is the allowlist seam. It mirrors ActiveRecord's
// ransackable_attributes / ransackable_associations: only whitelisted
// attributes and associations may be searched or sorted. This keeps a search
// engine safe when it is driven by untrusted params.
//
// A Context with a nil Attributes slice is "open" and allows every attribute
// name. Associations is a map from association name to the Context describing
// that associated model.
type Context struct {
	// Attributes is the set of searchable/sortable columns. Nil means open.
	Attributes []string
	// Associations maps an association name to its own Context.
	Associations map[string]*Context
}

// NewContext returns a Context allowing the given attributes. With no arguments
// it returns an open Context that allows every attribute.
func NewContext(attrs ...string) *Context {
	c := &Context{}
	if len(attrs) > 0 {
		c.Attributes = attrs
	}
	return c
}

// Associate registers an association and returns the receiver for chaining.
func (c *Context) Associate(name string, sub *Context) *Context {
	if c.Associations == nil {
		c.Associations = map[string]*Context{}
	}
	c.Associations[name] = sub
	return c
}

// allows reports whether name is a searchable attribute in this context.
func (c *Context) allows(name string) bool {
	if len(c.Attributes) == 0 {
		return true
	}
	for _, a := range c.Attributes {
		if a == name {
			return true
		}
	}
	return false
}

// matchAssoc greedily matches the longest registered association name that
// prefixes rem (followed by an underscore). It returns the association name, its
// sub-context and whether a match was found.
func (c *Context) matchAssoc(rem string) (string, *Context, bool) {
	if c.Associations == nil {
		return "", nil, false
	}
	names := make([]string, 0, len(c.Associations))
	for name := range c.Associations {
		names = append(names, name)
	}
	sort.SliceStable(names, func(i, j int) bool { return len(names[i]) > len(names[j]) })
	for _, name := range names {
		if strings.HasPrefix(rem, name+"_") {
			return name, c.Associations[name], true
		}
	}
	return "", nil, false
}
