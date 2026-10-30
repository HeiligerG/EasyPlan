package plan

import (
	"math"
	"time"
)

// daysPerYear is the average length of a year used to derive the weekly and
// the daily demand from the yearly costs.
const daysPerYear = 365.25

// Need holds the demand of a set of entries.
//
// The values are unrounded amounts in Rappen.
type Need struct {
	// Month is the demand per month.
	Month float64
	// Week is the demand per week.
	Week float64
	// Day is the demand per day.
	Day float64
}

// Totals returns the demand of the given entries.
//
// The yearly costs of the entries are summed up first, the monthly, weekly
// and daily demand is derived from that sum. One-off entries are no
// recurring demand and are not part of the result.
//
// Totals counts every entry it is given; filtering the entries is up to the
// caller. The result is never rounded, use RoundCHF for the display.
func Totals(entries []Entry) Need {
	var annual float64
	for _, e := range entries {
		annual += annualCost(e)
	}
	return Need{
		Month: annual / 12,
		Week:  annual * 7 / daysPerYear,
		Day:   annual / daysPerYear,
	}
}

// ActiveOn returns the entries that are still active on today.
//
// Entries whose end date lies before today are removed. Once entries are kept
// unchanged, because their end date is ignored by the calculations. Totals
// handles the once entries itself, so the caller decides which entries it
// passes on.
func ActiveOn(entries []Entry, today time.Time) []Entry {
	day := dayStart(today)
	active := make([]Entry, 0, len(entries))
	for _, e := range entries {
		if e.Unit != Once && e.EndsOn != nil && dayStart(*e.EndsOn).Before(day) {
			continue
		}
		active = append(active, e)
	}
	return active
}

// RoundCHF converts an amount in Rappen into whole CHF, rounding half away
// from zero.
//
// Zero is returned for zero. Amounts that do not fit into an int64 are
// clamped to math.MaxInt64 and math.MinInt64, a NaN is returned as zero.
func RoundCHF(minor float64) int64 {
	if math.IsNaN(minor) {
		return 0
	}
	chf := math.Round(minor / 100)
	if chf >= math.MaxInt64 {
		return math.MaxInt64
	}
	if chf <= math.MinInt64 {
		return math.MinInt64
	}
	return int64(chf)
}

// annualCost returns the yearly cost of an entry in Rappen.
func annualCost(e Entry) float64 {
	amount := float64(e.AmountMinor)
	interval := float64(e.Interval)
	if interval <= 0 {
		return 0
	}
	switch e.Unit {
	case Week:
		return amount * daysPerYear / 7 / interval
	case Month:
		return amount * 12 / interval
	case Year:
		return amount / interval
	}
	return 0
}
