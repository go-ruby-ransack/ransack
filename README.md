<p align="center"><img src="https://go-ruby-ransack.github.io/logo.png" alt="go-ruby-ransack/ransack" width="720"></p>

# ransack — go-ruby-ransack

[![Docs](https://img.shields.io/badge/docs-mkdocs--material-DC2626)](https://go-ruby-ransack.github.io/docs/)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.27.1%2B-00ADD8)](https://go.dev/dl/)
[![Coverage](https://img.shields.io/badge/coverage-100%25-1a7f37)](#tests--coverage)

**A pure-Go (no cgo) reimplementation of the query-building core of Ruby's
[`ransack`](https://github.com/activerecord-hackery/ransack) gem** — the
search/filter object builder for ActiveRecord. It parses a `q` params hash into
a structured search AST (attributes + predicates + values, sorts, groups) and
evaluates it — **without any Ruby runtime**.

It is the search builder for
[go-embedded-ruby](https://github.com/go-embedded-ruby/ruby), but a
**standalone, reusable** module. Everything ransack does *before* it hits the
database is deterministic and needs no interpreter, so it lives here as pure Go:
stripping the predicate suffix off a key, splitting the `_or_` / `_and_`
attribute grammar, resolving association prefixes, validating attributes against
an allowlist, and building the condition tree and sort list.

> **What it is — and isn't.** Turning a search into a result set is a **host
> seam**: a `Backend` performs the query. The rbgo/ActiveRecord binding
> implements `Backend` by translating the condition tree into SQL
> `WHERE`/`JOIN`/`ORDER BY`; the built-in **`MemoryBackend`** (and
> `Search.Match` / `Search.Apply`) evaluate a search over in-memory records so
> the engine is testable and usable **without an ORM**.

## Features

Faithful port of the ransack query builder:

- **Search parsing** — `New(ctx, q)` parses a `q` params hash into a `Search`:
  a tree of conditions (`Root`), a list of `Sorts`, a `Distinct` flag and any
  parse `Errors`.
- **Predicate suffixes** — the full predicate set: `eq`/`not_eq`,
  `lt`/`lteq`/`gt`/`gteq`, `cont`/`not_cont`/`i_cont`/`not_i_cont`,
  `start`/`not_start`/`end`/`not_end`, `matches`/`does_not_match` (SQL `LIKE`),
  `in`/`not_in`, the boolean-flag predicates `true`/`not_true`/`false`/
  `not_false`/`null`/`not_null`/`present`/`blank`, and the `_any`/`_all`
  compounds of every scalar predicate. The **longest** matching suffix wins.
- **Attribute combinators** — the `name_or_description_cont` /
  `first_and_last_eq` grammar joins several attributes under one predicate.
- **Grouped conditions** — `m: 'or'` / `m: 'and'` combinators plus `g[]` nested
  groups, each with its own combinator.
- **Associations** — `assoc_attr_pred` keys (e.g. `articles_title_cont`,
  `articles_comments_body_eq`) resolve association prefixes against the context;
  the in-memory evaluator fans out over has-many collections (EXISTS semantics).
- **Sorts** — `s` / `sorts` (`"name asc"`, `["age desc", …]`) → `[]Sort` of
  `{attr, dir}`.
- **Allowlist seam** — `Context` mirrors `ransackable_attributes` /
  `ransackable_associations`: attributes and associations that are not
  whitelisted are rejected (recorded in `Search.Errors`). A nil context is
  "open".
- **Blank-value skip** — conditions whose value is blank are dropped, exactly
  as ransack does; boolean-flag predicates keep their `false` direction.
- **Application seam** — `Backend` (`Result(*Search)`); the built-in
  `MemoryBackend` filters, orders and de-duplicates in memory.

CGO-free, dependency-free (stdlib only), **100% test coverage**, `gofmt` +
`go vet` clean, and green across the six 64-bit Go targets (amd64, arm64,
riscv64, loong64, ppc64le, **s390x** — big-endian) plus `js/wasm` and
`wasip1/wasm`.

## Install

```sh
go get github.com/go-ruby-ransack/ransack
```

## Usage

```go
package main

import (
	"fmt"

	"github.com/go-ruby-ransack/ransack"
)

func main() {
	rows := []map[string]any{
		{"name": "Alice", "age": 30, "role": "admin"},
		{"name": "Bob", "age": 18, "role": "editor"},
		{"name": "Carol", "age": 42, "role": "admin"},
	}

	// Model.ransack(q).result — parse a q params hash, then evaluate it.
	q := map[string]any{
		"name_cont": "a",     // name LIKE '%a%'
		"age_gteq":  20,       // age >= 20
		"role_in":   []any{"admin"},
		"s":         "age desc",
	}

	search := ransack.New(nil, q) // nil context = open (allow every attribute)
	result, _ := search.Result(&ransack.MemoryBackend{Records: rows})
	for _, r := range result.([]map[string]any) {
		fmt.Println(r["name"]) // Carol
	}
}
```

### Allowlist and associations

```go
ctx := ransack.NewContext("name", "age").
	Associate("articles", ransack.NewContext("title"))

// articles_title_cont resolves the association; age_lteq is allowlisted;
// a disallowed key (e.g. password_eq) is recorded in search.Errors.
search := ransack.New(ctx, map[string]any{
	"articles_title_cont": "Ruby",
	"age_lteq":            65,
})
```

### Grouped conditions

```go
// (age >= 40) OR (role == "editor")
search := ransack.New(nil, map[string]any{
	"m": "or",
	"g": []any{
		map[string]any{"age_gteq": 40},
		map[string]any{"role_eq": "editor"},
	},
})
```

## Value model

| gem                                       | this package                                        |
| ----------------------------------------- | --------------------------------------------------- |
| `Model.ransack(q)`                        | `ransack.New(ctx, q)` → `*Search`                   |
| `search.result`                           | `search.Result(backend)` / `search.Apply(records)`  |
| `q[:name_cont] = "x"`                     | `Condition{Attributes, Predicate, Values}`          |
| `q[:name_or_desc_cont]`                   | `Condition.Combinator` over `Attributes`            |
| `q[:g][0][:m] = 'or'` grouped conditions  | `Group{Combinator, Conditions, Groups}`             |
| `q[:s] = "name asc"` / `q[:sorts]`        | `[]Sort{Attr, Dir}`                                 |
| `ransackable_attributes` / associations   | `Context{Attributes, Associations}` (allowlist)     |
| the ActiveRecord relation                 | `Backend` (host seam; `MemoryBackend` built in)     |

## Tests & coverage

The suite is pure and self-contained: every predicate class, the `_or_`/`_and_`
attribute grammar, `and`/`or` grouping, `g[]` nested groups, asc/desc sorts,
association attributes, `in`/multi-value predicates, unknown-predicate and
allowlist-reject paths, and blank-value skipping are all exercised, holding
coverage at **100%** on every arch/OS lane.

```sh
COVERPKG=$(go list ./... | paste -sd, -)
go test -race -coverpkg="$COVERPKG" -coverprofile=cover.out ./...
go tool cover -func=cover.out | tail -1   # 100.0%
```

## WebAssembly

Being pure Go (CGO=0), this library also compiles to **WebAssembly** — both
`GOOS=js GOARCH=wasm` (browser / Node.js) and `GOOS=wasip1 GOARCH=wasm` (WASI).
CI builds both targets on every push, alongside the six 64-bit native/qemu arches.

```sh
GOOS=js     GOARCH=wasm go build ./...   # browser / Node
GOOS=wasip1 GOARCH=wasm go build ./...   # WASI (wasmtime, wasmer, wasmedge, …)
```

## License

BSD-3-Clause — see [LICENSE](LICENSE). Copyright the go-ruby-ransack/ransack authors.
