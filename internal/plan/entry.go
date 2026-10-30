package plan

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Unit describes the recurrence of an Entry.
type Unit string

const (
	// Once marks an entry that occurs exactly once, on its start date.
	Once Unit = "once"
	// Week marks an entry that recurs every Interval weeks.
	Week Unit = "week"
	// Month marks an entry that recurs every Interval months.
	Month Unit = "month"
	// Year marks an entry that recurs every Interval years.
	Year Unit = "year"
)

const (
	// MaxAmountMinor is the largest amount of a single occurrence in
	// Rappen that an entry may have. It is 10^12 Rappen, so the sums of the
	// amounts in the calculations stay inside the int64 range.
	MaxAmountMinor = 1_000_000_000_000
	// MaxInterval is the largest number of units between two occurrences
	// that an entry may have.
	MaxInterval = 10_000
)

// Valid reports whether u is one of the known units.
func (u Unit) Valid() bool {
	switch u {
	case Once, Week, Month, Year:
		return true
	}
	return false
}

// Entry is a recurring expense of an account.
type Entry struct {
	// ID is the database identifier of the entry.
	ID int64
	// AccountID is the database identifier of the owning account.
	AccountID int64
	// Name is the label of the entry.
	Name string
	// AmountMinor is the amount of a single occurrence in Rappen. It is
	// always positive and must not be greater than MaxAmountMinor; the sign
	// is added by the display.
	AmountMinor int64
	// StartDate is the first occurrence of the entry.
	StartDate time.Time
	// Unit is the recurrence unit of the entry.
	Unit Unit
	// Interval is the number of units between two occurrences. It must not
	// be greater than MaxInterval.
	Interval int
	// EndsOn is the last date on which the entry occurs. A nil value means
	// the entry is open-ended.
	EndsOn *time.Time
	// Notes is a free-form note.
	Notes string
}

// Validate checks the entry against the constraints of the data model.
//
// It reports an error for a non-positive amount, an amount above
// MaxAmountMinor, a non-positive interval, an interval above MaxInterval, an
// unknown unit, a blank name, a start date outside the supported year range
// and an end date outside the supported year range or before the start date.
// The dates are compared as calendar days, so a time of the day never
// decides whether an entry is valid.
//
// For the unit "once" the interval and the end date are ignored by the
// calculations. The interval of a once entry is still checked for a positive
// value, but not for its upper limit, because the interval is not used.
func (e Entry) Validate() error {
	if e.AmountMinor <= 0 {
		return fmt.Errorf("amount must be greater than zero, got %d", e.AmountMinor)
	}
	if e.AmountMinor > MaxAmountMinor {
		return fmt.Errorf("amount must not be greater than %d, got %d", MaxAmountMinor, e.AmountMinor)
	}
	if !e.Unit.Valid() {
		return fmt.Errorf("unknown unit %q", e.Unit)
	}
	if e.Interval <= 0 {
		return fmt.Errorf("interval must be greater than zero, got %d", e.Interval)
	}
	if e.Unit != Once && e.Interval > MaxInterval {
		return fmt.Errorf("interval must not be greater than %d, got %d", MaxInterval, e.Interval)
	}
	if strings.TrimSpace(e.Name) == "" {
		return errors.New("name must not be empty")
	}
	if !supported(dayStart(e.StartDate)) {
		return fmt.Errorf("start_date %s is outside the supported year range %d-%d", FormatDate(e.StartDate), minYear, maxYear)
	}
	if e.EndsOn != nil {
		if !supported(dayStart(*e.EndsOn)) {
			return fmt.Errorf("ends_on %s is outside the supported year range %d-%d", FormatDate(*e.EndsOn), minYear, maxYear)
		}
		if dayStart(*e.EndsOn).Before(dayStart(e.StartDate)) {
			return fmt.Errorf("ends_on %s is before start_date %s", FormatDate(*e.EndsOn), FormatDate(e.StartDate))
		}
	}
	return nil
}
