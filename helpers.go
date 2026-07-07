// Copyright (c) the go-ruby-kaminari/kaminari authors
//
// SPDX-License-Identifier: BSD-3-Clause

package kaminari

import "sort"

// EntriesInfo is the pure data kaminari's `page_entries_info` view helper renders
// into a sentence like "Displaying entries 1 - 25 of 100 in total". A formatter
// (the view layer, out of scope here) turns it into markup; this package only
// computes the numbers.
//
// The three shapes map to the helper's three branches:
//   - Empty:   total_count is 0                → "No entries found".
//   - OnePage: total_pages < 2 (fits one page) → "Displaying all N entries";
//     First is 1 and Last is total_count.
//   - neither: a middle/edge page of many      → "Displaying entries First - Last
//     of TotalCount"; on the last page Last is clamped to total_count.
type EntriesInfo struct {
	TotalCount int
	First      int
	Last       int
	OnePage    bool
	Empty      bool
}

// PageItem is one entry in a rendered page navigation: either a page link
// (IsGap false, Number set) or a truncation gap ("…", IsGap true) standing in
// for a run of hidden pages.
type PageItem struct {
	Number int
	IsGap  bool
}

// RelevantPages reproduces kaminari's `Kaminari::Helpers::Paginator#relevant_pages`:
// the sorted, de-duplicated set of page numbers worth showing, formed from the
// union of three ranges and clipped to [1, totalPages]:
//
//   - the left window:    pages 1 .. left+1
//   - the inside window:  pages currentPage-window .. currentPage+window
//   - the right window:   pages totalPages-right .. totalPages
//
// It returns nil when totalPages < 1.
func RelevantPages(currentPage, totalPages, window, left, right int) []int {
	if totalPages < 1 {
		return nil
	}
	seen := make(map[int]bool)
	add := func(lo, hi int) {
		for i := lo; i <= hi; i++ {
			if i >= 1 && i <= totalPages {
				seen[i] = true
			}
		}
	}
	add(1, left+1)
	add(currentPage-window, currentPage+window)
	add(totalPages-right, totalPages)

	out := make([]int, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	sort.Ints(out)
	return out
}

// PageItems reproduces the full navigation kaminari's `paginate` helper walks:
// the relevant pages in order, with a single gap marker inserted wherever the
// sequence skips one or more page numbers. It returns nil when totalPages < 1.
func PageItems(currentPage, totalPages, window, left, right int) []PageItem {
	pages := RelevantPages(currentPage, totalPages, window, left, right)
	if pages == nil {
		return nil
	}
	items := make([]PageItem, 0, len(pages))
	prev := 0
	for _, p := range pages {
		if prev != 0 && p > prev+1 {
			items = append(items, PageItem{IsGap: true})
		}
		items = append(items, PageItem{Number: p})
		prev = p
	}
	return items
}

// NavConfig mirrors the window knobs of `Kaminari.config` that shape the page
// navigation: Window (default 4) is the spread either side of the current page,
// OuterWindow is the default for both edges, and Left/Right override the edges
// individually (nil falls back to OuterWindow, matching kaminari's
// `left ||= outer_window`).
type NavConfig struct {
	Window      int
	OuterWindow int
	Left        *int
	Right       *int
}

// leftRight resolves the edge windows, applying the OuterWindow fallback.
func (c NavConfig) leftRight() (int, int) {
	l, r := c.OuterWindow, c.OuterWindow
	if c.Left != nil {
		l = *c.Left
	}
	if c.Right != nil {
		r = *c.Right
	}
	return l, r
}

// Pages returns the annotated navigation for the given current/total pages under
// this NavConfig — the convenient wrapper over [PageItems].
func (c NavConfig) Pages(currentPage, totalPages int) []PageItem {
	l, r := c.leftRight()
	return PageItems(currentPage, totalPages, c.Window, l, r)
}
