package plan

import (
	"strings"
	"testing"
	"time"
)

func TestParseDate_valid(t *testing.T) {
	cases := []struct {
		name  string
		value string
		want  time.Time
	}{
		{
			name:  "ordinary day",
			value: "2026-10-01",
			want:  time.Date(2026, time.October, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "first day of the year",
			value: "2026-01-01",
			want:  time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "last day of the year",
			value: "2026-12-31",
			want:  time.Date(2026, time.December, 31, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "leap day in a leap year",
			value: "2028-02-29",
			want:  time.Date(2028, time.February, 29, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "leap day in a year divisible by 400",
			value: "2000-02-29",
			want:  time.Date(2000, time.February, 29, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "last day of february in a common year",
			value: "2026-02-28",
			want:  time.Date(2026, time.February, 28, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseDate(tc.value)
			if err != nil {
				t.Fatalf("ParseDate(%q) returned an error: %v", tc.value, err)
			}
			if !got.Equal(tc.want) {
				t.Errorf("ParseDate(%q) = %v, want %v", tc.value, got, tc.want)
			}
			if got.Location() != time.UTC {
				t.Errorf("ParseDate(%q) location = %v, want UTC", tc.value, got.Location())
			}
			if got.Hour() != 0 || got.Minute() != 0 || got.Second() != 0 || got.Nanosecond() != 0 {
				t.Errorf("ParseDate(%q) = %v, want 00:00:00 UTC", tc.value, got)
			}
		})
	}
}

func TestParseDate_invalid(t *testing.T) {
	cases := []struct {
		name  string
		value string
	}{
		{name: "empty string", value: ""},
		{name: "no date at all", value: "not-a-date"},
		{name: "day 30 of february", value: "2026-02-30"},
		{name: "february 29 in a common year", value: "2026-02-29"},
		{name: "february 29 in a year divisible by 100 but not by 400", value: "2100-02-29"},
		{name: "day 0", value: "2026-01-00"},
		{name: "day 32", value: "2026-01-32"},
		{name: "month 0", value: "2026-00-10"},
		{name: "month 13", value: "2026-13-01"},
		{name: "single digit month", value: "2026-1-1"},
		{name: "single digit day", value: "2026-01-1"},
		{name: "slashes as separator", value: "2026/10/01"},
		{name: "with a time part", value: "2026-10-01T00:00:00Z"},
		{name: "without separators", value: "20261001"},
		{name: "trailing text", value: "2026-10-01x"},
		{name: "leading text", value: "x2026-10-01"},
		{name: "day of the week", value: "2026-10-01 (Thursday)"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseDate(tc.value)
			if err == nil {
				t.Fatalf("ParseDate(%q) = %v, want an error", tc.value, got)
			}
			if !strings.Contains(err.Error(), "invalid date") {
				t.Errorf("ParseDate(%q) error = %q, want it to mention an invalid date", tc.value, err.Error())
			}
		})
	}
}

func TestFormatDate(t *testing.T) {
	cases := []struct {
		name  string
		value string
	}{
		{name: "ordinary day", value: "2026-10-01"},
		{name: "leap day", value: "2028-02-29"},
		{name: "end of february in a common year", value: "2026-02-28"},
		{name: "first of january", value: "2026-01-01"},
		{name: "last of december", value: "2026-12-31"},
		{name: "end of february in a leap year", value: "2032-02-29"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatDate(day(t, tc.value))
			if got != tc.value {
				t.Errorf("FormatDate(%q) = %q, want %q", tc.value, got, tc.value)
			}
		})
	}
}

func TestFormatDate_uses_the_calendar_day_of_the_timestamp(t *testing.T) {
	// A timestamp in the afternoon stays on the same calendar day, the clock
	// time is cut off.
	afternoon := time.Date(2026, time.October, 1, 23, 45, 12, 0, time.UTC)
	if got := FormatDate(afternoon); got != "2026-10-01" {
		t.Errorf("FormatDate(23:45 UTC) = %q, want %q", got, "2026-10-01")
	}

	// A timestamp in a zone east of UTC keeps its own calendar day and is not
	// shifted to the previous or the next day.
	east := time.Date(2026, time.October, 1, 0, 30, 0, 0, time.FixedZone("UTC+2", 2*60*60))
	if got := FormatDate(east); got != "2026-10-01" {
		t.Errorf("FormatDate(00:30 UTC+2) = %q, want %q", got, "2026-10-01")
	}

	// The zero time is the first supported calendar day.
	if got := FormatDate(time.Time{}); got != "0001-01-01" {
		t.Errorf("FormatDate(zero) = %q, want %q", got, "0001-01-01")
	}
}

func TestParseDate_and_FormatDate_are_inverse(t *testing.T) {
	// A round trip over a leap year and a common year must not change a day.
	for _, value := range []string{
		"2026-01-01", "2026-02-28", "2026-03-01", "2026-12-31",
		"2028-02-28", "2028-02-29", "2028-03-01", "2032-02-29",
	} {
		parsed, err := ParseDate(value)
		if err != nil {
			t.Fatalf("ParseDate(%q) returned an error: %v", value, err)
		}
		if got := FormatDate(parsed); got != value {
			t.Errorf("round trip of %q = %q", value, got)
		}
	}
}
