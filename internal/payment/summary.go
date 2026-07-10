package payment

// AccruedTotal is gross earnings stored in the payment snapshot.
func (s Snapshot) AccruedTotal() float64 {
	return s.ShiftPay + s.RevenueBonus + s.BonusTotal
}

// DeductionsTotal is the sum of withholdings stored in the payment snapshot.
func (s Snapshot) DeductionsTotal() float64 {
	return s.AdvanceTotal + s.FineTotal + s.FlowersTotal
}

// HasDeductions reports whether the snapshot includes any withholdings.
func (s Snapshot) HasDeductions() bool {
	return s.DeductionsTotal() > 0
}
