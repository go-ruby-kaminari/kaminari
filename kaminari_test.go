// Copyright (c) the go-ruby-kaminari/kaminari authors
//
// SPDX-License-Identifier: BSD-3-Clause

package kaminari

import (
	"reflect"
	"testing"
)

// items builds an []any of the integers [1..n] for slicing assertions.
func items(n int) []any {
	out := make([]any, n)
	for i := range out {
		out[i] = i + 1
	}
	return out
}

// ---- config.go ----

func TestNewConfigDefaults(t *testing.T) {
	c := NewConfig()
	if c.DefaultPerPage != DefaultPerPage {
		t.Fatalf("default per page = %d, want %d", c.DefaultPerPage, DefaultPerPage)
	}
	if c.MaxPerPage != nil || c.MaxPages != nil {
		t.Fatalf("caps should be unset by default")
	}
}

func TestConfigOptions(t *testing.T) {
	c := NewConfig(WithDefaultPerPage(10), WithMaxPerPage(50), WithMaxPages(7))
	if c.DefaultPerPage != 10 {
		t.Fatalf("default per page = %d", c.DefaultPerPage)
	}
	if c.MaxPerPage == nil || *c.MaxPerPage != 50 {
		t.Fatalf("max per page = %v", c.MaxPerPage)
	}
	if c.MaxPages == nil || *c.MaxPages != 7 {
		t.Fatalf("max pages = %v", c.MaxPages)
	}
}

func TestConfigNormalizeFloor(t *testing.T) {
	// A non-positive default_per_page is repaired to the package default.
	c := NewConfig(WithDefaultPerPage(0))
	if c.DefaultPerPage != DefaultPerPage {
		t.Fatalf("normalize failed: %d", c.DefaultPerPage)
	}
}

func TestIntp(t *testing.T) {
	if p := Intp(3); p == nil || *p != 3 {
		t.Fatalf("Intp = %v", p)
	}
}

// ---- array.go: construction & slicing ----

func TestArrayBasicPagination(t *testing.T) {
	a := NewArray(items(100))
	p := a.Page(2).Per(Intp(10))
	if got := p.CurrentPage(); got != 2 {
		t.Fatalf("current page = %d", got)
	}
	if got := p.TotalPages(); got != 10 {
		t.Fatalf("total pages = %d", got)
	}
	if got := p.TotalCount(); got != 100 {
		t.Fatalf("total count = %d", got)
	}
	if got := p.LimitValue(); got != 10 {
		t.Fatalf("limit value = %d", got)
	}
	if got := p.OffsetValue(); got != 10 {
		t.Fatalf("offset value = %d", got)
	}
	want := items(100)[10:20]
	if got := p.Records(); !reflect.DeepEqual(got, want) {
		t.Fatalf("records = %v want %v", got, want)
	}
}

func TestPaginateArrayAliasAndNilItems(t *testing.T) {
	p := PaginateArray(nil)
	if p.TotalCount() != 0 {
		t.Fatalf("nil items total = %d", p.TotalCount())
	}
	if got := p.Page(1).Records(); len(got) != 0 {
		t.Fatalf("empty records = %v", got)
	}
}

func TestArrayOptions(t *testing.T) {
	// total_count override, seed limit/offset/padding, custom config.
	a := NewArray(items(5),
		WithConfig(NewConfig(WithDefaultPerPage(2))),
		WithTotalCount(100),
		WithLimit(3),
		WithOffset(1),
		WithPadding(1),
	)
	if a.TotalCount() != 100 {
		t.Fatalf("total = %d", a.TotalCount())
	}
	if a.LimitValue() != 3 {
		t.Fatalf("limit = %d", a.LimitValue())
	}
	// offset 1 + padding 1 = slice offset 2.
	if a.OffsetValue() != 2 {
		t.Fatalf("offset value = %d", a.OffsetValue())
	}
	if a.CurrentPerPage() != 3 {
		t.Fatalf("per = %d", a.CurrentPerPage())
	}
}

func TestArrayRecordsWindowClipAtEnd(t *testing.T) {
	// Last page slices a partial window.
	a := NewArray(items(7)).Page(3).Per(Intp(3))
	want := []any{7}
	if got := a.Records(); !reflect.DeepEqual(got, want) {
		t.Fatalf("records = %v want %v", got, want)
	}
}

func TestArrayRecordsStartOutOfRange(t *testing.T) {
	// Page far beyond the data => start >= len => empty.
	a := NewArray(items(5)).Per(Intp(2)).Page(9)
	if got := a.Records(); len(got) != 0 {
		t.Fatalf("out-of-range records = %v", got)
	}
}

func TestArrayRecordsNegativeStart(t *testing.T) {
	// A negative seeded offset yields start < 0 => empty (Ruby array[-n,len]).
	a := NewArray(items(5), WithOffset(-5))
	if got := a.Records(); len(got) != 0 {
		t.Fatalf("negative-start records = %v", got)
	}
}

func TestArrayRecordsZeroLimit(t *testing.T) {
	// per(0) => limit 0 => empty window even though start is in range.
	a := NewArray(items(5)).Per(Intp(0))
	if got := a.Records(); len(got) != 0 {
		t.Fatalf("zero-limit records = %v", got)
	}
}

func TestArrayRecordsUnlimited(t *testing.T) {
	a := NewArray(items(5)).Per(nil)
	want := items(5)
	if got := a.Records(); !reflect.DeepEqual(got, want) {
		t.Fatalf("unlimited records = %v want %v", got, want)
	}
	// Unlimited with padding skips the padded rows.
	a2 := NewArray(items(5)).Per(nil).Padding(2)
	want2 := []any{3, 4, 5}
	if got := a2.Records(); !reflect.DeepEqual(got, want2) {
		t.Fatalf("unlimited+padding records = %v want %v", got, want2)
	}
}

func TestArrayRecordsIsCopy(t *testing.T) {
	src := items(3)
	a := NewArray(src).Per(Intp(3))
	got := a.Records()
	got[0] = "mutated"
	if src[0] != 1 {
		t.Fatalf("Records aliased the backing array")
	}
}

// ---- scope.go: per / page / padding semantics ----

func TestPerNilNoLimit(t *testing.T) {
	p := NewArray(items(40)).Page(2).Per(nil)
	if !p.scope.unlimited {
		t.Fatalf("per(nil) should be unlimited")
	}
	if got := p.CurrentPage(); got != 1 {
		t.Fatalf("unlimited current page = %d", got)
	}
	if got := p.TotalPages(); got != 1 {
		t.Fatalf("unlimited total pages = %d", got)
	}
	if got := p.LimitValue(); got != 40 {
		t.Fatalf("unlimited limit value = %d", got)
	}
	// @_per reported as default_per_page.
	if got := p.CurrentPerPage(); got != DefaultPerPage {
		t.Fatalf("unlimited per = %d", got)
	}
}

func TestPerNilEmptyIsOutOfRange(t *testing.T) {
	p := NewArray(nil).Per(nil)
	if got := p.TotalPages(); got != 0 {
		t.Fatalf("empty unlimited total pages = %d", got)
	}
	if !p.OutOfRange() {
		t.Fatalf("empty unlimited should be out of range")
	}
}

func TestPerNegativeIgnored(t *testing.T) {
	// A negative per leaves the limit unchanged (still default), but @_per
	// records the requested value.
	p := NewArray(items(50)).Per(Intp(-3))
	if p.LimitValue() != DefaultPerPage {
		t.Fatalf("negative per changed limit: %d", p.LimitValue())
	}
	if p.CurrentPerPage() != -3 {
		t.Fatalf("current per = %d", p.CurrentPerPage())
	}
}

func TestPerZero(t *testing.T) {
	p := NewArray(items(50)).Per(Intp(0))
	if p.LimitValue() != 0 {
		t.Fatalf("per(0) limit = %d", p.LimitValue())
	}
	if p.CurrentPage() != 0 {
		t.Fatalf("per(0) current page = %d", p.CurrentPage())
	}
	if p.TotalPages() != 0 {
		t.Fatalf("per(0) total pages = %d", p.TotalPages())
	}
}

func TestPerMaxPerPageConfigClamp(t *testing.T) {
	a := NewArray(items(100), WithConfig(NewConfig(WithMaxPerPage(10))))
	p := a.Page(3).Per(Intp(50))
	if p.LimitValue() != 10 {
		t.Fatalf("clamped limit = %d", p.LimitValue())
	}
	// Requested value is still reported by current_per_page.
	if p.CurrentPerPage() != 50 {
		t.Fatalf("current per = %d", p.CurrentPerPage())
	}
	if p.CurrentPage() != 3 {
		t.Fatalf("page preserved = %d", p.CurrentPage())
	}
}

func TestPerExplicitMaxOverridesConfig(t *testing.T) {
	a := NewArray(items(100), WithConfig(NewConfig(WithMaxPerPage(80))))
	// Explicit per-call max_per_page (5) wins over the config cap (80).
	p := a.Per(Intp(50), 5)
	if p.LimitValue() != 5 {
		t.Fatalf("explicit-max clamp = %d", p.LimitValue())
	}
}

func TestPerBelowMaxNotClamped(t *testing.T) {
	a := NewArray(items(100), WithConfig(NewConfig(WithMaxPerPage(80))))
	p := a.Per(Intp(20))
	if p.LimitValue() != 20 {
		t.Fatalf("below-max limit = %d", p.LimitValue())
	}
}

func TestPerRecomputeFromZeroLimit(t *testing.T) {
	// per(0) then per(10): the page-preserving recompute divides by the old
	// limit only when it is positive; from a 0 limit the page is treated as 0.
	p := NewArray(items(100)).Per(Intp(0)).Per(Intp(10))
	if p.OffsetValue() != 0 {
		t.Fatalf("offset after 0->10 = %d", p.OffsetValue())
	}
	if p.CurrentPage() != 1 {
		t.Fatalf("current page = %d", p.CurrentPage())
	}
}

func TestPageClampAndDefaultPer(t *testing.T) {
	// page(0) clamps up to 1; a bare page() uses default_per_page.
	p := NewArray(items(100)).Page(0)
	if p.CurrentPage() != 1 {
		t.Fatalf("page(0) => %d", p.CurrentPage())
	}
	if p.LimitValue() != DefaultPerPage {
		t.Fatalf("default per = %d", p.LimitValue())
	}
	if p.CurrentPerPage() != DefaultPerPage {
		t.Fatalf("current per = %d", p.CurrentPerPage())
	}
}

func TestPageClampsPerToMaxPerPage(t *testing.T) {
	// When default_per_page exceeds max_per_page, page() already clamps.
	a := NewArray(items(100), WithConfig(NewConfig(WithDefaultPerPage(30), WithMaxPerPage(10))))
	p := a.Page(2)
	if p.LimitValue() != 10 {
		t.Fatalf("page clamp = %d", p.LimitValue())
	}
	if p.OffsetValue() != 10 {
		t.Fatalf("offset = %d", p.OffsetValue())
	}
}

func TestPadding(t *testing.T) {
	// padding shifts the data window but keeps the page number and trims
	// total_pages.
	p := NewArray(items(100)).Page(3).Per(Intp(10)).Padding(5)
	if p.OffsetValue() != 25 {
		t.Fatalf("offset value = %d", p.OffsetValue())
	}
	if p.CurrentPage() != 3 {
		t.Fatalf("current page = %d", p.CurrentPage())
	}
	// (100-5)/10 = 9.5 -> 10 pages.
	if p.TotalPages() != 10 {
		t.Fatalf("total pages = %d", p.TotalPages())
	}
	// adjustedTotal never goes negative.
	if got := NewArray(items(3)).Per(Intp(10)).Padding(100).TotalPages(); got != 0 {
		t.Fatalf("over-padded total pages = %d", got)
	}
}

// ---- scope.go: metadata edges ----

func TestFirstLastOutOfRange(t *testing.T) {
	// page(n).per(10) is the faithful order — page() resets per, per() refines it.
	page := func(n int) *Array { return NewArray(items(30)).Page(n).Per(Intp(10)) }
	if !page(1).FirstPage() || page(1).LastPage() {
		t.Fatalf("page 1 flags wrong")
	}
	if !page(3).LastPage() {
		t.Fatalf("page 3 should be last")
	}
	if !page(9).OutOfRange() {
		t.Fatalf("page 9 should be out of range")
	}
	// A middle page is neither first nor last.
	if page(2).FirstPage() || page(2).LastPage() || page(2).OutOfRange() {
		t.Fatalf("page 2 flags wrong")
	}
}

func TestPrevNextPage(t *testing.T) {
	page := func(n int) *Array { return NewArray(items(30)).Page(n).Per(Intp(10)) }

	if page(1).PrevPage() != nil {
		t.Fatalf("first page has no prev")
	}
	if got := page(2).PrevPage(); got == nil || *got != 1 {
		t.Fatalf("page 2 prev = %v", got)
	}
	if got := page(2).NextPage(); got == nil || *got != 3 {
		t.Fatalf("page 2 next = %v", got)
	}
	if page(3).NextPage() != nil {
		t.Fatalf("last page has no next")
	}
	// Out of range: neither prev nor next.
	oor := page(9)
	if oor.PrevPage() != nil || oor.NextPage() != nil {
		t.Fatalf("out-of-range should have no prev/next")
	}
}

func TestTotalPagesMaxPagesClamp(t *testing.T) {
	// 100/10 = 10 pages, clamped down to max_pages = 3.
	a := NewArray(items(100), WithConfig(NewConfig(WithMaxPages(3)))).Per(Intp(10))
	if got := a.TotalPages(); got != 3 {
		t.Fatalf("clamped total pages = %d", got)
	}
	// max_pages above the real count leaves it unchanged.
	b := NewArray(items(100), WithConfig(NewConfig(WithMaxPages(50)))).Per(Intp(10))
	if got := b.TotalPages(); got != 10 {
		t.Fatalf("unclamped total pages = %d", got)
	}
}

func TestZeroCount(t *testing.T) {
	a := NewArray(nil).Per(Intp(10))
	if a.TotalCount() != 0 {
		t.Fatalf("count = %d", a.TotalCount())
	}
	if a.TotalPages() != 0 {
		t.Fatalf("total pages = %d", a.TotalPages())
	}
	if a.CurrentPage() != 1 {
		t.Fatalf("current page = %d", a.CurrentPage())
	}
	if !a.OutOfRange() {
		t.Fatalf("page 1 of 0 pages is out of range")
	}
}

func TestCeilDiv(t *testing.T) {
	cases := map[[2]int]int{{0, 5}: 0, {1, 5}: 1, {5, 5}: 1, {6, 5}: 2, {-3, 5}: 0}
	for in, want := range cases {
		if got := ceilDiv(in[0], in[1]); got != want {
			t.Fatalf("ceilDiv%v = %d want %d", in, got, want)
		}
	}
}

// ---- EntriesInfo ----

func TestEntriesInfoEmpty(t *testing.T) {
	got := NewArray(nil).Per(Intp(10)).EntriesInfo()
	if !got.Empty || got.TotalCount != 0 {
		t.Fatalf("empty entries info = %+v", got)
	}
}

func TestEntriesInfoOnePage(t *testing.T) {
	got := NewArray(items(8)).Per(Intp(25)).Page(1).EntriesInfo()
	if !got.OnePage || got.First != 1 || got.Last != 8 || got.TotalCount != 8 {
		t.Fatalf("one-page entries info = %+v", got)
	}
}

func TestEntriesInfoMiddleAndLast(t *testing.T) {
	mid := NewArray(items(100)).Per(Intp(25)).Page(2).EntriesInfo()
	if mid.First != 26 || mid.Last != 50 || mid.TotalCount != 100 {
		t.Fatalf("middle entries info = %+v", mid)
	}
	last := NewArray(items(90)).Per(Intp(25)).Page(4).EntriesInfo()
	if last.First != 76 || last.Last != 90 {
		t.Fatalf("last entries info = %+v", last)
	}
}

// ---- relation.go ----

// fakeRelation records the last Slice call and returns the window as []int.
type fakeRelation struct {
	data        []int
	lastOffset  int
	lastLimit   int
	countCalled int
}

func (f *fakeRelation) Count() int {
	f.countCalled++
	return len(f.data)
}

func (f *fakeRelation) Slice(offset, limit int) any {
	f.lastOffset, f.lastLimit = offset, limit
	if limit < 0 {
		if offset >= len(f.data) {
			return []int{}
		}
		return f.data[offset:]
	}
	if offset >= len(f.data) {
		return []int{}
	}
	end := offset + limit
	if end > len(f.data) {
		end = len(f.data)
	}
	return f.data[offset:end]
}

func seq(n int) []int {
	out := make([]int, n)
	for i := range out {
		out[i] = i + 1
	}
	return out
}

func TestRelationPagination(t *testing.T) {
	rel := &fakeRelation{data: seq(100)}
	p := Paginate(rel).Page(3).Per(Intp(10)).Padding(2)
	if rel.countCalled != 1 {
		t.Fatalf("Count called %d times", rel.countCalled)
	}
	if p.CurrentPage() != 3 {
		t.Fatalf("current page = %d", p.CurrentPage())
	}
	if p.TotalPages() != 10 {
		t.Fatalf("total pages = %d", p.TotalPages())
	}
	got := p.Records().([]int)
	// offset = base 20 + padding 2 = 22, limit 10.
	if rel.lastOffset != 22 || rel.lastLimit != 10 {
		t.Fatalf("slice(%d,%d)", rel.lastOffset, rel.lastLimit)
	}
	want := seq(100)[22:32]
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("records = %v want %v", got, want)
	}
}

func TestRelationUnlimitedSlice(t *testing.T) {
	rel := &fakeRelation{data: seq(5)}
	p := Paginate(rel).Per(nil)
	got := p.Records().([]int)
	if rel.lastLimit != -1 {
		t.Fatalf("unlimited limit passed = %d", rel.lastLimit)
	}
	if !reflect.DeepEqual(got, seq(5)) {
		t.Fatalf("unlimited records = %v", got)
	}
}

func TestRelationOptions(t *testing.T) {
	rel := &fakeRelation{data: seq(3)}
	p := Paginate(rel, WithTotalCount(500), WithLimit(20), WithOffset(40), WithPadding(0))
	if p.TotalCount() != 500 {
		t.Fatalf("total = %d", p.TotalCount())
	}
	if p.LimitValue() != 20 || p.CurrentPerPage() != 20 {
		t.Fatalf("limit = %d per = %d", p.LimitValue(), p.CurrentPerPage())
	}
	if p.OffsetValue() != 40 {
		t.Fatalf("offset = %d", p.OffsetValue())
	}
}

// ---- helpers.go: navigation ----

func TestRelevantPagesEmpty(t *testing.T) {
	if RelevantPages(1, 0, 4, 0, 0) != nil {
		t.Fatalf("no pages should be nil")
	}
}

func TestRelevantPagesWindows(t *testing.T) {
	// current 10 of 20, window 2, left 1, right 1:
	// left {1,2}, inside {8..12}, right {19,20}.
	got := RelevantPages(10, 20, 2, 1, 1)
	want := []int{1, 2, 8, 9, 10, 11, 12, 19, 20}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("relevant pages = %v want %v", got, want)
	}
}

func TestPageItemsGaps(t *testing.T) {
	got := PageItems(10, 20, 2, 1, 1)
	// Expect gaps between 2 and 8, and between 12 and 19.
	var s []string
	for _, it := range got {
		if it.IsGap {
			s = append(s, "…")
		} else {
			s = append(s, itoa(it.Number))
		}
	}
	want := []string{"1", "2", "…", "8", "9", "10", "11", "12", "…", "19", "20"}
	if !reflect.DeepEqual(s, want) {
		t.Fatalf("page items = %v want %v", s, want)
	}
}

func TestPageItemsEmpty(t *testing.T) {
	if PageItems(1, 0, 4, 0, 0) != nil {
		t.Fatalf("no page items should be nil")
	}
}

func TestPageItemsNoGaps(t *testing.T) {
	// A small collection: every page shown, no truncation.
	got := PageItems(2, 4, 4, 0, 0)
	if len(got) != 4 {
		t.Fatalf("items = %d", len(got))
	}
	for i, it := range got {
		if it.IsGap || it.Number != i+1 {
			t.Fatalf("item %d = %+v", i, it)
		}
	}
}

func TestNavConfigPages(t *testing.T) {
	// Left/Right nil fall back to OuterWindow.
	cfg := NavConfig{Window: 1, OuterWindow: 1}
	got := cfg.Pages(10, 20)
	var nums []int
	for _, it := range got {
		if !it.IsGap {
			nums = append(nums, it.Number)
		}
	}
	want := []int{1, 2, 9, 10, 11, 19, 20}
	if !reflect.DeepEqual(nums, want) {
		t.Fatalf("nav pages = %v want %v", nums, want)
	}

	// Explicit Left/Right (0) override OuterWindow (5): left {1}, inside {10},
	// right {20} => 1 … 10 … 20.
	cfg2 := NavConfig{Window: 0, OuterWindow: 5, Left: Intp(0), Right: Intp(0)}
	var nums2 []int
	for _, it := range cfg2.Pages(10, 20) {
		if !it.IsGap {
			nums2 = append(nums2, it.Number)
		}
	}
	if !reflect.DeepEqual(nums2, []int{1, 10, 20}) {
		t.Fatalf("overridden nav = %v", nums2)
	}
}

// itoa is a tiny int->string for the gap-render assertion (avoids strconv noise
// in a table of expectations).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}
