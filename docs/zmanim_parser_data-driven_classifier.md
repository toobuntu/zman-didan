<!--
SPDX-FileCopyrightText: Copyright 2026 Todd Schulman

SPDX-License-Identifier: GPL-3.0-or-later
-->

This is exactly the kind of messy-but-structured data where a small, data-driven classifier pays off. Not for abstraction’s sake—but because you’ve now seen enough variants that hard-coding logic will keep growing in awkward ways.

Let me propose something that stays simple but handles everything you’ve shown, including:
	•	concatenated labels (Shkiah)Fast Begins)
	•	split labels (|)
	•	overlapping semantics (e.g. Candle Lighting == tzeis in some contexts)
	•	dynamic events (Chametz, Fast, etc.)

⸻

🧠 Core idea (refined)

Each label can map to multiple outputs:

(label) → {
  canonical fields (0..n)
  events (0..n)
}

So instead of:

string → single field

you move to:

string → []effects


⸻

🛠️ 1. Fix the parsing first (important!)

You now have this nasty case:

Sunset (Shkiah)Fast Begins

👉 There is no delimiter between parts.

Normalize it:

func normalizeLabel(s string) string {
	// Fix missing delimiter before "Fast"
	s = strings.ReplaceAll(s, ")Fast", ") | Fast")

	return s
}

Then reuse your existing splitter:

func splitLabel(s string) []string {
	s = normalizeLabel(s)

	parts := strings.Split(s, "|")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}


⸻

📊 2. Data-driven classifier (minimal, not over-engineered)

Define rules as data:

type rule struct {
	match   func(string) bool
	fields  []string
	event   bool
}


⸻

🔹 Rules

var rules = []rule{

	// --- Core zmanim ---
	{match: contains("Alot Hashachar"), fields: []string{"alos"}},
	{match: contains("Misheyakir"), fields: []string{"misheyakir"}},
	{match: contains("Sunrise"), fields: []string{"sunrise"}},
	{match: contains("Sunset"), fields: []string{"shkiah"}},
	{match: contains("Latest Shema"), fields: []string{"latest_shema"}},
	{match: contains("Midday"), fields: []string{"chatzos"}},
	{match: contains("Plag"), fields: []string{"plag_hamincha"}},
	{match: contains("Midnight"), fields: []string{"chatzos_halaila"}},

	// --- Tzeis-like (many variants) ---
	{
		match: anyContains(
			"Tzeit Hakochavim",
			"Shabbat Ends",
			"Holiday Ends",
			"Shabbat/Holiday Ends",
			"Holiday/Fast Ends",
			"Fast Ends",
		),
		fields: []string{"tzeis"},
	},

	// --- Candle lighting (dual meaning) ---
	{
		match: contains("Candle Lighting"),
		fields: []string{"tzeis"}, // 👈 your design decision
		event:  true,              // also keep as event
	},

	// --- Fast begins ---
	{
		match: contains("Fast Begins"),
		event: true,
	},

	// --- Chametz ---
	{
		match: contains("Chametz"),
		event: true,
	},
}

Helper functions:

func contains(substr string) func(string) bool {
	return func(s string) bool {
		return strings.Contains(s, substr)
	}
}

func anyContains(subs ...string) func(string) bool {
	return func(s string) bool {
		for _, sub := range subs {
			if strings.Contains(s, sub) {
				return true
			}
		}
		return false
	}
}


⸻

🔁 3. Apply rules

func classify(lbl string) (fields []string, isEvent bool) {
	for _, r := range rules {
		if r.match(lbl) {
			return r.fields, r.event
		}
	}
	return nil, true // default: event
}


⸻

🔧 4. Integration

Replace your matching block with:

labels := splitLabel(prefix)

for _, lbl := range labels {
	fields, isEvent := classify(lbl)

	// set canonical fields
	for _, f := range fields {
		setField(z, f, t)
	}

	// store event if needed
	if isEvent || len(fields) == 0 {
		if z.Events == nil {
			z.Events = make(map[string]time.Time)
		}
		z.Events[lbl] = t
	}
}


⸻

🎯 5. What this handles (your real examples)

✅ Tisha B’Av start

Sunset (Shkiah)Fast Begins

→ normalized to:

Sunset (Shkiah) | Fast Begins

→ results:
	•	shkiah = t
	•	event: "Fast Begins"

⸻

✅ Yom Kippur start

Candle Lighting | Fast Begins

→ results:
	•	tzeis = t (your chosen mapping)
	•	event: "Candle Lighting"
	•	event: "Fast Begins"

⸻

✅ Yom Kippur end

Holiday/Fast Ends

→ results:
	•	tzeis = t

⸻

✅ Chametz deadlines

Finish Eating Chametz before

→ results:
	•	stored as event only

⸻

⚖️ 6. Is data-driven worth it?

You asked if it’s “useless churn.”

👉 In your case: it’s justified, but only at this size.

Why?
	•	You already have multiple overlapping variants
	•	Rules are growing horizontally, not vertically
	•	A switch will become harder to reason about than a rule list

But:
	•	This is still simple data, not a framework
	•	No need for regex engines, DSLs, etc.

⸻

🧩 7. Subtle but important benefit

This lets you do things like:

// later
if z.Events["Fast Begins"] == z.Shkiah {
	// meaningful semantic relationship
}

You preserve intent, not just timestamps.

⸻

✅ Final takeaway

You’ve now got the right model:
	•	Normalize → Split → Classify → Apply
	•	Allow multiple meanings per label
	•	Preserve unknowns as events
	•	Keep rules small and explicit

That’s about as robust as you can get without the feed giving you real structured semantics.

⸻

If you want next step, we could:
	•	formalize event keys (instead of raw strings)
	•	or build a small test harness from captured RSS to lock this behavior down before it drifts again
