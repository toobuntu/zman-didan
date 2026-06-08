//go:build integration

// Package integration contains tests that make real network requests.
// They are excluded from normal `go test ./...` runs by the `integration`
// build tag. Run with:
//
//	go test -count=1 -tags integration -timeout 60s ./internal/integration/
//
// Or via the Makefile target:
//
//	make integration
//
// These tests validate that external services continue to return data in the
// formats we parse. They complement unit tests by catching API changes that
// fixed-fixture unit tests cannot detect.
package integration

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/toobuntu/zman-didan/internal/cache"
	"github.com/toobuntu/zman-didan/internal/chabad"
	"github.com/toobuntu/zman-didan/internal/hebcal"
	"github.com/toobuntu/zman-didan/internal/types"
)

// testLoc is Lancaster, PA — used for all integration tests requiring a location.
var testLoc = types.Location{
	ZIP:  "17601",
	TZID: "America/New_York",
	City: "Lancaster, PA 17601",
}

func testCfg() types.Config {
	return types.Config{
		LocationMode: types.LocationZIP,
		ZIP:          "17601",
		Lang:         "h",
		Candles:      25,
		Tosfos:       4,
		Refresh:      true, // always bypass cache in integration tests
	}
}

// ---- Hebcal ----

func TestHebcalFetchYear_ReturnsEvents(t *testing.T) {
	cfg := testCfg()
	cfg.StartDate = time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)
	cfg.EndDate = time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC)

	c := hebcal.NewClient()
	events, loc, _, err := c.FetchYear(cfg)
	if err != nil {
		t.Fatalf("FetchYear: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("expected events, got none")
	}
	if loc.TZID == "" {
		t.Error("expected TZID in location")
	}
	t.Logf("%d events — %s (%s)", len(events), loc.City, loc.TZID)

	var hasCandle, hasHavdala, hasParsha bool
	for _, ev := range events {
		switch ev.Category {
		case "candles":
			hasCandle = true
		case "havdalah":
			hasHavdala = true
		case "parashat":
			hasParsha = true
		}
	}
	if !hasCandle {
		t.Error("no candle lighting event")
	}
	if !hasHavdala {
		t.Error("no havdalah event")
	}
	if !hasParsha {
		t.Error("no parashat event")
	}
}

func TestHebcalFetchYear_ParashaHasSlug(t *testing.T) {
	// Parsha events must have a non-empty Slug — used as the haftorah table key.
	cfg := testCfg()
	cfg.StartDate = time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)
	cfg.EndDate = time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC)

	c := hebcal.NewClient()
	events, _, _, err := c.FetchYear(cfg)
	if err != nil {
		t.Fatalf("FetchYear: %v", err)
	}
	for _, ev := range events {
		if ev.Category == "parashat" && ev.Slug == "" {
			t.Errorf("parashat event %q has empty Slug — haftorah lookup will fail", ev.Title)
		}
	}
}

func TestHebcalFetchYear_AshkenaziNikud(t *testing.T) {
	// For --lang ah, the nikud enrichment fetch (lg=he) should supply nikud
	// codepoints in ev.Hebrew. If the second fetch fails or the field is wrong,
	// this fails visibly rather than silently serving un-nikud'ed Hebrew.
	cfg := testCfg()
	cfg.Lang = "ah"
	cfg.StartDate = time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	cfg.EndDate = time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)

	c := hebcal.NewClient()
	events, _, _, err := c.FetchYear(cfg)
	if err != nil {
		t.Fatalf("FetchYear: %v", err)
	}
	var nikudFound bool
	for _, ev := range events {
		for _, r := range ev.Hebrew {
			if r >= 0x05B0 && r <= 0x05C7 {
				nikudFound = true
				break
			}
		}
		if nikudFound {
			break
		}
	}
	if !nikudFound {
		t.Error("expected nikud codepoints in ev.Hebrew for lang=ah; nikud enrichment may have failed")
	}
}

// ---- Chabad zmanim ----

func TestChabadZmanim_CanonicalFields(t *testing.T) {
	// Verify the RSS parser populates the canonical ZmanimDay fields we rely on.
	c := chabad.NewClient(cache.New())
	date := time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)

	z, _, err := c.FetchZmanimInZone(date, testLoc, testCfg())
	if err != nil {
		t.Fatalf("FetchZmanimInZone: %v", err)
	}

	fields := []struct {
		name string
		t    time.Time
	}{
		{"Alos", z.Alos},
		{"Misheyakir", z.Misheyakir},
		{"Sunrise", z.Sunrise},
		{"LatestShema", z.LatestShema},
		{"Chatzos", z.Chatzos},
		{"Shkiah", z.Shkiah},
		{"Tzeis", z.Tzeis},
	}
	for _, f := range fields {
		if f.t.IsZero() {
			t.Errorf("ZmanimDay.%s is zero — chabad.org RSS label format may have changed", f.name)
		}
	}

	if !z.Alos.Before(z.Sunrise) {
		t.Errorf("Alos (%s) is not before Sunrise (%s)", z.Alos.Format("3:04 PM"), z.Sunrise.Format("3:04 PM"))
	}
	if !z.Sunrise.Before(z.Shkiah) {
		t.Errorf("Sunrise (%s) is not before Shkiah (%s)", z.Sunrise.Format("3:04 PM"), z.Shkiah.Format("3:04 PM"))
	}
	if !z.Shkiah.Before(z.Tzeis) {
		t.Errorf("Shkiah (%s) is not before Tzeis (%s)", z.Shkiah.Format("3:04 PM"), z.Tzeis.Format("3:04 PM"))
	}
	t.Logf("Alos=%s Shkiah=%s Tzeis=%s ShaahZmanit=%.1fmin",
		z.Alos.Format("3:04 PM"), z.Shkiah.Format("3:04 PM"), z.Tzeis.Format("3:04 PM"), z.ShaahZmanitMin)
}

func TestChabadZmanim_ShaahZmanitNonZero(t *testing.T) {
	// ShaahZmanitMin being zero is the canary for silent RSS format breakage —
	// if parsing fails completely all time fields are zero and ShaahZmanitMin is 0.
	c := chabad.NewClient(cache.New())
	date := time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)

	z, _, err := c.FetchZmanimInZone(date, testLoc, testCfg())
	if err != nil {
		t.Fatalf("FetchZmanimInZone: %v", err)
	}
	if z.ShaahZmanitMin == 0 {
		t.Error("ShaahZmanitMin is 0; the chabad.org RSS label format may have changed")
	}
}

func TestChabadZmanim_ShabbosAndYomTovVariants(t *testing.T) {
	// Regression guard for the Shabbos/Yom-Tov Misheyakir label variant. On
	// these days the RSS uses "Earliest Tallit (Misheyakir)" (no "and Tefillin")
	// and "Shabbat Ends"/"Holiday Ends" rather than plain Nightfall. An exact
	// label map dropped both, zeroing Misheyakir and Tzeis on every Shabbos and
	// Yom Tov (collapsing the Havdala "Shma" range to its end time); substring
	// classification must populate them.
	c := chabad.NewClient(cache.New())
	tests := []struct {
		name       string
		date       time.Time
		checkTzeis bool
	}{
		{"Shabbos", time.Date(2026, 3, 21, 0, 0, 0, 0, time.UTC), true}, // Saturday
		{"YomTov", time.Date(2026, 5, 22, 0, 0, 0, 0, time.UTC), false}, // Shavuos I
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			z, _, err := c.FetchZmanimInZone(tt.date, testLoc, testCfg())
			if err != nil {
				t.Fatalf("FetchZmanimInZone: %v", err)
			}
			if z.Misheyakir.IsZero() {
				t.Errorf("%s %s: Misheyakir is zero — the Shabbos/YT label variant was not classified",
					tt.name, tt.date.Format("2006-01-02"))
			}
			if tt.checkTzeis && z.Tzeis.IsZero() {
				t.Errorf("%s %s: Tzeis is zero — \"Shabbat Ends\" was not classified",
					tt.name, tt.date.Format("2006-01-02"))
			}
		})
	}
}

// ---- Chabad candle ICS ----

func TestChabadCandleICS_Types(t *testing.T) {
	// A range covering Pesach must yield Shabbos candles (IsYomTov=false),
	// YT candles (IsYomTov=true), and at least one AfterHavdala entry.
	c := chabad.NewClient(cache.New())

	days, _, err := c.FetchCandlesYear("4/1/2026", testLoc, testCfg())
	if err != nil {
		t.Fatalf("FetchCandlesYear: %v", err)
	}
	if len(days) == 0 {
		t.Fatal("expected candle entries, got none")
	}

	var hasShabbos, hasYT, hasAfter bool
	for _, d := range days {
		if d.Candles.IsZero() {
			continue
		}
		if d.IsYomTov {
			hasYT = true
		} else {
			hasShabbos = true
		}
		if d.AfterHavdala {
			hasAfter = true
		}
	}
	if !hasShabbos {
		t.Error("expected IsYomTov=false (Shabbos) entries")
	}
	if !hasYT {
		t.Error("expected IsYomTov=true (YT) entries")
	}
	if !hasAfter {
		t.Error("expected AfterHavdala=true entries ('Light Holiday Candles after')")
	}
	t.Logf("%d total entries; shabbos=%v yomtov=%v after=%v", len(days), hasShabbos, hasYT, hasAfter)
}

// ---- Special dates ICS ----

func TestSpecialDatesICS_VerboseNames(t *testing.T) {
	// The chabad-special-dates.ics must contain the exact verbose_names our
	// rebbes.json declares. If Hebcal changes these strings, normalizeNames
	// silently stops working.
	const url = "https://download.hebcal.com/ical/chabad-special-dates.ics"

	body, err := simpleGet(url)
	if err != nil {
		t.Fatalf("downloading special dates ICS: %v", err)
	}
	content := string(body)

	// These are the verbose_names from rebbes.json that appear in the feed.
	expected := []string{
		"R. Sholom DovBer of Lubavitch",  // Rebbe Rashab
		"R. Menachem Mendel Schneerson",  // the Rebbe
		"Tzemach Tzeddek",                // the Tzemach Tzedek
		"R. Yosef Yitzchak of Lubavitch", // Frierdiker Rebbe
		"R. Schneur Zalman of Liadi",     // Alter Rebbe
	}
	for _, s := range expected {
		if !strings.Contains(content, s) {
			t.Errorf("special dates ICS missing %q — rebbes.json verbose_names may need updating", s)
		}
	}
	t.Logf("ICS: %d bytes", len(body))
}

// simpleGet is a plain HTTP GET without the chabad.org User-Agent requirement.
// Only used for hebcal.com endpoints (download.hebcal.com does not 403 on Go UA).
func simpleGet(rawURL string) ([]byte, error) {
	resp, err := (&http.Client{Timeout: 30 * time.Second}).Get(rawURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, rawURL)
	}
	return io.ReadAll(resp.Body)
}
