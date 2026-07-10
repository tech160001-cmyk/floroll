package payroll

import "testing"

func TestResultSummary(t *testing.T) {
	r := Result{
		ShiftPay:     8000,
		RevenueBonus: 1500,
		BonusTotal:   500,
		AdvanceTotal: 2000,
		FineTotal:    0,
		FlowersTotal: 0,
		Total:        8000,
	}

	if got := r.AccruedTotal(); got != 10000 {
		t.Fatalf("AccruedTotal = %v, want 10000", got)
	}
	if got := r.DeductionsTotal(); got != 2000 {
		t.Fatalf("DeductionsTotal = %v, want 2000", got)
	}
	if !r.HasDeductions() {
		t.Fatal("HasDeductions = false, want true")
	}
}

func TestResultSummaryNoDeductions(t *testing.T) {
	r := Result{ShiftPay: 5000, Total: 5000}
	if r.HasDeductions() {
		t.Fatal("HasDeductions = true, want false")
	}
}
