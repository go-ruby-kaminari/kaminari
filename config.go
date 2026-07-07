// Copyright (c) the go-ruby-kaminari/kaminari authors
//
// SPDX-License-Identifier: BSD-3-Clause

package kaminari

// DefaultPerPage is kaminari's `Kaminari.config.default_per_page`: the number of
// records a page holds when `.per` is never called (or is called with an
// unusable value).
const DefaultPerPage = 25

// Config mirrors the subset of `Kaminari.config` that governs the scope algebra.
// The zero value is not usable directly; construct one with [NewConfig] (or let
// the paginator constructors fill it), so DefaultPerPage is applied.
//
// MaxPerPage and MaxPages are pointers because kaminari treats them as
// tri-state: nil means "unset" (no ceiling), whereas a set value — including a
// deliberately small one — is honoured.
type Config struct {
	// DefaultPerPage is the per-page count applied when `.per` is not used.
	DefaultPerPage int
	// MaxPerPage caps `.per`; nil disables the cap. When default_per_page
	// itself exceeds this cap, `.page` already clamps to it.
	MaxPerPage *int
	// MaxPages caps total_pages; nil disables the cap.
	MaxPages *int
}

// NewConfig returns a Config seeded with kaminari's defaults (default_per_page
// = 25, max_per_page and max_pages unset). The variadic options mutate it.
func NewConfig(opts ...ConfigOption) Config {
	c := Config{DefaultPerPage: DefaultPerPage}
	for _, o := range opts {
		o(&c)
	}
	c.normalize()
	return c
}

// normalize repairs a Config so the scope math never divides by a nonsensical
// default_per_page.
func (c *Config) normalize() {
	if c.DefaultPerPage <= 0 {
		c.DefaultPerPage = DefaultPerPage
	}
}

// ConfigOption customises a [Config].
type ConfigOption func(*Config)

// WithDefaultPerPage sets Kaminari.config.default_per_page.
func WithDefaultPerPage(n int) ConfigOption { return func(c *Config) { c.DefaultPerPage = n } }

// WithMaxPerPage sets Kaminari.config.max_per_page (the `.per` ceiling).
func WithMaxPerPage(n int) ConfigOption { return func(c *Config) { v := n; c.MaxPerPage = &v } }

// WithMaxPages sets Kaminari.config.max_pages (the total_pages ceiling).
func WithMaxPages(n int) ConfigOption { return func(c *Config) { v := n; c.MaxPages = &v } }

// Intp is a convenience for building the *int arguments the pointer-taking
// scope methods use (for example `page.Per(kaminari.Intp(10))`). It mirrors
// Ruby's ability to pass either an integer or nil.
func Intp(n int) *int { return &n }
