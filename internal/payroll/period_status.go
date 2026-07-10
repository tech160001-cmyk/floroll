package payroll

// PeriodStatusKind describes an employee's payout state within an open period.
//
// It is derived from live data (shifts, operations) and confirmed payments only.
// There is no separate "calculated but unpaid" state in storage.
type PeriodStatusKind string

const (
	PeriodStatusReady     PeriodStatusKind = "ready"
	PeriodStatusAttention PeriodStatusKind = "attention"
	PeriodStatusIdle      PeriodStatusKind = "idle"
)

// ClassifyLiveResult maps a read-only payroll calculation to a list status.
// Paid employees are handled separately via payment lookup.
func ClassifyLiveResult(r Result) PeriodStatusKind {
	if r.IsEmpty() {
		return PeriodStatusIdle
	}
	if r.Total > 0 {
		return PeriodStatusReady
	}
	return PeriodStatusAttention
}

// DueSummary aggregates the open-period payout queue for the hero block.
type DueSummary struct {
	TotalDue       float64
	ReadyCount     int
	AttentionCount int
	IdleCount      int
	PaidCount      int
}
