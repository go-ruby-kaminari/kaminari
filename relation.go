// Copyright (c) the go-ruby-kaminari/kaminari authors
//
// SPDX-License-Identifier: BSD-3-Clause

package kaminari

// Relation is the host seam for paginating a lazily-evaluated collection such as
// an ActiveRecord relation. It is the whole surface kaminari needs from the
// backend: a total count and the ability to materialise a window.
//
// The rbgo binding implements Relation against an ActiveRecord relation, mapping
// Count to `count` and Slice(offset, limit) to `offset(offset).limit(limit)` (a
// negative limit meaning "no LIMIT", i.e. `.per(nil)`). Because Slice returns
// `any`, the binding can hand back the concrete relation/array unchanged.
type Relation interface {
	// Count returns total_count of the underlying collection (`SELECT COUNT`).
	Count() int
	// Slice returns the records in the window [offset, offset+limit). A limit
	// of -1 means "no limit": every record from offset onward.
	Slice(offset, limit int) any
}

// RelationPaginator applies kaminari's scope algebra to a [Relation]. Like
// [Array] it is an immutable snapshot: Page, Per, and Padding return a new
// paginator over the same Relation, and the promoted scope methods report the
// page metadata. Records calls through the seam to materialise the window.
//
// The total_count is read from the Relation once, when [Paginate] is called, and
// carried through subsequent page/per/padding steps — kaminari likewise counts
// once per paginated relation.
type RelationPaginator struct {
	scope
	rel Relation
}

// Paginate wraps a [Relation] in a paginator, reading its Count up front. Without
// options it behaves like a bare relation: per = default_per_page, offset 0.
func Paginate(rel Relation, opts ...ArrayOption) *RelationPaginator {
	o := arrayOpts{cfg: NewConfig()}
	for _, opt := range opts {
		opt(&o)
	}
	o.cfg.normalize()
	total := rel.Count()
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
	return &RelationPaginator{scope: s, rel: rel}
}

// Page returns the paginator positioned on page num (`.page(num)`).
func (p *RelationPaginator) Page(num int) *RelationPaginator {
	return &RelationPaginator{scope: p.scope.page(num), rel: p.rel}
}

// Per returns the paginator with per-page set to num (`.per(num)`); a nil num
// means "no limit". An optional maxPerPage overrides the configured ceiling.
func (p *RelationPaginator) Per(num *int, maxPerPage ...int) *RelationPaginator {
	return &RelationPaginator{scope: p.scope.per(num, maxPerPage...), rel: p.rel}
}

// Padding returns the paginator with padding set to num (`.padding(num)`).
func (p *RelationPaginator) Padding(num int) *RelationPaginator {
	return &RelationPaginator{scope: p.scope.padding(num), rel: p.rel}
}

// Records materialises the current page by calling [Relation.Slice] with the
// computed offset (base+padding) and limit (-1 when unlimited). The result is
// whatever the seam returns — the rbgo binding hands back the ActiveRecord
// relation for the page.
func (p *RelationPaginator) Records() any {
	return p.rel.Slice(p.scope.sliceOffset(), p.scope.sliceLimit())
}
