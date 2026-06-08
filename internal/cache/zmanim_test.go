/*
 * SPDX-FileCopyrightText: Copyright 2026 Todd Schulman
 *
 * SPDX-License-Identifier: GPL-3.0-or-later
 */

package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/toobuntu/zman-didan/internal/types"
)

func newTestCache(t *testing.T) *ZmanimCache {
	t.Helper()
	dir := t.TempDir()
	return NewAt(filepath.Join(dir, "zmanim.json"))
}

func makeZmanim(date time.Time) types.ZmanimDay {
	loc, _ := time.LoadLocation("America/New_York")
	return types.ZmanimDay{
		Date:   date,
		Alos:   time.Date(date.Year(), date.Month(), date.Day(), 5, 30, 0, 0, loc),
		Shkiah: time.Date(date.Year(), date.Month(), date.Day(), 19, 17, 0, 0, loc),
		Tzeis:  time.Date(date.Year(), date.Month(), date.Day(), 19, 45, 0, 0, loc),
	}
}

func TestZmanimCache_GetMiss(t *testing.T) {
	c := newTestCache(t)
	loc, _ := time.LoadLocation("America/New_York")
	date := time.Date(2026, 3, 20, 0, 0, 0, 0, loc)
	_, ok := c.Get(date, "17601")
	if ok {
		t.Error("expected miss on empty cache, got hit")
	}
}

func TestZmanimCache_SetAndGet(t *testing.T) {
	c := newTestCache(t)
	loc, _ := time.LoadLocation("America/New_York")
	date := time.Date(2026, 3, 20, 0, 0, 0, 0, loc)
	z := makeZmanim(date)

	if err := c.Set(date, "17601", z); err != nil {
		t.Fatalf("Set error: %v", err)
	}

	got, ok := c.Get(date, "17601")
	if !ok {
		t.Fatal("expected hit after Set, got miss")
	}
	if !got.Shkiah.Equal(z.Shkiah) {
		t.Errorf("Shkiah: got %v, want %v", got.Shkiah, z.Shkiah)
	}
}

func TestZmanimCache_SetDoesNotPrune(t *testing.T) {
	// Set() must not call prune() internally. Calling Set() for dates in the
	// past should not evict those entries on the same call.
	c := newTestCache(t)
	loc, _ := time.LoadLocation("America/New_York")

	// Date is 2 days ago — within the 30-day retention window.
	recent := time.Now().AddDate(0, 0, -2).In(loc)
	recent = time.Date(recent.Year(), recent.Month(), recent.Day(), 0, 0, 0, 0, loc)
	z := makeZmanim(recent)

	if err := c.Set(recent, "17601", z); err != nil {
		t.Fatalf("Set error: %v", err)
	}
	_, ok := c.Get(recent, "17601")
	if !ok {
		t.Error("entry within retention window was evicted by Set(); Set() must not prune")
	}
}

func TestZmanimCache_PruneRetainsRecent(t *testing.T) {
	c := newTestCache(t)
	loc, _ := time.LoadLocation("America/New_York")

	// 3 days ago — within retention window.
	recent := time.Now().AddDate(0, 0, -3).In(loc)
	recent = time.Date(recent.Year(), recent.Month(), recent.Day(), 0, 0, 0, 0, loc)

	if err := c.Set(recent, "17601", makeZmanim(recent)); err != nil {
		t.Fatalf("Set error: %v", err)
	}
	if err := c.Prune(); err != nil {
		t.Fatalf("Prune error: %v", err)
	}
	_, ok := c.Get(recent, "17601")
	if !ok {
		t.Error("recent entry (3 days old) was incorrectly pruned")
	}
}

func TestZmanimCache_PruneEvictsOld(t *testing.T) {
	c := newTestCache(t)
	loc, _ := time.LoadLocation("America/New_York")

	// 35 days ago — beyond the 30-day retention window.
	old := time.Now().AddDate(0, 0, -35).In(loc)
	old = time.Date(old.Year(), old.Month(), old.Day(), 0, 0, 0, 0, loc)

	// Inject an old entry directly, bypassing Set() (which would also be
	// within a valid call, but we want to test prune specifically).
	if err := c.load(); err != nil {
		t.Fatalf("load error: %v", err)
	}
	c.data[cacheKey(old, "17601")] = serialize(makeZmanim(old))

	if err := c.Prune(); err != nil {
		t.Fatalf("Prune error: %v", err)
	}
	_, ok := c.Get(old, "17601")
	if ok {
		t.Error("entry older than 30 days was not pruned")
	}
}

func TestZmanimCache_PruneNoWriteIfUnchanged(t *testing.T) {
	c := newTestCache(t)
	loc, _ := time.LoadLocation("America/New_York")

	// Add one entry within the retention window.
	date := time.Now().AddDate(0, 0, -1).In(loc)
	date = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, loc)
	if err := c.Set(date, "17601", makeZmanim(date)); err != nil {
		t.Fatalf("Set error: %v", err)
	}

	// Record mtime after Set.
	info1, err := os.Stat(c.path)
	if err != nil {
		t.Fatalf("stat error: %v", err)
	}

	// Prune should find nothing to remove and must not rewrite the file.
	// Sleep briefly to ensure mtime would differ if the file were written.
	time.Sleep(10 * time.Millisecond)
	if err := c.Prune(); err != nil {
		t.Fatalf("Prune error: %v", err)
	}

	info2, err := os.Stat(c.path)
	if err != nil {
		t.Fatalf("stat error: %v", err)
	}
	if !info1.ModTime().Equal(info2.ModTime()) {
		t.Error("Prune rewrote the file when nothing changed")
	}
}

func TestZmanimCache_Persistence(t *testing.T) {
	// Verify data survives a new ZmanimCache pointing at the same path.
	dir := t.TempDir()
	path := filepath.Join(dir, "zmanim.json")
	loc, _ := time.LoadLocation("America/New_York")
	date := time.Date(2026, 3, 20, 0, 0, 0, 0, loc)
	z := makeZmanim(date)

	c1 := NewAt(path)
	if err := c1.Set(date, "17601", z); err != nil {
		t.Fatalf("Set error: %v", err)
	}

	c2 := NewAt(path)
	got, ok := c2.Get(date, "17601")
	if !ok {
		t.Fatal("data not persisted across cache instances")
	}
	if !got.Alos.Equal(z.Alos) {
		t.Errorf("Alos: got %v, want %v", got.Alos, z.Alos)
	}
}

func TestZmanimCache_LocationIDIsolation(t *testing.T) {
	// Different locationIDs must not collide.
	c := newTestCache(t)
	loc, _ := time.LoadLocation("America/New_York")
	date := time.Date(2026, 3, 20, 0, 0, 0, 0, loc)

	z1 := makeZmanim(date)
	z1.Shkiah = time.Date(2026, 3, 20, 19, 17, 0, 0, loc)

	z2 := makeZmanim(date)
	z2.Shkiah = time.Date(2026, 3, 20, 18, 45, 0, 0, loc)

	c.Set(date, "17601", z1) //nolint:errcheck
	c.Set(date, "19103", z2) //nolint:errcheck

	got1, _ := c.Get(date, "17601")
	got2, _ := c.Get(date, "19103")

	if got1.Shkiah.Equal(got2.Shkiah) {
		t.Error("different locationIDs returned the same Shkiah time")
	}
}
