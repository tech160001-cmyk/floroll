package payroll

import (
	"testing"

	"floroll/internal/employee"
	"floroll/internal/operation"
	"floroll/internal/shift"
)

func TestClassifyLiveResult(t *testing.T) {
	emp := employee.Employee{ID: 1, RevenuePercent: 10}
	period := Period{From: "2026-07-01", To: "2026-07-15"}

	t.Run("idle when no shifts or operations", func(t *testing.T) {
		got := ClassifyLiveResult(Calculate(emp, period, nil, nil))
		if got != PeriodStatusIdle {
			t.Fatalf("got %q, want %q", got, PeriodStatusIdle)
		}
	})

	t.Run("ready when total is positive", func(t *testing.T) {
		shifts := []shift.Shift{{Payment: 5000}}
		got := ClassifyLiveResult(Calculate(emp, period, shifts, nil))
		if got != PeriodStatusReady {
			t.Fatalf("got %q, want %q", got, PeriodStatusReady)
		}
	})

	t.Run("attention when total is zero or negative", func(t *testing.T) {
		ops := []operation.Operation{{Type: operation.TypeAdvance, Amount: 5000}}
		shifts := []shift.Shift{{Payment: 3000}}
		got := ClassifyLiveResult(Calculate(emp, period, shifts, ops))
		if got != PeriodStatusAttention {
			t.Fatalf("got %q, want %q", got, PeriodStatusAttention)
		}
	})
}
