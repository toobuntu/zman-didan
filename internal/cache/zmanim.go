// SPDX-FileCopyrightText: 2026 toobuntu
// SPDX-License-Identifier: GPL-3.0-or-later

// Package cache implements persistent caches for didan's external data sources.
//
// ZmanimCache: file-backed JSON store for per-date zmanim, keyed "YYYY-MM-DD|locationID".
// Entries are pruned when their date is more than zmanimRetention days in the past.
//
// Zmanim data is deterministic for a given date and location — it never changes.
// The retention window exists only to prevent unbounded cache growth, not to
// enforce freshness. Pruning on "before today" would re-fetch every past date
// on every run; a rolling 30-day window avoids that while bounding storage.
package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/toobuntu/zman-didan/internal/types"
)

// zmanimRetention is how far into the past we keep cached zmanim entries.
// Dates older than this are pruned to prevent unbounded growth.
const zmanimRetention = 30 * 24 * time.Hour

func defaultZmanimPath() string {
	return filepath.Join(CacheBaseDir(), "zmanim.json")
}

type entry struct {
	Date           string  `json:"date"`
	Alos           string  `json:"alos"`
	Misheyakir     string  `json:"misheyakir"`
	Sunrise        string  `json:"sunrise"`
	LatestShema    string  `json:"latest_shema"`
	Chatzos        string  `json:"chatzos"`
	PlagHamincha   string  `json:"plag_hamincha"`
	Shkiah         string  `json:"shkiah"`
	Tzeis          string  `json:"tzeis"`
	ChatzosHalaila string  `json:"chatzos_halaila"`
	ShaahZmanitMin float64 `json:"shaah_zmanit_min"`
}

// ZmanimCache is a file-backed key/value store for ZmanimDay values.
type ZmanimCache struct {
	path string
	data map[string]entry
}

// New returns a ZmanimCache backed by the default path (~/.cache/didan/zmanim.json).
func New() *ZmanimCache {
	return NewAt(defaultZmanimPath())
}

// NewAt returns a ZmanimCache backed by the given file path.
func NewAt(path string) *ZmanimCache {
	return &ZmanimCache{path: path}
}

// Get returns the cached ZmanimDay for date+locationID, or (zero, false) on miss.
func (c *ZmanimCache) Get(date time.Time, locationID string) (types.ZmanimDay, bool) {
	if err := c.load(); err != nil {
		return types.ZmanimDay{}, false
	}
	e, ok := c.data[cacheKey(date, locationID)]
	if !ok {
		return types.ZmanimDay{}, false
	}
	z, err := deserialize(e)
	if err != nil {
		return types.ZmanimDay{}, false
	}
	return z, true
}

// Set stores a ZmanimDay and persists to disk. Does not prune — pruning is
// done once per run via Prune() at startup to avoid redundant file writes.
func (c *ZmanimCache) Set(date time.Time, locationID string, z types.ZmanimDay) error {
	if err := c.load(); err != nil {
		return err
	}
	c.data[cacheKey(date, locationID)] = serialize(z)
	return c.save()
}

// Prune removes entries older than zmanimRetention and saves only if anything changed.
// Call once per run before the fetch loop.
func (c *ZmanimCache) Prune() error {
	if err := c.load(); err != nil {
		return err
	}
	before := len(c.data)
	c.prune()
	if len(c.data) != before {
		return c.save()
	}
	return nil
}

func (c *ZmanimCache) load() error {
	if c.data != nil {
		return nil
	}
	c.data = make(map[string]entry)
	b, err := os.ReadFile(c.path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading cache %s: %w", c.path, err)
	}
	if err := json.Unmarshal(b, &c.data); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: cache at %s is corrupt; starting fresh\n", c.path)
		c.data = make(map[string]entry)
	}
	return nil
}

func (c *ZmanimCache) save() error {
	if err := os.MkdirAll(filepath.Dir(c.path), 0o755); err != nil {
		return fmt.Errorf("creating cache directory: %w", err)
	}
	b, err := json.MarshalIndent(c.data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling cache: %w", err)
	}
	return os.WriteFile(c.path, b, 0o644)
}

// prune removes entries whose date is older than zmanimRetention.
func (c *ZmanimCache) prune() {
	cutoff := time.Now().Add(-zmanimRetention)
	for k := range c.data {
		datePart := strings.SplitN(k, "|", 2)[0]
		t, err := time.Parse("2006-01-02", datePart)
		if err != nil || t.Before(cutoff) {
			delete(c.data, k)
		}
	}
}

func cacheKey(date time.Time, locationID string) string {
	return fmt.Sprintf("%s|%s", date.Format("2006-01-02"), locationID)
}

func serialize(z types.ZmanimDay) entry {
	f := func(t time.Time) string { return t.Format(time.RFC3339) }
	return entry{
		Date:           z.Date.Format("2006-01-02"),
		Alos:           f(z.Alos),
		Misheyakir:     f(z.Misheyakir),
		Sunrise:        f(z.Sunrise),
		LatestShema:    f(z.LatestShema),
		Chatzos:        f(z.Chatzos),
		PlagHamincha:   f(z.PlagHamincha),
		Shkiah:         f(z.Shkiah),
		Tzeis:          f(z.Tzeis),
		ChatzosHalaila: f(z.ChatzosHalaila),
		ShaahZmanitMin: z.ShaahZmanitMin,
	}
}

func deserialize(e entry) (types.ZmanimDay, error) {
	p := func(s string) (time.Time, error) { return time.Parse(time.RFC3339, s) }
	date, err := time.Parse("2006-01-02", e.Date)
	if err != nil {
		return types.ZmanimDay{}, fmt.Errorf("parsing cache date %q: %w", e.Date, err)
	}
	alos, err := p(e.Alos)
	if err != nil {
		return types.ZmanimDay{}, err
	}
	misheyakir, err := p(e.Misheyakir)
	if err != nil {
		return types.ZmanimDay{}, err
	}
	sunrise, err := p(e.Sunrise)
	if err != nil {
		return types.ZmanimDay{}, err
	}
	latestShema, err := p(e.LatestShema)
	if err != nil {
		return types.ZmanimDay{}, err
	}
	chatzos, err := p(e.Chatzos)
	if err != nil {
		return types.ZmanimDay{}, err
	}
	plag, err := p(e.PlagHamincha)
	if err != nil {
		return types.ZmanimDay{}, err
	}
	shkiah, err := p(e.Shkiah)
	if err != nil {
		return types.ZmanimDay{}, err
	}
	tzeis, err := p(e.Tzeis)
	if err != nil {
		return types.ZmanimDay{}, err
	}
	chatzosHalaila, err := p(e.ChatzosHalaila)
	if err != nil {
		return types.ZmanimDay{}, err
	}
	return types.ZmanimDay{
		Date: date, Alos: alos, Misheyakir: misheyakir, Sunrise: sunrise,
		LatestShema: latestShema, Chatzos: chatzos, PlagHamincha: plag,
		Shkiah: shkiah, Tzeis: tzeis, ChatzosHalaila: chatzosHalaila,
		ShaahZmanitMin: e.ShaahZmanitMin,
	}, nil
}
