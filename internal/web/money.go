package web

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/HeiligerG/EasyPlan/internal/plan"
)

// The errors of the amount parser. They tell the caller why an amount was
// rejected.
var (
	// ErrAmountEmpty is returned for an amount without any character.
	ErrAmountEmpty = errors.New("web: amount is empty")
	// ErrAmountFormat is returned for an amount that is not a number.
	ErrAmountFormat = errors.New("web: amount is not a valid number")
	// ErrAmountNegative is returned for a negative amount.
	ErrAmountNegative = errors.New("web: amount must not be negative")
	// ErrAmountZero is returned for an amount of zero.
	ErrAmountZero = errors.New("web: amount must be greater than zero")
	// ErrAmountPrecision is returned for an amount with more than two
	// decimals.
	ErrAmountPrecision = errors.New("web: amount must have at most two decimals")
	// ErrAmountTooLarge is returned for an amount above plan.MaxAmountMinor.
	ErrAmountTooLarge = errors.New("web: amount is too large")
)

// FormatMoney formats an amount in Rappen as whole francs.
//
// The amount is rounded the way the whole application rounds it, with
// plan.RoundCHF, so the display and the calculation agree. The thousands are
// separated with an apostrophe, as in 2'510.
func FormatMoney(minor int64) string {
	return groupThousands(plan.RoundCHF(float64(minor)))
}

// FormatAmount formats an unrounded amount in Rappen as whole francs.
//
// The demands of the summaries are floats, because they are derived from the
// yearly costs. They are rounded the same way the exact amounts are rounded,
// with plan.RoundCHF, so the display and the calculation agree.
func FormatAmount(minor float64) string {
	return groupThousands(plan.RoundCHF(minor))
}

// ParseAmount parses an amount in francs and returns it in Rappen.
//
// The amount is read from the text, so no rounding error of a floating point
// number can occur. The apostrophe and the spaces are thousands separators
// and are ignored, the dot and the comma are decimal separators. If both
// kinds appear, the last one is the decimal separator and a separator of the
// other kind before it is a thousands separator. The same separator twice,
// as in "1.2.3", is rejected, because the decimal separator would be
// ambiguous.
//
// An empty amount, a negative amount, an amount of zero, an amount with more
// than two decimals and an amount above plan.MaxAmountMinor are rejected.
func ParseAmount(value string) (int64, error) {
	cleaned := strings.TrimSpace(value)
	cleaned = strings.TrimPrefix(cleaned, "+")
	if cleaned == "" {
		return 0, ErrAmountEmpty
	}
	if strings.HasPrefix(cleaned, "-") {
		return 0, ErrAmountNegative
	}

	cleaned = stripThousands(cleaned)
	if cleaned == "" {
		return 0, ErrAmountEmpty
	}

	whole, fraction, err := splitFraction(cleaned)
	if err != nil {
		return 0, err
	}
	if !isDigits(whole) {
		return 0, ErrAmountFormat
	}
	if len(fraction) > 2 {
		return 0, ErrAmountPrecision
	}

	francs, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		// The number does not fit into an int64, so it is far above the
		// largest amount of an entry.
		return 0, ErrAmountTooLarge
	}
	centimes := fractionToCentimes(fraction)

	if francs > (plan.MaxAmountMinor-int64(centimes))/100 {
		return 0, ErrAmountTooLarge
	}
	minor := francs*100 + int64(centimes)
	if minor == 0 {
		return 0, ErrAmountZero
	}
	return minor, nil
}

// FormatDate formats a calendar day for the display, as DD.MM.YYYY.
//
// The day is taken from the time zone of the timestamp itself, so the day of
// a timestamp of the local time zone stays the same and the day of a
// timestamp of the database stays the day it was stored with.
func FormatDate(t time.Time) string {
	year, month, day := t.Date()
	return fmt.Sprintf("%02d.%02d.%04d", day, int(month), year)
}

// FormatISODate formats a calendar day for a form field, as YYYY-MM-DD.
//
// It is the format the database uses, so a form can hand the value to the
// store without a conversion.
func FormatISODate(t time.Time) string {
	return plan.FormatDate(t)
}

// groupThousands formats a whole number of francs with an apostrophe
// between the thousands.
func groupThousands(value int64) string {
	negative := value < 0
	digits := strconv.FormatUint(absUint64(value), 10)

	var out strings.Builder
	if negative {
		out.WriteByte('-')
	}
	for i, digit := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			out.WriteByte('\'')
		}
		out.WriteRune(digit)
	}
	return out.String()
}

// absUint64 returns the absolute value of a signed number.
func absUint64(value int64) uint64 {
	if value < 0 {
		// The negation of the smallest int64 overflows, so the value is
		// negated in two steps.
		return uint64(-(value + 1)) + 1
	}
	return uint64(value)
}

// stripThousands removes the thousands separators of an amount.
//
// Besides the apostrophe and the typographic quote, the spaces of the
// non-breaking kinds are removed, so a copied amount keeps its value.
func stripThousands(value string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '\'', '\u2019', ' ', '\t', '\n', '\r', '\u00a0', '\u202f':
			return -1
		}
		return r
	}, value)
}

// splitFraction splits an amount into its whole part and its fraction.
//
// The last dot or comma is the decimal separator, a separator of the other
// kind before it is a thousands separator. The same separator before it would
// make the decimal separator ambiguous, so such a value is rejected.
func splitFraction(value string) (whole, fraction string, err error) {
	separator := strings.LastIndexAny(value, ".,")
	if separator < 0 {
		return value, "", nil
	}

	// The same separator twice would leave it open which one is the decimal
	// separator, so the value is rejected.
	if strings.IndexByte(value[:separator], value[separator]) >= 0 {
		return "", "", ErrAmountFormat
	}

	whole = strings.Map(func(r rune) rune {
		if r == '.' || r == ',' {
			return -1
		}
		return r
	}, value[:separator])
	fraction = value[separator+1:]
	if fraction == "" {
		return "", "", ErrAmountFormat
	}
	if !isDigits(fraction) {
		return "", "", ErrAmountFormat
	}
	return whole, fraction, nil
}

// fractionToCentimes converts the fraction of an amount into Rappen.
//
// The fraction must consist of digits only, which the parser checks before.
// A missing digit counts as zero, so "5" becomes 50 Rappen and an empty
// fraction becomes 0 Rappen. Digits beyond the second one are ignored, the
// parser rejects such a fraction before.
func fractionToCentimes(fraction string) int {
	centimes := 0
	for i := 0; i < 2; i++ {
		centimes *= 10
		if i < len(fraction) {
			centimes += int(fraction[i] - '0')
		}
	}
	return centimes
}

// isDigits reports whether the string consists of digits only.
func isDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
