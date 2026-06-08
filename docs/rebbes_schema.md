<!--
SPDX-FileCopyrightText: Copyright 2026 Todd Schulman

SPDX-License-Identifier: GPL-3.0-or-later
-->

# Lubavitcher Rebbes and Related Figures — Reference Data

Source file: `internal/embeddata/files/rebbes.json`

This file is the single source of truth for rebbe biographical data. It serves:
- Rebbe name normalization: verbose Hebcal feed names → standard honorifics
- "(N years ago)" annotations in birthday and histalkus descriptions
- Histalkus year disambiguation when two figures share a verbose name

---

## Schema

```json
{
  "honorific":            "the Alter Rebbe",
  "verbose_names":        ["R. Schneur Zalman of Liadi"],
  "huledes_year":         5505,
  "dob_hebrew":           "18 Elul 5505",
  "dob_gregorian":        "1745-09-04",
  "histalkus_year":       5572,
  "histalkus_hebrew":     "24 Tevet 5572",
  "histalkus_gregorian":  "1812-12-27"
}
```

### Fields

| Field | Type | Used at runtime | Purpose |
|---|---|---|---|
| `honorific` | string | ✓ | Output name; unique across all entries |
| `verbose_names` | []string | ✓ | Match raw Hebcal feed strings for normalization |
| `huledes_year` | int | ✓ | Compute "(N years ago)" for birthday descriptions |
| `dob_gregorian` | string ISO 8601 | ✓ | Enrich birthday descriptions with Gregorian date |
| `histalkus_year` | int | ✓ | Compute "(N years ago)"; disambiguate shared names |
| `histalkus_gregorian` | string ISO 8601 | ✓ | Enrich histalkus descriptions with Gregorian date |
| `dob_hebrew` | string | — | Reference only |
| `histalkus_hebrew` | string | — | Reference only |

**Important**: `verbose_names` must match the raw Hebcal feed strings exactly.
`reformatDescription` in `specialdates/merge.go` runs before rebbe name normalization,
so it looks up by verbose name. Do not change verbose names without verifying
against a live `chabad-special-dates.ics` feed.

---

## Figures

### 1. The Baal Shem Tov
**R. Yisrael ben Eliezer** — founder of Chassidus

| | Hebrew | Gregorian |
|--|--------|-----------|
| Yom Huledes | 18 Elul 5458 | 27 Aug 1698 |
| Yom Histalkus | 6 Sivan 5520 | 23 May 1760 |

---

### 2. The Alter Rebbe
**R. Schneur Zalman of Liadi** — founder of Chabad Chassidus; author of the Tanya and Shulchan Aruch HaRav

| | Hebrew | Gregorian |
|--|--------|-----------|
| Yom Huledes | 18 Elul 5505 | 4 Sep 1745 |
| Yom Histalkus | 24 Tevet 5572 | 27 Dec 1812 |

---

### 3. The Mitteler Rebbe
**R. DovBer Schneuri** — son of the Alter Rebbe; his histalkus (9 Kislev) and yom huledes (9 Kislev) fall on the same calendar date in different years

| | Hebrew | Gregorian |
|--|--------|-----------|
| Yom Huledes | 9 Kislev 5534 | 9 Nov 1773 |
| Yom Histalkus | 9 Kislev 5588 | 16 Nov 1827 |

---

### 4. The Tzemach Tzedek
**R. Menachem Mendel Schneersohn**

| | Hebrew | Gregorian |
|--|--------|-----------|
| Yom Huledes | 29 Elul 5549 | 9 Sep 1789 |
| Yom Histalkus | 13 Nisan 5626 | 1 Apr 1866 |

---

### 5. The Rebbe Maharash
**R. Shmuel Schneersohn**

| | Hebrew | Gregorian |
|--|--------|-----------|
| Yom Huledes | 2 Iyar 5594 | 2 May 1834 |
| Yom Histalkus | 13 Tishrei 5643 | 4 Oct 1882 |

---

### 6. The Rebbe Rashab
**R. Shalom DovBer Schneersohn** — founder of the Tomchei Temimim yeshiva network

| | Hebrew | Gregorian |
|--|--------|-----------|
| Yom Huledes | 20 Cheshvan 5621 | 4 Nov 1860 |
| Yom Histalkus | 2 Nisan 5680 | 21 Mar 1920 |

---

### 7. The Frierdiker Rebbe
**R. Yosef Yitzchak Schneersohn**

| | Hebrew | Gregorian |
|--|--------|-----------|
| Yom Huledes | 12 Tammuz 5640 | 21 Jun 1880 |
| Yom Histalkus | 10 Shevat 5710 | 28 Jan 1950 |

---

### 8. Rebbetzin Chana
**Rebbetzin Chana Schneerson** — mother of the Lubavitcher Rebbe; suffered greatly alongside her husband in Soviet Russia

| | Hebrew | Gregorian |
|--|--------|-----------|
| Yom Huledes | 28 Nisan 5640 | 28 Apr 1880 |
| Yom Histalkus | 6 Tishrei 5725 | 12 Oct 1964 |

---

### 9. The Rebbe
**R. Menachem Mendel Schneerson** — seventh Lubavitcher Rebbe

| | Hebrew | Gregorian |
|--|--------|-----------|
| Yom Huledes | 11 Nisan 5662 | 18 Apr 1902 |
| Yom Histalkus | 3 Tammuz 5754 | 12 Jun 1994 |

---

### 10. The Rebbetzin
**Rebbetzin Chaya Mushka Schneerson** — wife of the Lubavitcher Rebbe

| | Hebrew | Gregorian |
|--|--------|-----------|
| Yom Huledes | 25 Adar 5661 | 16 Mar 1901 |
| Yom Histalkus | 22 Shevat 5748 | 13 Feb 1988 |

*Note: Rebbetzin Chaya Mushka's dates should be verified against chabad.org biographies.*

---

### 11. R. Levi Yitzchak Schneerson
Father of the Lubavitcher Rebbe; imprisoned and exiled by the Soviet regime for strengthening Yiddishkeit

| | Hebrew | Gregorian |
|--|--------|-----------|
| Yom Huledes | 17 Nisan 5638 | 20 Apr 1878 |
| Yom Histalkus | 20 Av 5704 | 9 Aug 1944 |

---

## Notes on date accuracy

Gregorian dates are derived from the Hebrew calendar using standard conversion.
Pre-1752 dates are expressed in the proleptic Gregorian calendar (ISO 8601).
Sources: Chabad.org biographies, Encyclopedia Judaica. Some dates may vary
by one day between sources depending on the conversion algorithm.
