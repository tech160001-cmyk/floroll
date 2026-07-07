package payroll

import (
	"crypto/sha256"
	"fmt"

	"floroll/internal/employee"
	"floroll/internal/operation"
	"floroll/internal/shift"
)

type Result struct {
	Period   Period
	Employee employee.Employee

	ShiftCount   int
	ShiftPay     float64
	RevenueBonus float64
	BonusTotal   float64

	TotalPersonalRevenue float64
	TotalShopRevenue     float64

	AdvanceTotal float64
	FineTotal    float64
	FlowersTotal float64

	Total float64

	Shifts     []shift.Shift
	Operations []operation.Operation
}

func Calculate(emp employee.Employee, period Period, shifts []shift.Shift, ops []operation.Operation) Result {
	result := Result{
		Period:     period,
		Employee:   emp,
		Shifts:     shifts,
		Operations: ops,
		ShiftCount: len(shifts),
	}

	for _, sh := range shifts {
		result.ShiftPay += sh.Payment
		result.RevenueBonus += sh.Revenue * emp.RevenuePercent / 100
		result.TotalPersonalRevenue += sh.Revenue
		result.TotalShopRevenue += sh.ShopRevenue
	}

	for _, op := range ops {
		switch op.Type {
		case operation.TypeAdvance:
			result.AdvanceTotal += op.Amount
		case operation.TypeFine:
			result.FineTotal += op.Amount
		case operation.TypeFlowers, "debt":
			result.FlowersTotal += op.Amount
		case operation.TypeBonus:
			result.BonusTotal += op.Amount
		}
	}

	result.Total = result.ShiftPay +
		result.RevenueBonus +
		result.BonusTotal -
		result.AdvanceTotal -
		result.FineTotal -
		result.FlowersTotal

	return result
}

func (r Result) IsEmpty() bool {
	return r.ShiftCount == 0 && len(r.Operations) == 0
}

func (r Result) Signature() string {
	hash := sha256.New()
	_, _ = fmt.Fprintf(hash, "%d|%s|%s|%.2f|%.2f|%.2f|%.2f|%.2f|%.2f|%.2f|",
		r.Employee.ID,
		r.Period.From,
		r.Period.To,
		r.ShiftPay,
		r.RevenueBonus,
		r.BonusTotal,
		r.AdvanceTotal,
		r.FineTotal,
		r.FlowersTotal,
		r.Total,
	)
	for _, sh := range r.Shifts {
		_, _ = fmt.Fprintf(hash, "s:%d:%s:%.2f:%.2f:%.2f:%s|", sh.ID, sh.Date, sh.Revenue, sh.ShopRevenue, sh.Payment, sh.Comment)
	}
	for _, op := range r.Operations {
		_, _ = fmt.Fprintf(hash, "o:%d:%s:%s:%.2f:%s|", op.ID, op.Date, op.Type, op.Amount, op.Comment)
	}
	return fmt.Sprintf("%x", hash.Sum(nil))
}
