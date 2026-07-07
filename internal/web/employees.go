package web

import (
	"bytes"
	"errors"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"floroll/internal/employee"
	"floroll/internal/operation"
	"floroll/internal/payroll"
	"floroll/internal/shift"

	"github.com/go-chi/chi/v5"
)

const employeeProfileRecentLimit = 5

type employeesPageData struct {
	Title     string
	Employees []employee.Employee
}

type employeeFormData struct {
	Error          string
	ID             int64
	IsEdit         bool
	Name           string
	Shop           string
	ShiftRate      string
	RevenuePercent string
}

type employeeProfileData struct {
	Employee         employee.Employee
	RecentShifts     []shift.Shift
	RecentOperations []operation.Operation
	RecentPayments   []paymentHistoryView
}

func (h *Handler) employees(w http.ResponseWriter, r *http.Request) {
	list, err := h.employeeStore.List(r.Context())
	if err != nil {
		http.Error(w, "не удалось загрузить сотрудников", http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	if err := h.templates.ExecuteTemplate(&buf, "employees-content", employeesPageData{
		Title:     "Сотрудники",
		Employees: list,
	}); err != nil {
		http.Error(w, "ошибка отображения страницы", http.StatusInternalServerError)
		return
	}

	h.renderPage(w, pageData{
		Title:   "Сотрудники",
		Content: template.HTML(buf.String()),
	})
}

func (h *Handler) employeesForm(w http.ResponseWriter, r *http.Request) {
	h.renderPartial(w, "employees-form", employeeFormData{})
}

func (h *Handler) employeesCreate(w http.ResponseWriter, r *http.Request) {
	form, emp, ok := h.parseEmployeeForm(r, employeeFormData{})
	if !ok {
		h.renderFormError(w, form)
		return
	}

	created, err := h.employeeStore.Create(r.Context(), emp)
	if err != nil {
		form.Error = "Не удалось сохранить сотрудника. Попробуйте ещё раз."
		h.renderFormError(w, form)
		return
	}

	h.triggerDashboardRefresh(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "employees-card", created); err != nil {
		http.Error(w, "ошибка отображения", http.StatusInternalServerError)
	}
}

func (h *Handler) employeeEditForm(w http.ResponseWriter, r *http.Request) {
	emp, err := h.loadEmployeeByParam(w, r)
	if err != nil {
		return
	}

	h.renderPartial(w, "employees-form", employeeFormData{
		ID:             emp.ID,
		IsEdit:         true,
		Name:           emp.Name,
		Shop:           emp.Shop,
		ShiftRate:      strconv.FormatFloat(emp.ShiftRate, 'f', -1, 64),
		RevenuePercent: strconv.FormatFloat(emp.RevenuePercent, 'f', -1, 64),
	})
}

func (h *Handler) employeeUpdate(w http.ResponseWriter, r *http.Request) {
	current, err := h.loadEmployeeByParam(w, r)
	if err != nil {
		return
	}

	base := employeeFormData{ID: current.ID, IsEdit: true}
	form, emp, ok := h.parseEmployeeForm(r, base)
	if !ok {
		h.renderFormError(w, form)
		return
	}
	emp.ID = current.ID

	updated, err := h.employeeStore.Update(r.Context(), emp)
	if err != nil {
		form.Error = "Не удалось сохранить сотрудника. Попробуйте ещё раз."
		h.renderFormError(w, form)
		return
	}

	h.triggerDashboardRefresh(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "employees-card", updated); err != nil {
		http.Error(w, "ошибка отображения", http.StatusInternalServerError)
	}
}

func (h *Handler) employeeArchiveConfirm(w http.ResponseWriter, r *http.Request) {
	emp, err := h.loadEmployeeByParam(w, r)
	if err != nil {
		return
	}
	h.renderPartial(w, "employee-archive-confirm", emp)
}

func (h *Handler) employeeArchive(w http.ResponseWriter, r *http.Request) {
	emp, err := h.loadEmployeeByParam(w, r)
	if err != nil {
		return
	}

	if err := h.employeeStore.Archive(r.Context(), emp.ID); err != nil {
		h.renderModalError(w, "Не удалось архивировать сотрудника. Попробуйте ещё раз.")
		return
	}

	h.triggerDashboardRefresh(w)
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) employeeDetail(w http.ResponseWriter, r *http.Request) {
	emp, err := h.loadEmployeeByParam(w, r)
	if err != nil {
		return
	}

	recentShifts, err := h.shiftStore.ListRecentByEmployee(r.Context(), emp.ID, employeeProfileRecentLimit)
	if err != nil {
		http.Error(w, "не удалось загрузить смены", http.StatusInternalServerError)
		return
	}

	operations, err := h.operationStore.ListByEmployee(r.Context(), emp.ID)
	if err != nil {
		http.Error(w, "не удалось загрузить операции", http.StatusInternalServerError)
		return
	}
	if len(operations) > employeeProfileRecentLimit {
		operations = operations[:employeeProfileRecentLimit]
	}

	payments, err := h.paymentStore.ListRecentByEmployee(r.Context(), emp.ID, employeeProfileRecentLimit)
	if err != nil {
		http.Error(w, "не удалось загрузить выплаты", http.StatusInternalServerError)
		return
	}

	recentPayments := make([]paymentHistoryView, 0, len(payments))
	for _, p := range payments {
		recentPayments = append(recentPayments, paymentHistoryView{
			Payment:      p,
			EmployeeName: emp.Name,
			PeriodLabel: payroll.Period{
				From: p.PeriodFrom,
				To:   p.PeriodTo,
			}.Label(),
		})
	}

	var buf bytes.Buffer
	if err := h.templates.ExecuteTemplate(&buf, "employee-profile-content", employeeProfileData{
		Employee:         emp,
		RecentShifts:     recentShifts,
		RecentOperations: operations,
		RecentPayments:   recentPayments,
	}); err != nil {
		http.Error(w, "ошибка отображения страницы", http.StatusInternalServerError)
		return
	}

	h.renderPage(w, pageData{
		Title:   emp.Name,
		Content: template.HTML(buf.String()),
	})
}

func (h *Handler) employeeShiftNew(w http.ResponseWriter, r *http.Request) {
	emp, err := h.loadEmployeeByParam(w, r)
	if err != nil {
		return
	}
	if emp.IsArchived() {
		h.renderPageError(w, "Смены", "Архивному сотруднику нельзя добавить новую смену.")
		return
	}

	http.Redirect(w, r, "/shifts?employee_id="+strconv.FormatInt(emp.ID, 10), http.StatusSeeOther)
}

func (h *Handler) renderPartial(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "ошибка отображения", http.StatusInternalServerError)
	}
}

func (h *Handler) renderFormError(w http.ResponseWriter, form employeeFormData) {
	w.Header().Set("HX-Retarget", "#modal-content")
	w.Header().Set("HX-Reswap", "innerHTML")
	h.renderPartial(w, "employees-form", form)
}

func (h *Handler) parseEmployeeForm(r *http.Request, base employeeFormData) (employeeFormData, employee.Employee, bool) {
	if err := r.ParseForm(); err != nil {
		base.Error = "Неверные данные формы"
		return base, employee.Employee{}, false
	}

	form := base
	form.Name = strings.TrimSpace(r.FormValue("name"))
	form.Shop = strings.TrimSpace(r.FormValue("shop"))
	form.ShiftRate = strings.TrimSpace(r.FormValue("shift_rate"))
	form.RevenuePercent = strings.TrimSpace(r.FormValue("revenue_percent"))

	if form.Name == "" || form.Shop == "" {
		form.Error = "Заполните имя и магазин"
		return form, employee.Employee{}, false
	}

	shiftRate, err := strconv.ParseFloat(strings.Replace(form.ShiftRate, ",", ".", 1), 64)
	if err != nil || shiftRate < 0 {
		form.Error = "Укажите корректную ставку за смену"
		return form, employee.Employee{}, false
	}

	revenuePercent, err := strconv.ParseFloat(strings.Replace(form.RevenuePercent, ",", ".", 1), 64)
	if err != nil || revenuePercent < 0 {
		form.Error = "Укажите корректный процент от выручки"
		return form, employee.Employee{}, false
	}

	return form, employee.Employee{
		Name:           form.Name,
		Shop:           form.Shop,
		ShiftRate:      shiftRate,
		RevenuePercent: revenuePercent,
	}, true
}

func (h *Handler) loadEmployeeByParam(w http.ResponseWriter, r *http.Request) (employee.Employee, error) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return employee.Employee{}, err
	}

	emp, err := h.employeeStore.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, employee.ErrNotFound) {
			http.NotFound(w, r)
			return employee.Employee{}, err
		}
		http.Error(w, "не удалось загрузить сотрудника", http.StatusInternalServerError)
		return employee.Employee{}, err
	}

	return emp, nil
}
