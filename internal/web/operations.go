package web

import (
	"bytes"
	"errors"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"floroll/internal/employee"
	"floroll/internal/operation"

	"github.com/go-chi/chi/v5"
)

type employeeOperationsPageData struct {
	Employee   employee.Employee
	Operations []operation.Operation
}

type operationFormData struct {
	Error          string
	ID             int64
	IsEdit         bool
	EmployeeID     int64
	EmployeeName   string
	LockedEmployee bool
	Employees      []employee.Employee
	Types          []operation.Type
	Date           string
	Type           string
	Amount         string
	Comment        string
}

func (h *Handler) employeeOperations(w http.ResponseWriter, r *http.Request) {
	emp, err := h.loadEmployee(w, r)
	if err != nil {
		return
	}

	ops, err := h.operationStore.ListByEmployee(r.Context(), emp.ID)
	if err != nil {
		http.Error(w, "не удалось загрузить операции", http.StatusInternalServerError)
		return
	}

	page := employeeOperationsPageData{
		Employee:   emp,
		Operations: ops,
	}

	var buf bytes.Buffer
	if err := h.templates.ExecuteTemplate(&buf, "employee-operations-content", page); err != nil {
		http.Error(w, "ошибка отображения страницы", http.StatusInternalServerError)
		return
	}

	h.renderPage(w, pageData{
		Title:   "Операции — " + emp.Name,
		Content: template.HTML(buf.String()),
	})
}

func (h *Handler) employeeOperationsForm(w http.ResponseWriter, r *http.Request) {
	emp, err := h.loadEmployee(w, r)
	if err != nil {
		return
	}
	if emp.IsArchived() {
		h.renderPartial(w, "operations-form", operationFormData{
			Error:          "Архивному сотруднику нельзя добавить операцию",
			EmployeeID:     emp.ID,
			EmployeeName:   emp.Name,
			LockedEmployee: true,
			Types:          operation.FormTypes(),
			Date:           time.Now().Format("2006-01-02"),
			Type:           string(operation.TypeAdvance),
		})
		return
	}

	h.renderPartial(w, "operations-form", h.newOperationForm(emp, true))
}

func (h *Handler) operationForm(w http.ResponseWriter, r *http.Request) {
	employees, err := h.employeeStore.List(r.Context())
	if err != nil {
		h.renderModalError(w, "Не удалось загрузить сотрудников. Попробуйте ещё раз.")
		return
	}

	form := operationFormData{
		LockedEmployee: false,
		Employees:      employees,
		Types:          operation.FormTypes(),
		Date:           time.Now().Format("2006-01-02"),
		Type:           string(operation.TypeAdvance),
	}

	if len(employees) == 0 {
		form.Error = "Сначала добавьте сотрудников"
	}

	h.renderPartial(w, "operations-form", form)
}

func (h *Handler) operationNew(w http.ResponseWriter, r *http.Request) {
	var buf bytes.Buffer
	if err := h.templates.ExecuteTemplate(&buf, "operations-new-content", nil); err != nil {
		http.Error(w, "ошибка отображения страницы", http.StatusInternalServerError)
		return
	}

	h.renderPage(w, pageData{
		Title:   "Добавить операцию",
		Content: template.HTML(buf.String()),
	})
}

func (h *Handler) employeeOperationsCreate(w http.ResponseWriter, r *http.Request) {
	emp, err := h.loadEmployee(w, r)
	if err != nil {
		return
	}
	if emp.IsArchived() {
		h.renderOperationFormError(w, operationFormData{
			Error:          "Архивному сотруднику нельзя добавить операцию",
			EmployeeID:     emp.ID,
			EmployeeName:   emp.Name,
			LockedEmployee: true,
			Types:          operation.FormTypes(),
			Date:           time.Now().Format("2006-01-02"),
			Type:           string(operation.TypeAdvance),
		})
		return
	}

	form, ok := h.parseOperationForm(r, h.newOperationForm(emp, true))
	if !ok {
		h.renderOperationFormError(w, form)
		return
	}

	created, err := h.saveOperation(r, form)
	if err != nil {
		if errors.Is(err, errArchivedEmployee) {
			form.Error = "Архивному сотруднику нельзя добавить операцию"
			h.renderOperationFormError(w, form)
			return
		}
		form.Error = "Не удалось сохранить операцию. Попробуйте ещё раз."
		h.renderOperationFormError(w, form)
		return
	}

	h.triggerDashboardRefresh(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "timeline-row", created); err != nil {
		http.Error(w, "ошибка отображения", http.StatusInternalServerError)
	}
}

func (h *Handler) operationCreate(w http.ResponseWriter, r *http.Request) {
	employees, err := h.employeeStore.List(r.Context())
	if err != nil {
		h.renderModalError(w, "Не удалось загрузить сотрудников. Попробуйте ещё раз.")
		return
	}

	base := operationFormData{
		LockedEmployee: false,
		Employees:      employees,
		Types:          operation.FormTypes(),
		Date:           time.Now().Format("2006-01-02"),
		Type:           string(operation.TypeAdvance),
	}

	form, ok := h.parseOperationForm(r, base)
	if !ok {
		h.renderOperationFormError(w, form)
		return
	}

	created, err := h.saveOperation(r, form)
	if err != nil {
		if errors.Is(err, errArchivedEmployee) {
			form.Error = "Архивному сотруднику нельзя добавить операцию"
			h.renderOperationFormError(w, form)
			return
		}
		form.Error = "Не удалось сохранить операцию. Попробуйте ещё раз."
		h.renderOperationFormError(w, form)
		return
	}

	h.triggerDashboardRefresh(w)
	w.Header().Set("HX-Redirect", "/employees/"+strconv.FormatInt(created.EmployeeID, 10)+"/operations")
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) operationEditForm(w http.ResponseWriter, r *http.Request) {
	op, err := h.loadOperationByParam(w, r)
	if err != nil {
		return
	}

	emp, err := h.employeeStore.GetByID(r.Context(), op.EmployeeID)
	if err != nil {
		http.Error(w, "не удалось загрузить сотрудника", http.StatusInternalServerError)
		return
	}

	h.renderPartial(w, "operations-form", operationFormData{
		ID:             op.ID,
		IsEdit:         true,
		EmployeeID:     op.EmployeeID,
		EmployeeName:   emp.Name,
		LockedEmployee: true,
		Types:          operation.FormTypes(),
		Date:           op.Date,
		Type:           string(op.Type),
		Amount:         strconv.FormatFloat(op.Amount, 'f', -1, 64),
		Comment:        op.Comment,
	})
}

func (h *Handler) operationUpdate(w http.ResponseWriter, r *http.Request) {
	current, err := h.loadOperationByParam(w, r)
	if err != nil {
		return
	}

	emp, err := h.employeeStore.GetByID(r.Context(), current.EmployeeID)
	if err != nil {
		http.Error(w, "не удалось загрузить сотрудника", http.StatusInternalServerError)
		return
	}

	base := operationFormData{
		ID:             current.ID,
		IsEdit:         true,
		EmployeeID:     current.EmployeeID,
		EmployeeName:   emp.Name,
		LockedEmployee: true,
		Types:          operation.FormTypes(),
	}
	form, ok := h.parseOperationForm(r, base)
	if !ok {
		h.renderOperationFormError(w, form)
		return
	}

	updated, err := h.operationStore.Update(r.Context(), operation.Operation{
		ID:         current.ID,
		EmployeeID: current.EmployeeID,
		Date:       form.Date,
		Type:       operation.Type(form.Type),
		Amount:     mustParseAmount(form.Amount),
		Comment:    form.Comment,
	})
	if err != nil {
		form.Error = "Не удалось сохранить операцию. Попробуйте ещё раз."
		h.renderOperationFormError(w, form)
		return
	}

	h.triggerDashboardRefresh(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "timeline-row", updated); err != nil {
		http.Error(w, "ошибка отображения", http.StatusInternalServerError)
	}
}

func (h *Handler) operationDeleteConfirm(w http.ResponseWriter, r *http.Request) {
	op, err := h.loadOperationByParam(w, r)
	if err != nil {
		return
	}
	h.renderPartial(w, "operation-delete-confirm", op)
}

func (h *Handler) operationDelete(w http.ResponseWriter, r *http.Request) {
	op, err := h.loadOperationByParam(w, r)
	if err != nil {
		return
	}

	covered, err := h.paymentStore.ExistsForEmployeeDate(r.Context(), op.EmployeeID, op.Date)
	if err != nil {
		h.renderModalError(w, "Не удалось проверить выплаты. Попробуйте ещё раз.")
		return
	}
	if covered {
		err = h.operationStore.Cancel(r.Context(), op.ID)
	} else {
		err = h.operationStore.Delete(r.Context(), op.ID)
	}
	if err != nil {
		h.renderModalError(w, "Не удалось удалить операцию. Попробуйте ещё раз.")
		return
	}

	h.triggerDashboardRefresh(w)
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) newOperationForm(emp employee.Employee, locked bool) operationFormData {
	return operationFormData{
		EmployeeID:     emp.ID,
		EmployeeName:   emp.Name,
		LockedEmployee: locked,
		Types:          operation.FormTypes(),
		Date:           time.Now().Format("2006-01-02"),
		Type:           string(operation.TypeAdvance),
	}
}

func (h *Handler) parseOperationForm(r *http.Request, base operationFormData) (operationFormData, bool) {
	if err := r.ParseForm(); err != nil {
		base.Error = "Неверные данные формы"
		return base, false
	}

	form := base
	form.Date = strings.TrimSpace(r.FormValue("op_date"))
	form.Type = strings.TrimSpace(r.FormValue("operation_type"))
	form.Amount = strings.TrimSpace(r.FormValue("amount"))
	form.Comment = strings.TrimSpace(r.FormValue("comment"))

	if !form.LockedEmployee {
		employeeID, err := strconv.ParseInt(strings.TrimSpace(r.FormValue("employee_id")), 10, 64)
		if err != nil || employeeID <= 0 {
			form.Error = "Выберите сотрудника"
			return form, false
		}
		form.EmployeeID = employeeID
		for _, emp := range form.Employees {
			if emp.ID == employeeID {
				form.EmployeeName = emp.Name
				break
			}
		}
	}

	if len(form.Employees) == 0 && !form.LockedEmployee {
		form.Error = "Сначала добавьте сотрудников"
		return form, false
	}

	if form.Date == "" {
		form.Error = "Укажите дату"
		return form, false
	}

	if _, err := time.Parse("2006-01-02", form.Date); err != nil {
		form.Error = "Неверный формат даты"
		return form, false
	}

	opType := operation.Type(form.Type)
	if !opType.IsFormType() {
		form.Error = "Выберите тип операции"
		return form, false
	}

	amount, err := parseAmount(form.Amount)
	if err != nil || amount <= 0 {
		form.Error = "Укажите корректную сумму"
		return form, false
	}

	form.Type = string(opType)
	return form, true
}

func (h *Handler) saveOperation(r *http.Request, form operationFormData) (operation.Operation, error) {
	emp, err := h.employeeStore.GetByID(r.Context(), form.EmployeeID)
	if err != nil {
		return operation.Operation{}, err
	}
	if emp.IsArchived() {
		return operation.Operation{}, errArchivedEmployee
	}

	amount, _ := parseAmount(form.Amount)
	return h.operationStore.Create(r.Context(), operation.Operation{
		EmployeeID: form.EmployeeID,
		Date:       form.Date,
		Type:       operation.Type(form.Type),
		Amount:     amount,
		Comment:    form.Comment,
	})
}

func (h *Handler) loadOperationByParam(w http.ResponseWriter, r *http.Request) (operation.Operation, error) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return operation.Operation{}, err
	}

	op, err := h.operationStore.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, operation.ErrNotFound) {
			http.NotFound(w, r)
			return operation.Operation{}, err
		}
		http.Error(w, "не удалось загрузить операцию", http.StatusInternalServerError)
		return operation.Operation{}, err
	}

	return op, nil
}

func (h *Handler) loadEmployee(w http.ResponseWriter, r *http.Request) (employee.Employee, error) {
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

func (h *Handler) renderOperationFormError(w http.ResponseWriter, form operationFormData) {
	w.Header().Set("HX-Retarget", "#modal-content")
	w.Header().Set("HX-Reswap", "innerHTML")
	h.renderPartial(w, "operations-form", form)
}

func parseAmount(value string) (float64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	return strconv.ParseFloat(strings.Replace(value, ",", ".", 1), 64)
}

func mustParseAmount(value string) float64 {
	amount, _ := parseAmount(value)
	return amount
}

var errArchivedEmployee = errors.New("employee is archived")
