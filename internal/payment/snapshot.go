package payment

import "floroll/internal/payroll"

type Snapshot struct {
	ShiftPay     float64
	RevenueBonus float64
	BonusTotal   float64
	AdvanceTotal float64
	FineTotal    float64
	FlowersTotal float64
}

func SnapshotFromResult(r payroll.Result) Snapshot {
	return Snapshot{
		ShiftPay:     r.ShiftPay,
		RevenueBonus: r.RevenueBonus,
		BonusTotal:   r.BonusTotal,
		AdvanceTotal: r.AdvanceTotal,
		FineTotal:    r.FineTotal,
		FlowersTotal: r.FlowersTotal,
	}
}
