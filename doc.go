// Copyright (c) the go-ruby-kaminari/kaminari authors
//
// SPDX-License-Identifier: BSD-3-Clause

// Package kaminari is a pure-Go (CGO-free) reimplementation of the deterministic
// pagination core of Ruby's [kaminari] gem. It reproduces the `page(n).per(m).
// padding(p)` scope algebra — the offset/limit arithmetic and the page-metadata
// methods (`current_page`, `total_pages`, `total_count`, `limit_value`,
// `offset_value`, `first_page?`, `last_page?`, `out_of_range?`, `prev_page`,
// `next_page`) — together with kaminari's `PaginatableArray` (paginating an
// in-memory slice) and the pure data behind the `page_entries_info` and
// `paginate` view helpers.
//
// It is the pagination engine for
// [go-embedded-ruby](https://github.com/go-embedded-ruby/ruby), but a
// standalone, reusable module. A future rbgo binding wires it to
// `Model.page(n).per(m)` and `Array.page`.
//
// # What it is — and isn't
//
// Everything kaminari does to compute a page — clamping `per` (default 25, an
// optional `max_per_page` ceiling, `per(nil)` meaning "no limit"), applying
// `padding`, deriving `total_pages` with a ceiling division, clamping the
// requested page into range, and reporting the navigation metadata — is
// deterministic and needs no ActiveRecord. It lives here as pure Go.
//
// The [Array] paginator is fully implemented: it slices a Go `[]any` in memory
// exactly as `Kaminari.paginate_array(...).page(n).per(m)` slices a Ruby array.
//
// The relation paginator is a host seam. A [Relation] is any type exposing
// `Count() int` and `Slice(offset, limit int) any`; [Paginate] wraps it and
// applies the same scope algebra, calling `Slice` with the computed offset and
// limit (a limit of -1 means "no limit"). The rbgo binding implements [Relation]
// against an ActiveRecord relation (`limit` / `offset` / `count`). No SQL, no
// view rendering, and no Ruby runtime live in this package; view rendering is a
// downstream formatter's concern, fed by [EntriesInfo] and [NavConfig.Pages].
//
// [kaminari]: https://github.com/kaminari/kaminari
package kaminari
