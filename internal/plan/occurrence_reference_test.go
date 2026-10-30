package plan

import (
	"fmt"
	"slices"
	"testing"
	"time"
)

// The tests of this file compare the production code with a reference
// implementation that is written from scratch. The reference calculates the
// terms of an entry only from its start date and with its own arithmetic for
// the calendar days: it uses neither time.AddDate nor a function of the
// package under test. A mistake in the production code therefore cannot hide
// behind the same mistake in the reference.

// refMaxTerms is the number of terms the reference calculates at most. The
// windows of the test are at most 1000 days long and the shortest interval of
// the test is a week, so 1000 terms are far more than needed.
const refMaxTerms = 10000

// refDate is a calendar day of the reference implementation.
type refDate struct {
	year  int
	month int
	day   int
}

// refDayNumber returns the number of days between the calendar day of d and
// the 1st of January 1970. The formula counts the days of the proleptic
// Gregorian calendar and works for every year.
func refDayNumber(d refDate) int64 {
	y := int64(d.year)
	m := int64(d.month)
	if m <= 2 {
		y--
	}
	era := y / 400
	if y < 0 && y%400 != 0 {
		era--
	}
	yoe := y - era*400                     // the year inside the era, 0 to 399
	mp := (m + 9) % 12                     // the month inside the year, March first
	doy := (153*mp+2)/5 + int64(d.day) - 1 // the day inside the year
	doe := yoe*365 + yoe/4 - yoe/100 + doy
	return era*146097 + doe - 719468
}

// refDateFromDayNumber is the inverse of refDayNumber.
func refDateFromDayNumber(z int64) refDate {
	z += 719468
	era := z / 146097
	if z < 0 && z%146097 != 0 {
		era--
	}
	doe := z - era*146097
	yoe := (doe - doe/1460 + doe/36524 - doe/146096) / 365
	y := yoe + era*400
	doy := doe - (365*yoe + yoe/4 - yoe/100)
	mp := (5*doy + 2) / 153
	day := doy - (153*mp+2)/5 + 1
	month := mp + 3
	if mp >= 10 {
		month = mp - 9
	}
	if month <= 2 {
		y++
	}
	return refDate{year: int(y), month: int(month), day: int(day)}
}

// refDaysInMonth returns the number of days of a month.
func refDaysInMonth(year, month int) int {
	switch month {
	case 1, 3, 5, 7, 8, 10, 12:
		return 31
	case 4, 6, 9, 11:
		return 30
	case 2:
		if year%4 == 0 && (year%100 != 0 || year%400 == 0) {
			return 29
		}
		return 28
	}
	return 0
}

// refAddDays returns the calendar day that is the given number of days after
// d.
func refAddDays(d refDate, days int64) refDate {
	return refDateFromDayNumber(refDayNumber(d) + days)
}

// refAddMonths returns the calendar day that is the given number of months
// after d. The day of the month is clamped to the length of the target
// month, the 31st of January becomes the 28th of February.
func refAddMonths(d refDate, months int64) refDate {
	total := int64(d.year)*12 + int64(d.month) - 1 + months
	year := total / 12
	month := total%12 + 1
	day := d.day
	if last := refDaysInMonth(int(year), int(month)); day > last {
		day = last
	}
	return refDate{year: int(year), month: int(month), day: day}
}

// refKey formats a calendar day of the reference as YYYY-MM-DD.
func refKey(d refDate) string {
	return fmt.Sprintf("%04d-%02d-%02d", d.year, d.month, d.day)
}

// refOccurrence returns the n-th term of an entry, counted from the start
// date.
//
// The n-th term of a weekly entry is the start date plus n * interval weeks,
// the n-th term of a monthly entry is the start date plus n * interval months
// and the n-th term of a yearly entry is the start date plus n * interval
// years. The day of the month is clamped to the length of the month, the
// terms therefore never move forward when the month is too short.
func refOccurrence(unit Unit, start refDate, interval int64, n int64) refDate {
	switch unit {
	case Week:
		return refAddDays(start, n*interval*7)
	case Month:
		return refAddMonths(start, n*interval)
	case Year:
		return refAddMonths(start, n*interval*12)
	}
	return refDate{}
}

// refOccurrences returns the terms of an entry in the half open range
// [from, to). The end date is inclusive, a term on the end date is still a
// term of the entry.
func refOccurrences(unit Unit, start refDate, interval int64, from, to refDate, endsOn *refDate) []refDate {
	fromNumber := refDayNumber(from)
	toNumber := refDayNumber(to)
	endsOnNumber := int64(0)
	if endsOn != nil {
		endsOnNumber = refDayNumber(*endsOn)
	}
	var terms []refDate
	for n := int64(0); n < refMaxTerms; n++ {
		term := refOccurrence(unit, start, interval, n)
		number := refDayNumber(term)
		if number >= toNumber {
			break
		}
		if endsOn != nil && number > endsOnNumber {
			break
		}
		if number >= fromNumber {
			terms = append(terms, term)
		}
	}
	return terms
}

// refNext returns the first term of an entry that lies on or after the
// reference day. The second return value is false when the entry has no
// further term.
func refNext(unit Unit, start refDate, interval int64, from refDate, endsOn *refDate) (refDate, bool) {
	fromNumber := refDayNumber(from)
	endsOnNumber := int64(0)
	if endsOn != nil {
		endsOnNumber = refDayNumber(*endsOn)
	}
	for n := int64(0); n < refMaxTerms; n++ {
		term := refOccurrence(unit, start, interval, n)
		number := refDayNumber(term)
		if number < fromNumber {
			continue
		}
		if endsOn != nil && number > endsOnNumber {
			return refDate{}, false
		}
		return term, true
	}
	return refDate{}, false
}

// refDateOf returns the calendar day of a time.Time.
func refDateOf(t time.Time) refDate {
	return refDate{year: t.Year(), month: int(t.Month()), day: t.Day()}
}

// refKeys formats the terms of the production code as YYYY-MM-DD, without
// the help of the package under test.
func refKeys(terms []time.Time) []string {
	keys := make([]string, 0, len(terms))
	for _, term := range terms {
		keys = append(keys, refKey(refDateOf(term)))
	}
	return keys
}

// refWindow is a range of the brute force test. The bounds are calendar days,
// they are built from the start date and from the terms of the entry.
type refWindow struct {
	name   string
	from   refDate
	to     refDate
	endsOn *refDate
}

// refWindowsFor returns the ranges of the brute force test for one entry.
//
// The ranges are fixed, they are built from the start date and from the first
// terms of the entry, so the test always compares the same combinations.
// Some of the ranges end exactly on a term of the entry: the range is half
// open, such a term belongs to the next range. Some of the ranges end
// exactly on the end date of the entry: the end date is inclusive, such a
// term is still a term of the entry.
func refWindowsFor(unit Unit, start refDate, interval int64) []refWindow {
	first := refOccurrence(unit, start, interval, 1)
	second := refOccurrence(unit, start, interval, 2)
	third := refOccurrence(unit, start, interval, 3)
	endDate := refAddDays(start, 200)
	shortEndDate := refAddDays(start, 10)
	endOnTerm := second

	return []refWindow{
		{name: "the range lies before the start date", from: refAddDays(start, -400), to: refAddDays(start, -1)},
		{name: "the range starts before the start date", from: refAddDays(start, -1), to: refAddDays(start, 400)},
		{name: "the range lies after the start date", from: refAddDays(start, 45), to: refAddDays(start, 400)},
		{name: "the range is longer than a year and has an end date", from: refAddDays(start, -10), to: refAddDays(start, 1000), endsOn: &endDate},
		{name: "the end date lies inside the range", from: start, to: refAddDays(start, 500), endsOn: &shortEndDate},
		{name: "the end date lies on a term", from: refAddDays(start, -1), to: refAddDays(start, 1000), endsOn: &endOnTerm},
		{name: "the range ends on the second term", from: refAddDays(start, -1), to: second},
		{name: "the range starts on the second term", from: second, to: refAddDays(second, 400)},
		{name: "the range ends on the third term", from: first, to: third},
		{name: "the range holds the first three terms", from: start, to: refAddDays(third, 1)},
	}
}

// refStarts are the start dates of the brute force test. They are fixed, so
// the test always compares the same combinations. The dates cover the end of
// a month, a leap day and the 31st of March.
var refStarts = []string{
	"2024-01-29",
	"2024-01-30",
	"2024-01-31",
	"2024-02-29",
	"2025-03-31",
	"2026-01-01",
}

// TestReference_uses_the_same_calendar_arithmetic_as_the_standard_library
// proves that the reference implementation of this file counts the calendar
// days in the same way as the standard library. Without this test a mistake
// in the reference would make the brute force test compare two wrong
// calendars.
func TestReference_uses_the_same_calendar_arithmetic_as_the_standard_library(t *testing.T) {
	dates := []time.Time{
		time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC),
		time.Date(1969, time.December, 31, 0, 0, 0, 0, time.UTC),
		time.Date(2000, time.February, 29, 0, 0, 0, 0, time.UTC),
		time.Date(2024, time.February, 29, 0, 0, 0, 0, time.UTC),
		time.Date(2100, time.February, 28, 0, 0, 0, 0, time.UTC),
		time.Date(9999, time.December, 31, 0, 0, 0, 0, time.UTC),
	}
	// Every day of the years 2023 to 2026, so the formula is checked over a
	// whole leap year as well.
	for year := 2023; year <= 2026; year++ {
		for month := time.January; month <= time.December; month++ {
			for day := 1; day <= refDaysInMonth(year, int(month)); day++ {
				dates = append(dates, time.Date(year, month, day, 0, 0, 0, 0, time.UTC))
			}
		}
	}

	for _, moment := range dates {
		day := refDateOf(moment)
		want := moment.Unix() / 86400
		if got := refDayNumber(day); got != want {
			t.Fatalf("refDayNumber(%s) = %d, want %d", refKey(day), got, want)
		}
		if got := refDateFromDayNumber(want); got != day {
			t.Errorf("refDateFromDayNumber(%d) = %s, want %s", want, refKey(got), refKey(day))
		}
	}
}

// TestOccurrences_matches_the_brute_force_reference compares the terms of the
// production code with the reference implementation over a fixed set of
// combinations of start date, unit, interval and range.
func TestOccurrences_matches_the_brute_force_reference(t *testing.T) {
	units := []Unit{Week, Month, Year}
	intervals := []int{1, 2, 3, 5, 13}

	for _, start := range refStarts {
		for _, unit := range units {
			for _, interval := range intervals {
				name := fmt.Sprintf("%s_%s_%d", start, unit, interval)
				t.Run(name, func(t *testing.T) {
					entry := recurring(t, 1, 1, "Referenz", 100000, start, unit, interval)
					startDate := refDateOf(entry.StartDate)

					for _, window := range refWindowsFor(unit, startDate, int64(interval)) {
						from := window.from
						to := window.to
						endsOn := window.endsOn

						entry := entry
						if endsOn != nil {
							entry = withEndsOn(t, entry, refKey(*endsOn))
						}

						want := refOccurrences(unit, startDate, int64(interval), from, to, endsOn)
						wantKeys := make([]string, 0, len(want))
						for _, term := range want {
							wantKeys = append(wantKeys, refKey(term))
						}

						got := Occurrences(entry, day(t, refKey(from)), day(t, refKey(to)))
						if !slices.Equal(refKeys(got), wantKeys) {
							t.Errorf("Occurrences(%s .. %s) = %v, want %v (%s)",
								refKey(from), refKey(to), refKeys(got), wantKeys, window.name)
						}
					}
				})
			}
		}
	}
}

// TestNext_matches_the_brute_force_reference compares the next term of the
// production code with the reference implementation over the same fixed set
// of combinations.
func TestNext_matches_the_brute_force_reference(t *testing.T) {
	units := []Unit{Week, Month, Year}
	intervals := []int{1, 2, 3, 5, 13}

	for _, start := range refStarts {
		for _, unit := range units {
			for _, interval := range intervals {
				name := fmt.Sprintf("%s_%s_%d", start, unit, interval)
				t.Run(name, func(t *testing.T) {
					entry := recurring(t, 1, 1, "Referenz", 100000, start, unit, interval)
					startDate := refDateOf(entry.StartDate)

					for _, window := range refWindowsFor(unit, startDate, int64(interval)) {
						from := window.from
						to := window.to
						endsOn := window.endsOn

						entry := entry
						if endsOn != nil {
							entry = withEndsOn(t, entry, refKey(*endsOn))
						}

						// The reference day is the beginning and the end of the
						// range, so the term on the border of the range is
						// checked as well.
						for _, reference := range []refDate{from, to} {
							want, wantOK := refNext(unit, startDate, int64(interval), reference, endsOn)
							got, ok := Next(entry, day(t, refKey(reference)))
							if ok != wantOK {
								t.Errorf("Next(from %s) ok = %v, want %v (%s)", refKey(reference), ok, wantOK, window.name)
								continue
							}
							if !ok {
								continue
							}
							if formatted := refKey(refDateOf(got)); formatted != refKey(want) {
								t.Errorf("Next(from %s) = %s, want %s (%s)", refKey(reference), formatted, refKey(want), window.name)
							}
						}
					}
				})
			}
		}
	}
}
