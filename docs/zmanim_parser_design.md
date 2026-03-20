# Zmanim RSS Parsing -- Design & Implementation

## Problem Statement

The Chabad zmanim RSS feed is not a stable or strictly structured API.
The `<title>` (and `<guid>`) fields contain human-readable labels that
vary depending on:

-   Day type (weekday, Shabbos, Yom Tov, fast days)
-   Contextual meaning (e.g., "Fast Begins", "Holiday Ends")
-   Formatting inconsistencies (missing delimiters, combined tokens)
-   Rare or one-off events (e.g., chametz deadlines)

Examples of variability:

-   "Nightfall (Tzeit Hakochavim)"
-   "Shabbat Ends"
-   "Holiday Ends"
-   "Holiday/Fast Ends"
-   "Candle Lighting after"
-   "Sunset (Shkiah)Fast Begins" (missing delimiter)
-   "Dawn (Alot Hashachar) \| Fast Begins"
-   "Finish Eating Chametz before"

This makes exact string matching brittle and incomplete.

------------------------------------------------------------------------

## Key Observations

1.  **Prefix Extraction Is Correct** Current logic extracts everything
    before `" - "`:

    ``` go
    prefix := title[:dashIdx]
    ```

    This is good and should be preserved.

2.  **`guid` vs `title`** They contain effectively the same semantic
    label. No need to switch sources.

3.  **Labels Are Semi-Structured** They contain:

    -   Core zman concept (e.g., "Sunset", "Tzeit Hakochavim")
    -   Optional modifiers (e.g., "Fast Begins", "Holiday Ends")

4.  **Composite Labels Exist** Example:

        "Sunset (Shkiah)Fast Begins"

    This should logically map to:

    -   `shkiah`
    -   event: "Fast Begins"

5.  **Multiple Meanings Per Entry** Example:

        "Candle Lighting | Fast Begins"

    This may map to:

    -   `tzeis`
    -   event: "Fast Begins"

6.  **Unknown Labels Must Be Preserved** Some entries only appear
    occasionally and should not be discarded.

------------------------------------------------------------------------

## Design Goals

-   Deterministic where possible
-   Resilient to format variations
-   No need for exhaustive dataset
-   Preserve unknown data
-   Allow multiple semantic mappings per label

------------------------------------------------------------------------

## Proposed Solution: Hybrid Semantic Classifier

### Pipeline

1.  Normalize label
2.  Split into components
3.  Classify each component
4.  Map to:
    -   canonical zmanim fields
    -   dynamic events

------------------------------------------------------------------------

## Implementation

### 1. Normalize Input

Fix known formatting issues:

``` go
func normalizeLabel(s string) string {
    // Fix missing delimiter cases
    s = strings.ReplaceAll(s, ")Fast", ") | Fast")
    return s
}
```

------------------------------------------------------------------------

### 2. Split Composite Labels

``` go
func splitLabel(s string) []string {
    s = normalizeLabel(s)

    parts := strings.Split(s, "|")
    for i := range parts {
        parts[i] = strings.TrimSpace(parts[i])
    }
    return parts
}
```

------------------------------------------------------------------------

### 3. Classifier

Returns: - zmanim fields to set - whether this should also be stored as
an event

``` go
func classify(label string) (fields []string, isEvent bool) {
    switch {
    case strings.Contains(label, "Alot Hashachar"):
        return []string{"alos"}, false

    case strings.Contains(label, "Misheyakir"):
        return []string{"misheyakir"}, false

    case strings.Contains(label, "Sunrise"):
        return []string{"sunrise"}, false

    case strings.Contains(label, "Sunset"):
        return []string{"shkiah"}, false

    case strings.Contains(label, "Tzeit Hakochavim"),
         strings.Contains(label, "Shabbat Ends"),
         strings.Contains(label, "Holiday Ends"),
         strings.Contains(label, "Fast Ends"),
         strings.Contains(label, "Holiday/Fast Ends"):
        return []string{"tzeis"}, false

    case strings.Contains(label, "Candle Lighting"):
        // overlaps with tzeis in many cases
        return []string{"tzeis"}, true
    }

    return nil, true
}
```

------------------------------------------------------------------------

### 4. Integration into Existing Code

Replace `matchPrefix` usage with:

``` go
labels := splitLabel(prefix)

for _, lbl := range labels {
    fields, isEvent := classify(lbl)

    for _, f := range fields {
        setField(z, f, t)
    }

    if isEvent || len(fields) == 0 {
        if z.Events == nil {
            z.Events = make(map[string]time.Time)
        }
        z.Events[lbl] = t
    }
}
```

------------------------------------------------------------------------

## Handling `tzeis`

### Variants observed:

-   Nightfall (Tzeit Hakochavim)
-   Shabbat Ends
-   Holiday Ends
-   Fast Ends
-   Holiday/Fast Ends
-   Candle Lighting after

### Strategy:

All of the above map to:

``` go
"tzeis"
```

But: - "Candle Lighting" should ALSO be stored as an event - "Fast Ends"
should be preserved as event context

------------------------------------------------------------------------

## Handling Special Cases

### Chametz deadlines:

-   "Finish Eating Chametz before"
-   "Sell and Burn Chametz before"
-   "Nullify Chametz before"

These: - should NOT map to standard zmanim - should ALWAYS be stored in
`Events`

------------------------------------------------------------------------

## Data Model Extension

Add:

``` go
Events map[string]time.Time
```

to `ZmanimDay`.

------------------------------------------------------------------------

## Why Not Deterministic Mapping?

-   Too many variants
-   Requires multi-year (≈19-year cycle) dataset
-   Feed is not guaranteed stable
-   High maintenance cost

------------------------------------------------------------------------

## Why This Works

-   Robust to unseen labels
-   Handles composite meanings
-   Preserves all data
-   Minimal assumptions about feed stability
-   Easy to extend with new rules

------------------------------------------------------------------------

## Optional Improvements

-   Move classifier rules into data structure:

``` go
type Rule struct {
    Match func(string) bool
    Fields []string
    Event bool
}
```

-   Add debug logging behind a flag:

``` go
if debug && len(fields) == 0 {
    fmt.Fprintf(os.Stderr, "unclassified label: %q\n", lbl)
}
```

------------------------------------------------------------------------

## Summary

This approach replaces fragile prefix matching with:

-   normalization
-   tokenization
-   semantic classification

It ensures: - correctness for known zmanim - graceful handling of
unknown cases - future-proofing against RSS variability
