package web

import (
	"testing"
	"time"

	"github.com/HeiligerG/EasyPlan/internal/plan"
)

// TestEntryProblem_maps_every_problem_of_the_plan_package checks that every
// problem of plan.Entry.Validate has a field of the form.
//
// The plan package checks the entry and reports the problem in English. The
// web layer repeats the checks and reports them in German on the field of the
// form, so the visitor sees the problem next to the value he has to change.
// Every problem of the plan package needs such a field, otherwise the error
// would fall back to the message of the whole page.
func TestEntryProblem_maps_every_problem_of_the_plan_package(t *testing.T) {
	base := plan.Entry{
		AccountID:   1,
		Name:        "Miete",
		AmountMinor: 200000,
		StartDate:   mustDay(t, "2026-01-01"),
		Unit:        plan.Month,
		Interval:    1,
	}

	cases := []struct {
		name      string
		entry     plan.Entry
		values    EntryForm
		wantField string
	}{
		{
			name:      "the amount is not greater than zero",
			entry:     withAmount(base, 0),
			values:    withValue(validEntryForm(1), "amount", "0"),
			wantField: fieldAmount,
		},
		{
			name:      "the amount is too large",
			entry:     withAmount(base, plan.MaxAmountMinor+1),
			values:    withValue(validEntryForm(1), "amount", "10000000000.01"),
			wantField: fieldAmount,
		},
		{
			name:      "the name is empty",
			entry:     withName(base, "   "),
			values:    withValue(validEntryForm(1), "name", ""),
			wantField: fieldName,
		},
		{
			name:      "the unit is unknown",
			entry:     withUnit(base, "day"),
			values:    withValue(validEntryForm(1), "unit", "day"),
			wantField: fieldUnit,
		},
		{
			name:      "the interval is not greater than zero",
			entry:     withInterval(base, 0),
			values:    withValue(validEntryForm(1), "interval", "0"),
			wantField: fieldInterval,
		},
		{
			name:      "the interval is too large",
			entry:     withInterval(base, plan.MaxInterval+1),
			values:    withValue(validEntryForm(1), "interval", "10001"),
			wantField: fieldInterval,
		},
		{
			name:      "the start date is outside of the supported years",
			entry:     withStart(base, time.Date(0, time.January, 1, 0, 0, 0, 0, time.UTC)),
			values:    withValue(validEntryForm(1), "start_date", "0000-01-01"),
			wantField: fieldStart,
		},
		{
			name:      "the end date is outside of the supported years",
			entry:     withEndsOn(base, time.Date(0, time.January, 1, 0, 0, 0, 0, time.UTC)),
			values:    withValue(validEntryForm(1), "ends_on", "0000-01-01"),
			wantField: fieldEndsOn,
		},
		{
			name:      "the end date lies before the start date",
			entry:     withEndsOn(base, mustDay(t, "2025-12-31")),
			values:    withValue(validEntryForm(1), "ends_on", "2025-12-31"),
			wantField: fieldEndsOn,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.entry.Validate(); err == nil {
				t.Fatalf("the entry %+v is valid, want an error of the plan package", tc.entry)
			}

			field, message := entryProblem(tc.entry)
			if field != tc.wantField {
				t.Errorf("the problem of the entry is on the field %q, want %q", field, tc.wantField)
			}
			if message == "" {
				t.Error("the problem of the entry has no message")
			}

			// The form reports the problem on the same field and does not
			// fall back to the message of the whole page.
			result := ValidateEntryForm(tc.values, activeTestAccounts(), testAccounts())
			if result.Error != "" {
				t.Errorf("the form has the message %q of the page, want none", result.Error)
			}
			if _, ok := result.Errors[tc.wantField]; !ok {
				t.Errorf("the form has the errors %v, want an error of the field %q", result.Errors, tc.wantField)
			}
		})
	}
}

// TestEntryProblem_reports_no_problem_for_a_valid_entry checks the other
// direction: an entry that the plan package accepts has no field and no
// message.
func TestEntryProblem_reports_no_problem_for_a_valid_entry(t *testing.T) {
	base := plan.Entry{
		AccountID:   1,
		Name:        "Miete",
		AmountMinor: 200000,
		StartDate:   mustDay(t, "2026-01-01"),
		Unit:        plan.Month,
		Interval:    1,
	}

	cases := []struct {
		name  string
		entry plan.Entry
	}{
		{name: "a monthly entry", entry: base},
		{name: "an entry with an end date", entry: withEndsOn(base, mustDay(t, "2026-12-31"))},
		{name: "the largest amount", entry: withAmount(base, plan.MaxAmountMinor)},
		{name: "the largest interval", entry: withInterval(base, plan.MaxInterval)},
		{
			name:  "a once entry with a large interval",
			entry: withInterval(withUnit(base, plan.Once), plan.MaxInterval+1),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.entry.Validate(); err != nil {
				t.Fatalf("the entry %+v is not valid: %v", tc.entry, err)
			}
			field, message := entryProblem(tc.entry)
			if field != "" || message != "" {
				t.Errorf("the problem of the entry is %q on the field %q, want none", message, field)
			}
		})
	}
}

// withAmount returns a copy of the entry with the given amount.
func withAmount(entry plan.Entry, amount int64) plan.Entry {
	entry.AmountMinor = amount
	return entry
}

// withName returns a copy of the entry with the given name.
func withName(entry plan.Entry, name string) plan.Entry {
	entry.Name = name
	return entry
}

// withUnit returns a copy of the entry with the given unit.
func withUnit(entry plan.Entry, unit plan.Unit) plan.Entry {
	entry.Unit = unit
	return entry
}

// withInterval returns a copy of the entry with the given interval.
func withInterval(entry plan.Entry, interval int) plan.Entry {
	entry.Interval = interval
	return entry
}

// withStart returns a copy of the entry with the given start date.
func withStart(entry plan.Entry, start time.Time) plan.Entry {
	entry.StartDate = start
	return entry
}

// withEndsOn returns a copy of the entry with the given end date.
func withEndsOn(entry plan.Entry, endsOn time.Time) plan.Entry {
	entry.EndsOn = &endsOn
	return entry
}

// withValue returns a copy of the values of the form with the given value of
// the named field.
func withValue(values EntryForm, field, value string) EntryForm {
	switch field {
	case "name":
		values.Name = value
	case "amount":
		values.Amount = value
	case "start_date":
		values.StartDate = value
	case "unit":
		values.Unit = value
	case "interval":
		values.Interval = value
	case "ends_on":
		values.EndsOn = value
	}
	return values
}
