package web

import (
	"errors"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/HeiligerG/EasyPlan/internal/plan"
)

func TestFormatMoney_formats_whole_francs_with_apostrophes(t *testing.T) {
	cases := []struct {
		minor int64
		want  string
	}{
		{minor: 0, want: "0"},
		{minor: 1, want: "0"},
		{minor: 99, want: "1"},
		{minor: 100, want: "1"},
		{minor: 251000, want: "2'510"},
		{minor: 263791, want: "2'638"},
		{minor: 3165500, want: "31'655"},
		{minor: 123456789, want: "1'234'568"},
		{minor: 100000000000, want: "1'000'000'000"},
	}
	for _, tc := range cases {
		t.Run(strconv.FormatInt(tc.minor, 10), func(t *testing.T) {
			if got := FormatMoney(tc.minor); got != tc.want {
				t.Errorf("FormatMoney(%d) = %q, want %q", tc.minor, got, tc.want)
			}
		})
	}
}

func TestFormatMoney_rounds_half_away_from_zero(t *testing.T) {
	cases := []struct {
		minor int64
		want  string
	}{
		{minor: 449, want: "4"},
		{minor: 450, want: "5"},
		{minor: 499, want: "5"},
		{minor: 500, want: "5"},
		{minor: 501, want: "5"},
		{minor: -450, want: "-5"},
		{minor: -499, want: "-5"},
		{minor: -500, want: "-5"},
	}
	for _, tc := range cases {
		t.Run(strconv.FormatInt(tc.minor, 10), func(t *testing.T) {
			if got := FormatMoney(tc.minor); got != tc.want {
				t.Errorf("FormatMoney(%d) = %q, want %q", tc.minor, got, tc.want)
			}
		})
	}
}

func TestFormatMoney_survives_the_extreme_values(t *testing.T) {
	// The values are the extremes of the int64. The rounding happens on a
	// float64, whose resolution is 16 at that magnitude, so the last digits
	// of the amount are rounded as well.
	cases := []struct {
		name  string
		minor int64
		want  string
	}{
		{name: "smallest", minor: -9_223_372_036_854_775_808, want: "-92'233'720'368'547'760"},
		{name: "largest", minor: 9_223_372_036_854_775_807, want: "92'233'720'368'547'760"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := FormatMoney(tc.minor); got != tc.want {
				t.Errorf("FormatMoney(%d) = %q, want %q", tc.minor, got, tc.want)
			}
		})
	}
}

func TestFormatAmount_formats_the_unrounded_demands(t *testing.T) {
	cases := []struct {
		name  string
		minor float64
		want  string
	}{
		{name: "zero", minor: 0, want: "0"},
		{name: "exact", minor: 251000, want: "2'510"},
		{name: "rounded down", minor: 251040, want: "2'510"},
		{name: "rounded up", minor: 251060, want: "2'511"},
		{name: "half away from zero", minor: 250050, want: "2'501"},
		{name: "negative", minor: -251000, want: "-2'510"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := FormatAmount(tc.minor); got != tc.want {
				t.Errorf("FormatAmount(%v) = %q, want %q", tc.minor, got, tc.want)
			}
		})
	}
}

func TestParseAmount_reads_the_valid_amounts(t *testing.T) {
	cases := []struct {
		value string
		want  int64
	}{
		{value: "12.50", want: 1250},
		{value: "12,50", want: 1250},
		{value: "12", want: 1200},
		{value: "12.5", want: 1250},
		{value: "1'200", want: 120000},
		{value: "1’200", want: 120000},
		{value: "1200,5", want: 120050},
		{value: "1'200.75", want: 120075},
		{value: "1 200", want: 120000},
		{value: "1 200.75", want: 120075},
		{value: "  12.50  ", want: 1250},
		{value: "+12.50", want: 1250},
		{value: "0.01", want: 1},
		{value: "10000000000", want: plan.MaxAmountMinor},
	}
	for _, tc := range cases {
		t.Run(tc.value, func(t *testing.T) {
			got, err := ParseAmount(tc.value)
			if err != nil {
				t.Fatalf("ParseAmount(%q) failed: %v", tc.value, err)
			}
			if got != tc.want {
				t.Errorf("ParseAmount(%q) = %d, want %d", tc.value, got, tc.want)
			}
		})
	}
}

func TestParseAmount_rejects_the_invalid_amounts(t *testing.T) {
	cases := []struct {
		name  string
		value string
		want  error
	}{
		{name: "empty", value: "", want: ErrAmountEmpty},
		{name: "blank", value: "   ", want: ErrAmountEmpty},
		{name: "only an apostrophe", value: "'", want: ErrAmountEmpty},
		{name: "zero", value: "0", want: ErrAmountZero},
		{name: "zero with decimals", value: "0.00", want: ErrAmountZero},
		{name: "negative", value: "-5", want: ErrAmountNegative},
		{name: "negative with decimals", value: "-0.01", want: ErrAmountNegative},
		{name: "too many decimals", value: "12.505", want: ErrAmountPrecision},
		{name: "no number", value: "abc", want: ErrAmountFormat},
		{name: "no number with a separator", value: "abc.50", want: ErrAmountFormat},
		{name: "no number behind the separator", value: "12.abc", want: ErrAmountFormat},
		{name: "no number before the separator", value: ".50", want: ErrAmountFormat},
		{name: "no number before the comma", value: ",50", want: ErrAmountFormat},
		{name: "overflow", value: "99999999999999999999", want: ErrAmountTooLarge},
		{name: "above the maximum", value: "10000000000.01", want: ErrAmountTooLarge},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseAmount(tc.value)
			if err == nil {
				t.Fatalf("ParseAmount(%q) = %d, want the error %v", tc.value, got, tc.want)
			}
			if !errors.Is(err, tc.want) {
				t.Errorf("ParseAmount(%q) failed with %v, want %v", tc.value, err, tc.want)
			}
			if got != 0 {
				t.Errorf("ParseAmount(%q) = %d, want 0 next to the error", tc.value, got)
			}
		})
	}
}

// TestParseAmount_rejects_two_decimal_separators expects the parser to
// reject a value with two decimal separators.
//
// The same separator twice, as in "1.2.3", would leave it open which one is
// the decimal separator, so the parser rejects such a value. A separator of
// the other kind before the last one is a thousands separator and stays
// valid, as in "1,2.3".
func TestParseAmount_rejects_two_decimal_separators(t *testing.T) {
	if _, err := ParseAmount("1.2.3"); err == nil {
		t.Error("ParseAmount(\"1.2.3\") was accepted, want an error")
	}
}

// TestParseAmount_rejects_the_repeated_separator_with_a_format_error checks
// that a repeated decimal separator is rejected with the format error and not
// with another one.
//
// The order of the checks matters: a value with a repeated separator and
// three decimals behind the last separator ("12.505.505") is a format error
// too, because the separator is checked before the decimals.
func TestParseAmount_rejects_the_repeated_separator_with_a_format_error(t *testing.T) {
	cases := []struct {
		name  string
		value string
	}{
		{name: "two dots", value: "1.2.3"},
		{name: "two commas", value: "1,2,3"},
		{name: "three dots", value: "1.2.3.4"},
		{name: "dots with three decimals", value: "12.505.505"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseAmount(tc.value)
			if !errors.Is(err, ErrAmountFormat) {
				t.Errorf("ParseAmount(%q) failed with %v, want %v", tc.value, err, ErrAmountFormat)
			}
			if got != 0 {
				t.Errorf("ParseAmount(%q) = %d, want 0 next to the error", tc.value, got)
			}
		})
	}
}

// TestParseAmount_reads_the_mixed_separators checks that a thousands
// separator of the other kind before the last separator stays valid.
//
// The amount with the apostrophe as the thousands separator ("1'200.75") is
// covered by TestParseAmount_reads_the_valid_amounts.
func TestParseAmount_reads_the_mixed_separators(t *testing.T) {
	cases := []struct {
		value string
		want  int64
	}{
		{value: "1,200.50", want: 120050},
		{value: "1.200,50", want: 120050},
	}
	for _, tc := range cases {
		t.Run(tc.value, func(t *testing.T) {
			got, err := ParseAmount(tc.value)
			if err != nil {
				t.Fatalf("ParseAmount(%q) failed: %v", tc.value, err)
			}
			if got != tc.want {
				t.Errorf("ParseAmount(%q) = %d, want %d", tc.value, got, tc.want)
			}
		})
	}
}

// TestFractionToCentimes_converts_the_fraction_into_rappen checks the
// conversion of the fraction of an amount into Rappen.
//
// The fraction is empty or has one or two digits, the parser checks that
// before.
func TestFractionToCentimes_converts_the_fraction_into_rappen(t *testing.T) {
	cases := []struct {
		name     string
		fraction string
		want     int
	}{
		{name: "empty", fraction: "", want: 0},
		{name: "one digit", fraction: "5", want: 50},
		{name: "leading zero", fraction: "05", want: 5},
		{name: "two digits", fraction: "50", want: 50},
		{name: "largest fraction", fraction: "99", want: 99},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := fractionToCentimes(tc.fraction); got != tc.want {
				t.Errorf("fractionToCentimes(%q) = %d, want %d", tc.fraction, got, tc.want)
			}
		})
	}
}

// TestParseAmount_reads_the_last_separator_as_the_decimal_separator holds the
// documented behaviour of the parser down: the last dot or comma is the
// decimal separator, a separator of the other kind before it is a thousands
// separator.
//
// The value uses a dot and a comma, the two separators that the documentation
// names. A value with two of the same separator ("1.2.3") belongs to the
// rejected case of TestParseAmount_rejects_two_decimal_separators.
func TestParseAmount_reads_the_last_separator_as_the_decimal_separator(t *testing.T) {
	got, err := ParseAmount("1,2.3")
	if err != nil {
		t.Fatalf("ParseAmount(\"1,2.3\") failed: %v", err)
	}
	if got != 1230 {
		t.Errorf("ParseAmount(\"1,2.3\") = %d, want 1230", got)
	}
}

func TestParseAmount_rejects_the_documented_thousand_separator(t *testing.T) {
	// "1.200" has three digits behind the separator, so the parser reads
	// three decimals and rejects the value. The documented behaviour of the
	// money package is that a thousand separator is the apostrophe.
	_, err := ParseAmount("1.200")
	if !errors.Is(err, ErrAmountPrecision) {
		t.Errorf("ParseAmount(\"1.200\") failed with %v, want %v", err, ErrAmountPrecision)
	}
}

func TestFormatAmountInput_and_ParseAmount_round_trip(t *testing.T) {
	values := make([]int64, 0, 2100)
	for minor := int64(1); minor <= 2000; minor++ {
		values = append(values, minor)
	}
	values = append(values,
		1234,
		1250,
		4999,
		5000,
		125000,
		251000,
		999999,
		100000000,
		plan.MaxAmountMinor,
	)

	for _, minor := range values {
		t.Run(strconv.FormatInt(minor, 10), func(t *testing.T) {
			text := FormatAmountInput(minor)
			got, err := ParseAmount(text)
			if err != nil {
				t.Fatalf("ParseAmount(FormatAmountInput(%d)) = %q failed: %v", minor, text, err)
			}
			if got != minor {
				t.Errorf("ParseAmount(FormatAmountInput(%d)) = %d, want %d", minor, got, minor)
			}
		})
	}
}

func TestFormatAmountInput_writes_the_amount_without_a_thousand_separator(t *testing.T) {
	cases := []struct {
		minor int64
		want  string
	}{
		{minor: 1, want: "0.01"},
		{minor: 50, want: "0.50"},
		{minor: 1250, want: "12.50"},
		{minor: 1200, want: "12"},
		{minor: 120000, want: "1200"},
		{minor: 125000, want: "1250"},
		{minor: -1250, want: "-12.50"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			if got := FormatAmountInput(tc.minor); got != tc.want {
				t.Errorf("FormatAmountInput(%d) = %q, want %q", tc.minor, got, tc.want)
			}
		})
	}
}

func TestFormatDate_writes_the_day_for_the_display(t *testing.T) {
	cases := []struct {
		value string
		want  string
	}{
		{value: "2026-10-01", want: "01.10.2026"},
		{value: "2026-02-28", want: "28.02.2026"},
		{value: "2024-02-29", want: "29.02.2024"},
		{value: "2026-12-31", want: "31.12.2026"},
	}
	for _, tc := range cases {
		t.Run(tc.value, func(t *testing.T) {
			if got := FormatDate(mustDay(t, tc.value)); got != tc.want {
				t.Errorf("FormatDate(%s) = %q, want %q", tc.value, got, tc.want)
			}
		})
	}
}

func TestFormatISODate_writes_the_day_for_the_form(t *testing.T) {
	cases := []struct {
		value string
		want  string
	}{
		{value: "2026-10-01", want: "2026-10-01"},
		{value: "2026-02-28", want: "2026-02-28"},
		{value: "2024-02-29", want: "2024-02-29"},
	}
	for _, tc := range cases {
		t.Run(tc.value, func(t *testing.T) {
			if got := FormatISODate(mustDay(t, tc.value)); got != tc.want {
				t.Errorf("FormatISODate(%s) = %q, want %q", tc.value, got, tc.want)
			}
		})
	}
}

// TestFormatDate_keeps_the_day_of_the_timestamp checks that the day of a
// timestamp is taken from the time zone of the timestamp itself.
func TestFormatDate_keeps_the_day_of_the_timestamp(t *testing.T) {
	zurich, err := time.LoadLocation("Europe/Zurich")
	if err != nil {
		t.Fatalf("time.LoadLocation() failed: %v", err)
	}
	// 1.10.2026, 00:30 in Zurich is still the 30.9. in UTC.
	moment := time.Date(2026, time.October, 1, 0, 30, 0, 0, zurich)
	if got := FormatDate(moment); got != "01.10.2026" {
		t.Errorf("FormatDate(%v) = %q, want %q", moment, got, "01.10.2026")
	}
	if got := FormatDate(moment.UTC()); got != "30.09.2026" {
		t.Errorf("FormatDate(%v) = %q, want %q", moment.UTC(), got, "30.09.2026")
	}
}

// TestParseAmount_keeps_the_amount_exact checks that the parser does not lose
// precision on the way from the text to the Rappen.
func TestParseAmount_keeps_the_amount_exact(t *testing.T) {
	const want int64 = 123456789
	got, err := ParseAmount("1'234'567.89")
	if err != nil {
		t.Fatalf("ParseAmount() failed: %v", err)
	}
	if got != want {
		t.Errorf("ParseAmount() = %d, want %d", got, want)
	}
	if text := fmt.Sprintf("%d", got); text != "123456789" {
		t.Errorf("the amount %d does not hold every Rappen", got)
	}
}
