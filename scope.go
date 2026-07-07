// Copyright (c) the go-ruby-kaminari/kaminari authors
//
// SPDX-License-Identifier: BSD-3-Clause

package kaminari

// scope is the deterministic heart of kaminari's `PageScopeMethods`: the
// offset/limit state produced by chaining `page`, `per`, and `padding`, plus the
// arithmetic behind every page-metadata method. Both the in-memory [Array]
// paginator and the seam-backed [Relation] paginator embed a scope by value, so
// each `page/per/padding` step yields a fresh, independent scope (kaminari's
// scopes are likewise immutable snapshots).
//
// offset is the *base* page offset — `limit * (page-1)` — and does not include
// padding. padding is carried separately; the data-source offset the paginators
// pass to their backend is offset+padding (see [scope.sliceOffset]). Keeping the
// base offset padding-free is what lets current_page stay padding-agnostic while
// the slice still skips the padded rows, reproducing kaminari's add-to-offset /
// subtract-in-current_page dance without its double bookkeeping.
type scope struct {
	limit     int  // per-page count; meaningless (0) while unlimited
	unlimited bool // true after per(nil): no LIMIT, everything on one page
	offset    int  // base page offset, excludes padding
	pad       int  // rows skipped ahead of the page window
	total     int  // total_count of the underlying collection
	perSet    int  // @_per: the raw value last requested via per (or default)
	cfg       Config
}

// newScope builds the initial scope for a collection of the given total size,
// before any page/per/padding, matching a bare relation: offset 0, per =
// default_per_page.
func newScope(total int, cfg Config) scope {
	cfg.normalize()
	return scope{
		limit:  cfg.DefaultPerPage,
		offset: 0,
		total:  total,
		perSet: cfg.DefaultPerPage,
		cfg:    cfg,
	}
}

// page reproduces kaminari's `.page(num)` class method: per-page is
// default_per_page (already clamped to max_per_page when default exceeds it),
// and the offset is per_page*(num-1) with num clamped up to 1.
func (s scope) page(num int) scope {
	perPage := s.cfg.DefaultPerPage
	if s.cfg.MaxPerPage != nil && perPage > *s.cfg.MaxPerPage {
		perPage = *s.cfg.MaxPerPage
	}
	if num < 1 {
		num = 1
	}
	s.unlimited = false
	s.limit = perPage
	s.perSet = perPage
	s.offset = perPage * (num - 1)
	return s
}

// per reproduces kaminari's `.per(num, max_per_page:)`. A nil num means "no
// limit" (the whole collection on a single page). A negative num is ignored
// (the scope is returned unchanged). Zero yields a degenerate limit-0 page.
// Otherwise the new per-page is clamped to the effective max_per_page, and the
// offset is recomputed as (offset/limit)*new_per so the current page number is
// preserved across the change.
func (s scope) per(num *int, maxPerPage ...int) scope {
	if num == nil {
		// @_per becomes (nil || default) => default_per_page.
		s.perSet = s.cfg.DefaultPerPage
		s.unlimited = true
		s.offset = 0
		return s
	}
	n := *num
	// @_per records the *requested* value, even when it is later clamped or
	// rejected — this is what current_per_page reports.
	s.perSet = n
	if n < 0 {
		return s
	}
	s.unlimited = false
	if n == 0 {
		s.limit = 0
		return s
	}
	max := s.effectiveMax(maxPerPage)
	if max != nil && *max < n {
		n = *max
	}
	page := 0
	if s.limit > 0 {
		page = s.offset / s.limit
	}
	s.offset = page * n
	s.limit = n
	return s
}

// effectiveMax resolves the max_per_page ceiling: an explicit per-call argument
// wins, else the configured Config.MaxPerPage, else nil (no ceiling).
func (s scope) effectiveMax(perCall []int) *int {
	if len(perCall) > 0 {
		return &perCall[0]
	}
	return s.cfg.MaxPerPage
}

// padding reproduces kaminari's `.padding(num)`: it records the padding and
// shifts the data-source offset forward by num, while leaving the base page
// offset (and thus current_page) untouched.
func (s scope) padding(num int) scope {
	s.pad = num
	return s
}

// adjustedTotal is total_count minus padding, floored at zero — the count that
// feeds total_pages.
func (s scope) adjustedTotal() int {
	a := s.total - s.pad
	if a < 0 {
		a = 0
	}
	return a
}

// sliceOffset is the offset handed to the data source: the base page offset plus
// padding.
func (s scope) sliceOffset() int { return s.offset + s.pad }

// sliceLimit is the limit handed to the data source, or -1 when unlimited.
func (s scope) sliceLimit() int {
	if s.unlimited {
		return -1
	}
	return s.limit
}

// TotalCount returns the size of the underlying collection (kaminari's
// total_count).
func (s scope) TotalCount() int { return s.total }

// LimitValue reproduces `limit_value`: the per-page count. While unlimited it is
// the number of rows the single page actually yields (adjustedTotal).
func (s scope) LimitValue() int {
	if s.unlimited {
		return s.adjustedTotal()
	}
	return s.limit
}

// OffsetValue reproduces `offset_value`: the data-source offset (base+padding).
func (s scope) OffsetValue() int { return s.sliceOffset() }

// CurrentPerPage reproduces `current_per_page`: the last requested per value,
// unclamped, defaulting to default_per_page.
func (s scope) CurrentPerPage() int { return s.perSet }

// CurrentPage reproduces `current_page`. Unlimited collapses to page 1. A
// degenerate limit-0 page (kaminari raises ZeroPerPageOperation there) reports 0
// rather than panicking, so the metadata getters stay total functions.
func (s scope) CurrentPage() int {
	if s.unlimited {
		return 1
	}
	if s.limit <= 0 {
		return 0
	}
	return s.offset/s.limit + 1
}

// TotalPages reproduces `total_pages`: ceil(adjustedTotal/limit), clamped down
// to max_pages when configured. Unlimited is a single page (zero when empty).
func (s scope) TotalPages() int {
	if s.unlimited {
		if s.adjustedTotal() > 0 {
			return 1
		}
		return 0
	}
	if s.limit <= 0 {
		return 0
	}
	tp := ceilDiv(s.adjustedTotal(), s.limit)
	if s.cfg.MaxPages != nil && *s.cfg.MaxPages < tp {
		return *s.cfg.MaxPages
	}
	return tp
}

// FirstPage reproduces `first_page?`.
func (s scope) FirstPage() bool { return s.CurrentPage() == 1 }

// LastPage reproduces `last_page?`.
func (s scope) LastPage() bool { return s.CurrentPage() == s.TotalPages() }

// OutOfRange reproduces `out_of_range?`.
func (s scope) OutOfRange() bool { return s.CurrentPage() > s.TotalPages() }

// PrevPage reproduces `prev_page`: the previous page number, or nil on the first
// page or when out of range.
func (s scope) PrevPage() *int {
	if s.FirstPage() || s.OutOfRange() {
		return nil
	}
	v := s.CurrentPage() - 1
	return &v
}

// NextPage reproduces `next_page`: the next page number, or nil on the last page
// or when out of range.
func (s scope) NextPage() *int {
	if s.LastPage() || s.OutOfRange() {
		return nil
	}
	v := s.CurrentPage() + 1
	return &v
}

// EntriesInfo reproduces the pure data behind `page_entries_info`.
func (s scope) EntriesInfo() EntriesInfo {
	tc := s.total
	if tc == 0 {
		return EntriesInfo{TotalCount: 0, Empty: true}
	}
	if s.TotalPages() < 2 {
		return EntriesInfo{TotalCount: tc, OnePage: true, First: 1, Last: tc}
	}
	off := s.OffsetValue()
	info := EntriesInfo{TotalCount: tc, First: off + 1}
	if s.LastPage() {
		info.Last = tc
	} else {
		info.Last = off + s.limit
	}
	return info
}

// ceilDiv returns ceil(a/b) for a >= 0 and b > 0.
func ceilDiv(a, b int) int {
	if a <= 0 {
		return 0
	}
	return (a + b - 1) / b
}
