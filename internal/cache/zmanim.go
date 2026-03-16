// Package cache implements a persistent JSON cache for per-date zmanim.
//
// Cache file: ~/.cache/didan/zmanim.json
// Keys:       "YYYY-MM-DD|ZIP"
// Eviction:   entries whose date is strictly before today are pruned on every
//             write, and on an explicit Prune call.
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

func defaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "zmanim.json"
	}
	return filepath.Join(home, ".cache", "didan", "zmanim.json")
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
// Construct with New or NewAt.
type ZmanimCache struct {
	path string
	data map[string]entry
}

// New returns a ZmanimCache backed by the default path (~/.cache/didan/zmanim.json).
func New() *ZmanimCache {
	return NewAt(defaultPath())
}

// NewAt returns a ZmanimCache backed by the given file path.
func NewAt(path string) *ZmanimCache {
	return &ZmanimCache{path: path}
}

// Get returns the cached ZmanimDay for date+zip, or (zero, false) on a miss.
func (c *ZmanimCache) Get(date time.Time, zip string) (types.ZmanimDay, bool) {
	if err := c.load(); err != nil {
		return types.ZmanimDay{}, false
	}
	e, ok := c.data[cacheKey(date, zip)]
	if !ok {
		return types.ZmanimDay{}, false
	}
	z, err := deserialize(e)
	if err != nil {
		return types.ZmanimDay{}, false
	}
	return z, true
}

// Set stores a ZmanimDay, prunes stale entries, and persists to disk.
func (c *ZmanimCache) Set(date time.Time, zip string, z types.ZmanimDay) error {
	if err := c.load(); err != nil {
		return err
	}
	c.data[cacheKey(date, zip)] = serialize(z)
	c.prune()
	return c.save()
}

// Prune removes entries whose date is before today and saves if anything changed.
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

func (c *ZmanimCache) prune() {
	today := time.Now().Truncate(24 * time.Hour)
	for k := range c.data {
		datePart := strings.SplitN(k, "|", 2)[0]
		t, err := time.Parse("2006-01-02", datePart)
		if err != nil || t.Before(today) {
			delete(c.data, k)
		}
	}
}

func cacheKey(date time.Time, zip string) string {
	return fmt.Sprintf("%s|%s", date.Format("2006-01-02"), zip)
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
