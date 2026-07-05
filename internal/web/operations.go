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
	Error      string
	EmployeeID int64
	Date       string
	Type       string
	Amount     string
	Comment    string
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

	h.renderPartial(w, "operations-form", operationFormData{
		EmployeeID: emp.ID,
		Date:       time.Now().Format("2006-01-02"),
		Type:       string(operation.TypeFine),
	})
}

func (h *Handler) employeeOperationsCreate(w http.ResponseWriter, r *http.Request) {
	emp, err := h.loadEmployee(w, r)
	if err != nil {
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "неверные данные формы", http.StatusBadRequest)
		return
	}

	form := operationFormData{
		EmployeeID: emp.ID,
		Date:       strings.TrimSpace(r.FormValue("op_date")),
		Type:       strings.TrimSpace(r.FormValue("op_type")),
		Amount:     strings.TrimSpace(r.FormValue("amount")),
		Comment:    strings.TrimSpace(r.FormValue("comment")),
	}

	if form.Date == "" {
		form.Error = "Укажите дату"
		h.renderOperationFormError(w, form)
		return
	}

	if _, err := time.Parse("2006-01-02", form.Date); err != nil {
		form.Error = "Неверный формат даты"
		h.renderOperationFormError(w, form)
		return
	}

	opType := operation.Type(form.Type)
	if !isValidOperationType(opType) {
		form.Error = "Выберите тип операции"
		h.renderOperationFormError(w, form)
		return
	}

	amount, err := parseAmount(form.Amount)
	if err != nil || amount <= 0 {
		form.Error = "Укажите корректную сумму"
		h.renderOperationFormError(w, form)
		return
	}

	created, err := h.operationStore.Create(r.Context(), operation.Operation{
		EmployeeID: emp.ID,
		Date:       form.Date,
		Type:       opType,
		Amount:     amount,
		Comment:    form.Comment,
	})
	if err != nil {
		http.Error(w, "не удалось сохранить операцию", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "timeline-row", created); err != nil {
		http.Error(w, "ошибка отображения", http.StatusInternalServerError)
	}
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

func isValidOperationType(t operation.Type) bool {
	for _, valid := range operation.FormTypes() {
		if t == valid {
			return true
		}
	}
	return false
}

func (h *Handler) renderOperationFormError(w http.ResponseWriter, form operationFormData) {
	w.Header().Set("HX-Retarget", "#modal-content")
	w.Header().Set("HX-Reswap", "innerHTML")
	h.renderPartial(w, "operations-form", form)
}
