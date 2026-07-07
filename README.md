<p align="center"><img src="https://go-ruby-kaminari.github.io/logo.png" alt="go-ruby-kaminari/kaminari" width="720"></p>

# kaminari — go-ruby-kaminari

[![Docs](https://img.shields.io/badge/docs-mkdocs--material-DC2626)](https://go-ruby-kaminari.github.io/docs/)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.26.4%2B-00ADD8)](https://go.dev/dl/)
[![Coverage](https://img.shields.io/badge/coverage-100%25-1a7f37)](#tests--coverage)

**A pure-Go (no cgo) reimplementation of the deterministic pagination core of
Ruby's [`kaminari`](https://github.com/kaminari/kaminari) gem.** It reproduces
the `page(n).per(m).padding(p)` scope algebra — the offset/limit arithmetic and
the page-metadata methods — kaminari's `PaginatableArray` for in-memory slices,
and the pure data behind the `page_entries_info` / `paginate` view helpers —
**without any Ruby runtime**.

It is the pagination engine for
[go-embedded-ruby](https://github.com/go-embedded-ruby/ruby), but a
**standalone, reusable** module. A future rbgo binding wires it to
`Model.page(n).per(m)` and `Array.page`.

> **What it is — and isn't.** Everything kaminari does to *compute* a page is
> deterministic and needs **no ActiveRecord**, so it lives here as pure Go:
> clamping `per` (default 25, an optional `max_per_page` ceiling, `per(nil)`
> meaning "no limit"), applying `padding`, deriving `total_pages` with a ceiling
> division, clamping the requested page into range, and reporting every
> navigation metadatum. The **in-memory `Array` paginator is fully
> implemented** — it slices a Go `[]any` exactly as
> `Kaminari.paginate_array(...)` slices a Ruby array. The **relation paginator
> is a host seam**: a `Relation` is any type exposing `Count() int` and
> `Slice(offset, limit int) any`; the rbgo binding implements it against an
> ActiveRecord relation (`limit` / `offset` / `count`). No SQL and no view
> rendering live here; rendering is a downstream formatter fed by `EntriesInfo`
> and `NavConfig.Pages`.

## Features

Faithful port of kaminari's scope algebra:

- **Scope algebra** — `Page(n)`, `Per(*int, maxPerPage...)`, `Padding(n)`, each
  returning a fresh immutable snapshot, exactly like kaminari's chainable
  scopes. `page(0)`/`page(-n)` clamp up to 1; `page` sets per-page to
  `default_per_page` (already clamped to `max_per_page`); `per` re-pages to
  preserve the current page number.
- **`per` clamping** — default `25`; capped by `max_per_page` (config or a
  per-call argument); `per(nil)` removes the limit (whole collection, one page);
  a negative `per` is ignored; `per(0)` is a degenerate limit-0 page.
- **Page metadata** — `CurrentPage`, `TotalPages` (ceiling division, clamped by
  `max_pages`), `TotalCount`, `LimitValue`, `OffsetValue`, `CurrentPerPage`,
  `FirstPage`, `LastPage`, `OutOfRange`, `PrevPage`, `NextPage` (the last two
  nil at the ends / out of range).
- **`PaginatableArray`** — `NewArray(items)` / `PaginateArray(items)` slices an
  in-memory `[]any`; `total_count` from the length or set explicitly.
- **View-helper data** — `EntriesInfo` (the numbers behind `page_entries_info`)
  and `RelevantPages` / `PageItems` / `NavConfig.Pages` (the window/outer-window
  page list, with truncation gaps, behind `paginate`).

## Usage

### In-memory array

```go
p := kaminari.NewArray(records).Page(2).Per(kaminari.Intp(10))

p.CurrentPage()  // 2
p.TotalPages()   // ceil(len/10)
p.OutOfRange()   // false
p.Records()      // []any window records[10:20]
```

`per(nil)` returns everything on a single page; `Padding` skips leading rows:

```go
kaminari.NewArray(records).Per(nil)              // all records, one page
kaminari.NewArray(records).Page(3).Per(kaminari.Intp(10)).Padding(2)
```

### Relation seam (wired to ActiveRecord by the rbgo binding)

```go
type Relation interface {
    Count() int                        // SELECT COUNT
    Slice(offset, limit int) any       // .offset(offset).limit(limit); limit -1 = no LIMIT
}

p := kaminari.Paginate(rel).Page(3).Per(kaminari.Intp(25))
p.TotalPages()
p.Records()  // rel.Slice(50, 25)
```

### Ruby surface (via the future rbgo binding)

```ruby
Model.page(3).per(25).padding(5)   # => relation seam
@users.total_pages
@users.current_page
@array.page(2).per(10)             # => PaginatableArray
page_entries_info(@users)          # => EntriesInfo
```

## Tests & coverage

`CGO_ENABLED=0`, stdlib-only, `go 1.26.4`. The suite is pure arithmetic — no
sockets, no filesystem — and holds **100 % statement coverage** (including the
out-of-range, `per(nil)`, `per(0)`, padding, `max_per_page` / `max_pages`
clamp, zero-count, and array-vs-relation-seam branches). CI runs the three host
OSes with gofmt + a 100 %-coverage gate, all six 64-bit arches (amd64 / arm64
native, riscv64 / loong64 / ppc64le / s390x under qemu), and both wasm targets
(`js` / `wasip1`).

```sh
go test -race -cover ./...
```

## License

BSD-3-Clause © the go-ruby-kaminari/kaminari authors. See [LICENSE](LICENSE).
