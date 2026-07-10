package payroll

import "time"

// OpenPeriod is the single payout cycle FloRoll currently works within.
//
// The product recognises only two regular cycles — 1–15 and 16–end of month —
// and always operates in the context of the one that contains "today". This
// gives every screen (Today, Team, Payments) a shared, implicit period so the
// owner never has to pick a period just to see what needs doing now.
type OpenPeriod struct {
	Preset   Preset
	Period   Period
	DaysLeft int
}

// CurrentOpenPeriod resolves the open payout period for the given moment.
func CurrentOpenPeriod(now time.Time) OpenPeriod {
	preset := DefaultPreset(now)
	period, err := ResolvePeriod(preset, now.Format("2006-01"), "", "")
	if err != nil {
		return OpenPeriod{Preset: preset}
	}
	return OpenPeriod{
		Preset:   preset,
		Period:   period,
		DaysLeft: daysUntil(now, period.To),
	}
}

// Label is the human-readable name of the open period, e.g. "1–15 июля 2026".
func (o OpenPeriod) Label() string {
	return o.Period.Label()
}

// daysUntil counts whole days from now's date up to and including the "to" date.
// It never returns a negative value.
func daysUntil(now time.Time, to string) int {
	toDate, err := time.Parse("2006-01-02", to)
	if err != nil {
		return 0
	}
	today, err := time.Parse("2006-01-02", now.Format("2006-01-02"))
	if err != nil {
		return 0
	}
	days := int(toDate.Sub(today).Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}
