package payroll

// AccruedTotal is gross earnings before deductions (shifts, percent, bonuses).
func (r Result) AccruedTotal() float64 {
	return r.ShiftPay + r.RevenueBonus + r.BonusTotal
}

// DeductionsTotal is the sum of advances, fines, and flower withholdings.
func (r Result) DeductionsTotal() float64 {
	return r.AdvanceTotal + r.FineTotal + r.FlowersTotal
}

// HasDeductions reports whether any withholdings apply in this period.
func (r Result) HasDeductions() bool {
	return r.DeductionsTotal() > 0
}
