package payroll

import (
	"testing"
	"time"
)

func TestCurrentOpenPeriodFirstHalf(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	op := CurrentOpenPeriod(now)

	if op.Preset != PresetFirstHalf {
		t.Fatalf("preset = %s, want %s", op.Preset, PresetFirstHalf)
	}
	if op.Period.From != "2026-07-01" || op.Period.To != "2026-07-15" {
		t.Fatalf("period = %s..%s, want 2026-07-01..2026-07-15", op.Period.From, op.Period.To)
	}
	if op.DaysLeft != 5 {
		t.Fatalf("daysLeft = %d, want 5", op.DaysLeft)
	}
}

func TestCurrentOpenPeriodSecondHalf(t *testing.T) {
	now := time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
	op := CurrentOpenPeriod(now)

	if op.Preset != PresetSecondHalf {
		t.Fatalf("preset = %s, want %s", op.Preset, PresetSecondHalf)
	}
	if op.Period.From != "2026-07-16" || op.Period.To != "2026-07-31" {
		t.Fatalf("period = %s..%s, want 2026-07-16..2026-07-31", op.Period.From, op.Period.To)
	}
	if op.DaysLeft != 11 {
		t.Fatalf("daysLeft = %d, want 11", op.DaysLeft)
	}
}

func TestCurrentOpenPeriodLastDayHasNoDaysLeft(t *testing.T) {
	now := time.Date(2026, 7, 15, 23, 0, 0, 0, time.UTC)
	op := CurrentOpenPeriod(now)

	if op.DaysLeft != 0 {
		t.Fatalf("daysLeft = %d, want 0", op.DaysLeft)
	}
}

func TestCurrentOpenPeriodFebruaryEndsOnLastDay(t *testing.T) {
	now := time.Date(2028, 2, 20, 9, 0, 0, 0, time.UTC) // 2028 is a leap year
	op := CurrentOpenPeriod(now)

	if op.Period.To != "2028-02-29" {
		t.Fatalf("period.To = %s, want 2028-02-29", op.Period.To)
	}
	if op.DaysLeft != 9 {
		t.Fatalf("daysLeft = %d, want 9", op.DaysLeft)
	}
}
