package plan

import (
	"math"
	"testing"
	"time"
)

// day parses a calendar day for the tests and fails the test when the literal
// is not a valid date. A typo in a test table therefore shows up as a clear
// error instead of silently changing the expectation.
func day(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := ParseDate(value)
	if err != nil {
		t.Fatalf("ParseDate(%q) failed: %v", value, err)
	}
	return parsed
}

// dayPtr returns a pointer to the parsed calendar day. It is used for the
// optional end date of an entry.
func dayPtr(t *testing.T, value string) *time.Time {
	t.Helper()
	d := day(t, value)
	return &d
}

// recurring builds a valid recurring entry with the given properties.
func recurring(t *testing.T, id, accountID int64, name string, amountMinor int64, start string, unit Unit, interval int) Entry {
	t.Helper()
	return Entry{
		ID:          id,
		AccountID:   accountID,
		Name:        name,
		AmountMinor: amountMinor,
		StartDate:   day(t, start),
		Unit:        unit,
		Interval:    interval,
	}
}

// withEndsOn returns a copy of the entry with an end date.
func withEndsOn(t *testing.T, e Entry, endsOn string) Entry {
	t.Helper()
	e.EndsOn = dayPtr(t, endsOn)
	return e
}

// dateStrings formats the occurrences as YYYY-MM-DD for the comparison.
func dateStrings(list []time.Time) []string {
	out := make([]string, 0, len(list))
	for _, d := range list {
		out = append(out, FormatDate(d))
	}
	return out
}

// assertClose compares two amounts in Rappen with a small tolerance. The
// tolerance is only used for the unrounded demands, the expected values are
// derived by hand and rounded to two decimals.
func assertClose(t *testing.T, name string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.01 {
		t.Errorf("%s = %v, want %v (tolerance 0.01 Rappen)", name, got, want)
	}
}
