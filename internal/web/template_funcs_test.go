package web

import (
	"testing"

	"floroll/internal/payroll"
)

func TestFormatMoney(t *testing.T) {
	tests := []struct {
		in   float64
		want string
	}{
		{0, "0 ₽"},
		{18200, "18 200 ₽"},
		{1234567, "1 234 567 ₽"},
		{-500, "− 500 ₽"},
	}
	for _, tc := range tests {
		if got := formatMoney(tc.in); got != tc.want {
			t.Fatalf("formatMoney(%v) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestPayrollDueSubtitle(t *testing.T) {
	got := payrollDueSubtitle(payroll.DueSummary{
		TotalDue:       18200,
		ReadyCount:     2,
		AttentionCount: 2,
	})
	want := "Ещё 2 сотрудника требуют внимания"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
