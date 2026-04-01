// SPDX-FileCopyrightText: 2026 toobuntu
// SPDX-License-Identifier: GPL-3.0-or-later

// Package cache implements persistent caches for didan's external data sources.
//
// # HTTPCache
//
// File-based cache for raw HTTP response bodies. Used for responses that don't
// require per-date keying: Hebcal calendar JSON, chabad.org candle ICS, and
// the Hebcal special-dates ICS.
//
// Cache directory: ~/.cache/didan/http/
// Key:             hex(SHA-256(url)) — the full URL including all query params
// Eviction:        file mtime older than TTL
//
// Using the full URL as key means that different parameter combinations
// (year, location, candle minutes, language) are cached independently.
// The same parameters always produce the same data, so the TTL only exists
// as a backstop for long gaps between runs; --refresh is the explicit override.
// 7 days is a reasonable TTL for this data.
package cache

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const httpSubdir = "http"

// CacheBaseDir returns ~/.cache/didan. Exported so ZmanimCache can use it
// without redeclaring.
func CacheBaseDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return filepath.Join(home, ".cache", "didan")
}

// HTTPCache caches raw HTTP response bodies on disk, keyed by URL.
type HTTPCache struct {
	dir string
	ttl time.Duration
}

// NewHTTP returns an HTTPCache under ~/.cache/didan/http/ with the given TTL.
func NewHTTP(ttl time.Duration) *HTTPCache {
	return &HTTPCache{
		dir: filepath.Join(CacheBaseDir(), httpSubdir),
		ttl: ttl,
	}
}

// Get returns the cached body for rawURL if the entry exists and is within TTL.
// Returns nil, false on a miss or expired entry.
func (c *HTTPCache) Get(rawURL string) ([]byte, bool) {
	p := c.path(rawURL)
	info, err := os.Stat(p)
	if err != nil || time.Since(info.ModTime()) > c.ttl {
		return nil, false
	}
	body, err := os.ReadFile(p)
	if err != nil {
		return nil, false
	}
	return body, true
}

// Set writes body to the cache for rawURL, creating the directory if needed.
func (c *HTTPCache) Set(rawURL string, body []byte) error {
	if err := os.MkdirAll(c.dir, 0o755); err != nil {
		return fmt.Errorf("creating http cache dir: %w", err)
	}
	return os.WriteFile(c.path(rawURL), body, 0o644)
}

func (c *HTTPCache) path(rawURL string) string {
	sum := sha256.Sum256([]byte(rawURL))
	return filepath.Join(c.dir, fmt.Sprintf("%x", sum))
}
