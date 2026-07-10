package web

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"time"

	"floroll/internal/employee"
	"floroll/internal/payment"
	"floroll/internal/payroll"
)

// payoutStatus is the read-only state of an employee within the open period.
// It is always derived on the fly from shifts/operations and confirmed
// payments — FloRoll never persists an intermediate "calculated" state.
type payoutStatus string

const (
	// payoutReady: not paid yet, has activity, live total > 0 — can be paid now.
	payoutReady payoutStatus = "ready"
	// payoutAttention: not paid, has activity, but live total <= 0 — owner must look.
	payoutAttention payoutStatus = "attention"
	// payoutPaid: a confirmed payment already exists for this period.
	payoutPaid payoutStatus = "paid"
	// payoutIdle: no shifts and no operations in the period — nothing to do.
	payoutIdle payoutStatus = "idle"
)

type payoutRow struct {
	Employee   employee.Employee
	Status     payoutStatus
	Amount     float64
	StatusNote string
	ShiftCount int
	Paid       *payment.Payment
}

type payoutsPageData struct {
	Open        payroll.OpenPeriod
	PeriodLabel string
	PeriodMonth string

	Ready     []payoutRow
	Attention []payoutRow
	PaidRows  []payoutRow
	Idle      []payoutRow

	TotalToPay float64

	Employees []employee.Employee
	Form      payrollFormData
}

// HasEmployees reports whether there is anyone to show at all.
func (d payoutsPageData) HasEmployees() bool {
	return len(d.Employees) > 0
}

// HeroSubtitle is a calm note under the hero total — only when it adds context.
func (d payoutsPageData) HeroSubtitle() string {
	if len(d.Attention) == 0 {
		return ""
	}
	return "Ещё " + employeeCountPhrase(len(d.Attention)) + " требуют внимания"
}

// ShowHero reports whether the aggregate payout total should be shown.
func (d payoutsPageData) ShowHero() bool {
	return d.TotalToPay > 0
}

// NeedsAction reports whether anyone requires a payout decision now.
func (d payoutsPageData) NeedsAction() bool {
	return len(d.Ready) > 0 || len(d.Attention) > 0
}

// AllClearTitle is the completion message when nobody needs action.
func (d payoutsPageData) AllClearTitle() string {
	if len(d.PaidRows) > 0 {
		return "Все выплаты за текущий период завершены"
	}
	return "На текущий момент нет сотрудников, ожидающих выплаты"
}

// AllClearEmoji decorates the completion state.
func (d payoutsPageData) AllClearEmoji() string {
	if len(d.PaidRows) > 0 {
		return "✅"
	}
	return "🌿"
}

func (r payoutRow) ShowsAmount() bool {
	return r.Status == payoutReady || r.Status == payoutPaid
}

// buildPayoutsPage assembles the "Выплаты" screen for the current open period.
// It performs only read-only calculations and never writes to the database.
func (h *Handler) buildPayoutsPage(ctx context.Context, r *http.Request) (payoutsPageData, error) {
	open := payroll.CurrentOpenPeriod(time.Now())

	employees, err := h.employeeStore.List(ctx)
	if err != nil {
		return payoutsPageData{}, err
	}

	data := payoutsPageData{
		Open:        open,
		PeriodLabel: open.Label(),
		PeriodMonth: open.Period.From[:7],
		Employees:   employees,
		Form:        payrollFormFromRequest(r, employees),
	}
	data.Form.PeriodPreset = string(open.Preset)
	data.Form.PeriodMonth = open.Period.From[:7]

	for _, emp := range employees {
		row := payoutRow{Employee: emp}

		paid, err := h.paymentStore.FindByEmployeePeriod(ctx, emp.ID, open.Period.From, open.Period.To)
		if err == nil {
			p := paid
			row.Status = payoutPaid
			row.Amount = paid.Amount
			row.StatusNote = "Выплата подтверждена"
			row.Paid = &p
			data.PaidRows = append(data.PaidRows, row)
			continue
		}
		if !errors.Is(err, payment.ErrNotFound) {
			return payoutsPageData{}, err
		}

		result, err := h.computePayrollForPeriod(ctx, emp.ID, open.Period)
		if err != nil {
			return payoutsPageData{}, err
		}
		row.ShiftCount = result.ShiftCount
		row.Amount = result.Total

		switch payroll.ClassifyLiveResult(result) {
		case payroll.PeriodStatusIdle:
			row.Status = payoutIdle
			row.StatusNote = "Нет смен и операций за период"
			data.Idle = append(data.Idle, row)
		case payroll.PeriodStatusReady:
			row.Status = payoutReady
			row.StatusNote = "Можно выплатить"
			data.TotalToPay += result.Total
			data.Ready = append(data.Ready, row)
		case payroll.PeriodStatusAttention:
			row.Status = payoutAttention
			row.StatusNote = payoutAttentionNote(result.Total)
			data.Attention = append(data.Attention, row)
		}
	}

	return data, nil
}

func payoutAttentionNote(total float64) string {
	if total < 0 {
		return "Баланс отрицательный — проверьте удержания"
	}
	return "К выплате 0 ₽ — проверьте начисления"
}

// nextReadyEmployeeAfter returns the next employee who can still be paid in the
// same period, or nil when the queue is empty.
func (h *Handler) nextReadyEmployeeAfter(ctx context.Context, afterEmployeeID int64, period payroll.Period) *payoutNextEmployee {
	employees, err := h.employeeStore.List(ctx)
	if err != nil {
		return nil
	}

	open := payroll.CurrentOpenPeriod(time.Now())
	preset := open.Preset
	month := period.From[:7]
	if period.From != open.Period.From || period.To != open.Period.To {
		preset = payroll.PresetFirstHalf
		if period.From[8:10] >= "16" {
			preset = payroll.PresetSecondHalf
		}
		month = period.From[:7]
	}

	var ready []employee.Employee
	for _, emp := range employees {
		if _, err := h.paymentStore.FindByEmployeePeriod(ctx, emp.ID, period.From, period.To); err == nil {
			continue
		} else if !errors.Is(err, payment.ErrNotFound) {
			return nil
		}

		result, err := h.computePayrollForPeriod(ctx, emp.ID, period)
		if err != nil {
			return nil
		}
		if payroll.ClassifyLiveResult(result) != payroll.PeriodStatusReady {
			continue
		}
		ready = append(ready, emp)
	}

	if len(ready) == 0 {
		return nil
	}

	sort.SliceStable(ready, func(i, j int) bool {
		return ready[i].Name < ready[j].Name
	})

	pastCurrent := false
	for _, emp := range ready {
		if pastCurrent {
			return &payoutNextEmployee{Employee: emp, Preset: preset, Month: month}
		}
		if emp.ID == afterEmployeeID {
			pastCurrent = true
		}
	}

	for _, emp := range ready {
		if emp.ID != afterEmployeeID {
			return &payoutNextEmployee{Employee: emp, Preset: preset, Month: month}
		}
	}

	return nil
}
