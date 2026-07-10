package web

import (
	"bytes"
	"context"
	"errors"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"floroll/internal/employee"
	"floroll/internal/payment"
	"floroll/internal/payroll"
)

type payrollFormData struct {
	EmployeeID   string
	PeriodPreset string
	PeriodMonth  string
	PeriodFrom   string
	PeriodTo     string
	Error        string
}

type payrollPaidView struct {
	Employee employee.Employee
	Period   payroll.Period
	Payment  payment.Payment
	Next     *payoutNextEmployee
}

type payoutNextEmployee struct {
	Employee employee.Employee
	Preset   payroll.Preset
	Month    string
}

type payrollErrorData struct {
	Title   string
	Message string
}

func (h *Handler) payroll(w http.ResponseWriter, r *http.Request) {
	data, err := h.buildPayoutsPage(r.Context(), r)
	if err != nil {
		h.renderPageError(w, "Выплаты", "Не удалось загрузить выплаты. Попробуйте обновить страницу.")
		return
	}

	var buf bytes.Buffer
	if err := h.templates.ExecuteTemplate(&buf, "payroll-content", data); err != nil {
		h.renderPageError(w, "Выплаты", "Не удалось отобразить страницу выплат.")
		return
	}

	h.renderPage(w, pageData{
		Title:   "Выплаты",
		Content: template.HTML(buf.String()),
	})
}

func (h *Handler) payrollCalculate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	employees, err := h.employeeStore.List(ctx)
	if err != nil {
		h.renderPayrollError(w, "Не удалось загрузить сотрудников. Попробуйте ещё раз.")
		return
	}

	form := payrollFormFromRequest(r, employees)
	form = validatePayrollForm(ctx, h, form, employees)
	if form.Error != "" {
		h.renderPartial(w, "payroll-result-error", form)
		return
	}

	result, err := h.computePayrollFromForm(ctx, form)
	if err != nil {
		h.renderPayrollError(w, "Не удалось выполнить расчёт. Проверьте данные и попробуйте ещё раз.")
		return
	}

	if result.IsEmpty() {
		h.renderPartial(w, "payroll-result-no-data", result)
		return
	}

	h.renderPayrollResult(w, ctx, result)
}

func (h *Handler) payrollConfirm(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.renderPayrollError(w, "Не удалось прочитать данные формы. Выполните расчёт ещё раз.")
		return
	}

	ctx := r.Context()

	employeeID, err := strconv.ParseInt(r.FormValue("employee_id"), 10, 64)
	if err != nil || employeeID <= 0 {
		h.renderPartial(w, "payroll-result-error", payrollFormData{Error: "Выберите сотрудника"})
		return
	}

	periodFrom := r.FormValue("period_from")
	periodTo := r.FormValue("period_to")
	period, err := payroll.PeriodFromDates(periodFrom, periodTo)
	if err != nil {
		h.renderPartial(w, "payroll-result-error", payrollFormData{Error: err.Error()})
		return
	}

	result, err := h.computePayrollForPeriod(ctx, employeeID, period)
	if err != nil {
		if errors.Is(err, employee.ErrNotFound) {
			h.renderPartial(w, "payroll-result-error", payrollFormData{Error: "Сотрудник не найден"})
			return
		}
		h.renderPayrollError(w, "Не удалось выполнить расчёт. Выполните расчёт ещё раз.")
		return
	}

	if result.IsEmpty() {
		h.renderPartial(w, "payroll-result-no-data", result)
		return
	}

	if r.FormValue("calculation_signature") != result.Signature() {
		h.renderPayrollError(w, "Данные изменились. Выполните расчёт ещё раз.")
		return
	}

	if result.Total <= 0 {
		h.renderPayrollError(w, "Итоговая сумма меньше или равна 0 ₽. Такую выплату нельзя подтвердить.")
		return
	}

	existing, err := h.paymentStore.FindByEmployeePeriod(ctx, result.Employee.ID, period.From, period.To)
	if err == nil {
		h.renderPartial(w, "payroll-result-already-paid", payrollPaidView{
			Employee: result.Employee,
			Period:   result.Period,
			Payment:  existing,
			Next:     h.nextReadyEmployeeAfter(ctx, result.Employee.ID, period),
		})
		return
	}
	if !errors.Is(err, payment.ErrNotFound) {
		h.renderPayrollError(w, "Не удалось проверить, была ли выплата уже подтверждена. Попробуйте ещё раз.")
		return
	}

	comment := strings.TrimSpace(r.FormValue("payment_comment"))
	created, err := h.paymentStore.Create(ctx, payment.Payment{
		EmployeeID: result.Employee.ID,
		PeriodFrom: period.From,
		PeriodTo:   period.To,
		Amount:     result.Total,
		Comment:    comment,
		Snapshot:   payment.SnapshotFromResult(result),
	})
	if err != nil {
		h.renderPayrollError(w, "Не удалось сохранить выплату. Попробуйте ещё раз.")
		return
	}

	h.renderPartial(w, "payroll-result-confirmed", payrollPaidView{
		Employee: result.Employee,
		Period:   result.Period,
		Payment:  created,
		Next:     h.nextReadyEmployeeAfter(ctx, result.Employee.ID, result.Period),
	})
}

func (h *Handler) renderPayrollResult(w http.ResponseWriter, ctx context.Context, result payroll.Result) {
	existing, err := h.paymentStore.FindByEmployeePeriod(ctx, result.Employee.ID, result.Period.From, result.Period.To)
	if err == nil {
		h.renderPartial(w, "payroll-result-already-paid", payrollPaidView{
			Employee: result.Employee,
			Period:   result.Period,
			Payment:  existing,
			Next:     h.nextReadyEmployeeAfter(ctx, result.Employee.ID, result.Period),
		})
		return
	}
	if !errors.Is(err, payment.ErrNotFound) {
		h.renderPayrollError(w, "Не удалось проверить, была ли выплата уже подтверждена. Попробуйте ещё раз.")
		return
	}

	h.renderPartial(w, "payroll-result", result)
}

func validatePayrollForm(ctx context.Context, h *Handler, form payrollFormData, employees []employee.Employee) payrollFormData {

	employeeID, err := strconv.ParseInt(form.EmployeeID, 10, 64)
	if err != nil || employeeID <= 0 {
		form.Error = "Выберите сотрудника"
		return form
	}

	preset := payroll.Preset(form.PeriodPreset)
	if !preset.IsValid() {
		form.Error = "Выберите тип периода"
		return form
	}

	if _, err := payroll.ResolvePeriod(preset, form.PeriodMonth, form.PeriodFrom, form.PeriodTo); err != nil {
		form.Error = err.Error()
		return form
	}

	if _, err := h.employeeStore.GetByID(ctx, employeeID); err != nil {
		form.Error = "Сотрудник не найден"
		return form
	}

	return form
}

func (h *Handler) computePayrollFromForm(ctx context.Context, form payrollFormData) (payroll.Result, error) {
	employeeID, _ := strconv.ParseInt(form.EmployeeID, 10, 64)
	preset := payroll.Preset(form.PeriodPreset)
	period, err := payroll.ResolvePeriod(preset, form.PeriodMonth, form.PeriodFrom, form.PeriodTo)
	if err != nil {
		return payroll.Result{}, err
	}
	return h.computePayrollForPeriod(ctx, employeeID, period)
}

func (h *Handler) computePayrollForPeriod(ctx context.Context, employeeID int64, period payroll.Period) (payroll.Result, error) {
	emp, err := h.employeeStore.GetByID(ctx, employeeID)
	if err != nil {
		return payroll.Result{}, err
	}

	shifts, err := h.shiftStore.ListByEmployeePeriod(ctx, employeeID, period.From, period.To)
	if err != nil {
		return payroll.Result{}, err
	}

	ops, err := h.operationStore.ListByEmployeePeriod(ctx, employeeID, period.From, period.To)
	if err != nil {
		return payroll.Result{}, err
	}

	return payroll.Calculate(emp, period, shifts, ops), nil
}

func (h *Handler) renderPayrollError(w http.ResponseWriter, message string) {
	h.renderPartial(w, "payroll-result-error", payrollFormData{Error: message})
}

func (h *Handler) renderPageError(w http.ResponseWriter, title, message string) {
	var buf bytes.Buffer
	if err := h.templates.ExecuteTemplate(&buf, "app-error-card", payrollErrorData{
		Title:   title,
		Message: message,
	}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	h.renderPage(w, pageData{
		Title:   title,
		Content: template.HTML(buf.String()),
	})
}

func payrollFormFromRequest(r *http.Request, employees []employee.Employee) payrollFormData {
	now := time.Now()

	preset := r.FormValue("period_preset")
	if preset == "" {
		preset = string(payroll.DefaultPreset(now))
	}

	month := r.FormValue("period_month")
	if month == "" {
		month = now.Format("2006-01")
	}

	from := r.FormValue("period_from")
	if from == "" {
		from = now.Format("2006-01-02")
	}

	to := r.FormValue("period_to")
	if to == "" {
		to = now.Format("2006-01-02")
	}

	employeeID := r.FormValue("employee_id")
	if employeeID == "" && len(employees) == 1 {
		employeeID = strconv.FormatInt(employees[0].ID, 10)
	}

	return payrollFormData{
		EmployeeID:   employeeID,
		PeriodPreset: preset,
		PeriodMonth:  month,
		PeriodFrom:   from,
		PeriodTo:     to,
	}
}
