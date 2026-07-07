// Copyright (c) the go-ruby-kaminari/kaminari authors
//
// SPDX-License-Identifier: BSD-3-Clause

package kaminari

// Array is kaminari's PaginatableArray: an in-memory collection that paginates
// itself. It mirrors `Kaminari.paginate_array(items).page(n).per(m).padding(p)`
// — the underlying slice is sliced by the computed window, and every page
// metadata method (promoted from the embedded scope) reports against it.
//
// An Array is an immutable snapshot: Page, Per, and Padding return a new Array
// sharing the same backing items, so a base paginator can be reused to derive
// several pages. Records materialises the current window.
type Array struct {
	scope
	items []any
}

// ArrayOption customises the initial [Array] built by [NewArray] / [PaginateArray].
type ArrayOption func(*arrayOpts)

type arrayOpts struct {
	cfg        Config
	totalCount *int
	limit      *int
	offset     int
	padding    int
}

// WithTotalCount overrides total_count, matching
// `Kaminari.paginate_array(items, total_count: n)`. When unset, total_count is
// len(items).
func WithTotalCount(n int) ArrayOption { return func(o *arrayOpts) { v := n; o.totalCount = &v } }

// WithLimit seeds the initial limit (per-page), matching the `:limit` option.
func WithLimit(n int) ArrayOption { return func(o *arrayOpts) { v := n; o.limit = &v } }

// WithOffset seeds the initial base offset, matching the `:offset` option.
func WithOffset(n int) ArrayOption { return func(o *arrayOpts) { o.offset = n } }

// WithPadding seeds the initial padding, matching the `:padding` option.
func WithPadding(n int) ArrayOption { return func(o *arrayOpts) { o.padding = n } }

// WithConfig supplies the pagination [Config] (default_per_page / max_per_page /
// max_pages) for the paginator.
func WithConfig(c Config) ArrayOption { return func(o *arrayOpts) { o.cfg = c } }

// NewArray builds a PaginatableArray over items. Without options it behaves like
// a bare `Kaminari.paginate_array(items)`: total_count = len(items), per =
// default_per_page, offset 0.
func NewArray(items []any, opts ...ArrayOption) *Array {
	o := arrayOpts{cfg: NewConfig()}
	for _, opt := range opts {
		opt(&o)
	}
	o.cfg.normalize()
	if items == nil {
		items = []any{}
	}
	total := len(items)
	if o.totalCount != nil {
		total = *o.totalCount
	}
	s := newScope(total, o.cfg)
	if o.limit != nil {
		s.limit = *o.limit
		s.perSet = *o.limit
	}
	s.offset = o.offset
	s.pad = o.padding
	return &Array{scope: s, items: items}
}

// PaginateArray is an alias for [NewArray] that reads like kaminari's
// module-level `Kaminari.paginate_array(...)`.
func PaginateArray(items []any, opts ...ArrayOption) *Array { return NewArray(items, opts...) }

// Page returns the Array positioned on page num (`.page(num)`), clamping num up
// to 1.
func (a *Array) Page(num int) *Array {
	return &Array{scope: a.scope.page(num), items: a.items}
}

// Per returns the Array with per-page set to num (`.per(num)`); a nil num means
// "no limit". An optional maxPerPage overrides the configured ceiling.
func (a *Array) Per(num *int, maxPerPage ...int) *Array {
	return &Array{scope: a.scope.per(num, maxPerPage...), items: a.items}
}

// Padding returns the Array with padding set to num (`.padding(num)`).
func (a *Array) Padding(num int) *Array {
	return &Array{scope: a.scope.padding(num), items: a.items}
}

// Records materialises the current page: the window of the backing slice
// selected by offset+padding and limit. It follows Ruby's `array[start, len]`
// semantics — an out-of-range start yields an empty slice, and the window is
// clipped at the end of the backing slice. A returned slice is always a fresh
// copy, never an alias of the backing array.
func (a *Array) Records() []any {
	start := a.scope.sliceOffset()
	n := len(a.items)
	if start < 0 || start >= n {
		return []any{}
	}
	end := n
	if !a.scope.unlimited {
		if a.scope.limit <= 0 {
			return []any{}
		}
		end = start + a.scope.limit
		if end > n {
			end = n
		}
	}
	out := make([]any, end-start)
	copy(out, a.items[start:end])
	return out
}
