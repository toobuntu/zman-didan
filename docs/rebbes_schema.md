# Lubavitcher Rebbes and Related Figures — Reference Data

Source file: `internal/embeddata/files/rebbes.json`

This file is the single source of truth for:
- Rebbe name normalization (verbose Hebcal feed names → standard honorifics)
- Birthday "(N years ago)" annotations in calendar descriptions
- Yahrzeit death-year disambiguation when two figures share a verbose name

---

## Schema

```json
{
  "honorific":           "the Alter Rebbe",
  "verbose_names":       ["R. Schneur Zalman of Liadi"],
  "birth_year":          5505,
  "birth_date_hebrew":   "18 Elul 5505",
  "birth_date_gregorian":"1745-09-04",
  "death_year":          5572,
  "death_date_hebrew":   "24 Tevet 5572",
  "death_date_gregorian":"1812-12-27"
}
```

- `honorific` — standard Chabad name used in calendar output; unique
- `verbose_names` — all name forms observed in `chabad-special-dates.ics`; keys for runtime lookup
- `birth_year` / `death_year` — Hebrew calendar year integers; used by application logic
- `birth_date_hebrew` / `death_date_hebrew` — full Hebrew date for reference; not read by code
- `birth_date_gregorian` / `death_date_gregorian` — proleptic Gregorian (ISO 8601); not read by code

**Important**: `verbose_names` must match the raw Hebcal feed strings exactly.
`reformatDescription` in `specialdates/merge.go` runs before rebbe name normalization,
so it looks up by verbose name. Do not change verbose names to match honorifics.

---

## Figures

### 1. The Alter Rebbe
**R. Schneur Zalman of Liadi** — founder of Chabad Chassidus, author of the Tanya and Shulchan Aruch HaRav

| | Hebrew | Gregorian |
|--|--------|-----------|
| Born | 18 Elul 5505 | 4 September 1745 |
| Died | 24 Tevet 5572 | 27 December 1812 |

---

### 2. The Mitteler Rebbe
**R. DovBer Schneuri** — son of the Alter Rebbe; his yahrzeit (9 Kislev) and birthday (9 Kislev) fall on the same date

| | Hebrew | Gregorian |
|--|--------|-----------|
| Born | 9 Kislev 5534 | 9 November 1773 |
| Died | 9 Kislev 5588 | 16 November 1827 |

---

### 3. The Tzemach Tzeddek
**R. Menachem Mendel Schneersohn**

| | Hebrew | Gregorian |
|--|--------|-----------|
| Born | 29 Elul 5549 | 9 September 1789 |
| Died | 13 Nisan 5626 | 1 April 1866 |

---

### 4. The Rebbe Maharash
**R. Shmuel Schneersohn**

| | Hebrew | Gregorian |
|--|--------|-----------|
| Born | 2 Iyar 5594 | 2 May 1834 |
| Died | 13 Tishrei 5643 | 4 October 1882 |

---

### 5. The Rebbe Rashab
**R. Shalom DovBer Schneersohn** — founder of the Tomchei Temimim yeshiva network

| | Hebrew | Gregorian |
|--|--------|-----------|
| Born | 20 Cheshvan 5621 | 4 November 1860 |
| Died | 2 Nisan 5680 | 21 March 1920 |

---

### 6. The Frierdiker Rebbe
**R. Yosef Yitzchak Schneersohn**

| | Hebrew | Gregorian |
|--|--------|-----------|
| Born | 12 Tammuz 5640 | 21 June 1880 |
| Died | 10 Shevat 5710 | 28 January 1950 |

---

### 7. The Lubavitcher Rebbe
**R. Menachem Mendel Schneerson**

| | Hebrew | Gregorian |
|--|--------|-----------|
| Born | 11 Nisan 5662 | 18 April 1902 |
| Died | 3 Tammuz 5754 | 12 June 1994 |

---

### 8. R. Levi Yitzchak Schneerson
Father of the Lubavitcher Rebbe; suffered greatly for strengthening Yiddishkeit in Soviet Russia

| | Hebrew | Gregorian |
|--|--------|-----------|
| Born | 17 Nisan 5638 | 20 April 1878 |
| Died | 20 Av 5704 | 9 August 1944 |

---

### 9. Rebbetzin Chana Schneerson
Mother of the Lubavitcher Rebbe

| | Hebrew | Gregorian |
|--|--------|-----------|
| Born | 28 Nisan 5640 | 28 April 1880 |
| Died | 6 Tishrei 5725 | 12 October 1964 |

---

### 10. The Baal Shem Tov
**R. Yisrael ben Eliezer** — founder of Chassidus

| | Hebrew | Gregorian |
|--|--------|-----------|
| Born | 18 Elul 5458 | 27 August 1698 |
| Died | 6 Sivan 5520 | 23 May 1760 |

---

## Notes on date accuracy

Gregorian dates are derived from the Hebrew calendar using standard astronomical
conversion. Pre-1752 English calendar reform dates are expressed in the proleptic
Gregorian calendar, consistent with ISO 8601. Sources: Chabad.org biographies,
Encyclopedia Judaica. Some dates (particularly birth dates for earlier figures)
may vary by one day between sources depending on the conversion algorithm used.
