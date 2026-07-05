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

	"github.com/go-chi/chi/v5"
)

type employeesPageData struct {
	Title     string
	Employees []employee.Employee
}

type employeeFormData struct {
	Error          string
	Name           string
	Shop           string
	ShiftRate      string
	RevenuePercent string
}

type employeeProfileData struct {
	Employee   employee.Employee
	Operations []operation.Operation
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
	if err := r.ParseForm(); err != nil {
		http.Error(w, "неверные данные формы", http.StatusBadRequest)
		return
	}

	form := employeeFormData{
		Name:           strings.TrimSpace(r.FormValue("name")),
		Shop:           strings.TrimSpace(r.FormValue("shop")),
		ShiftRate:      strings.TrimSpace(r.FormValue("shift_rate")),
		RevenuePercent: strings.TrimSpace(r.FormValue("revenue_percent")),
	}

	if form.Name == "" || form.Shop == "" {
		form.Error = "Заполните имя и магазин"
		h.renderFormError(w, form)
		return
	}

	shiftRate, err := strconv.ParseFloat(strings.Replace(form.ShiftRate, ",", ".", 1), 64)
	if err != nil || shiftRate < 0 {
		form.Error = "Укажите корректную ставку за смену"
		h.renderFormError(w, form)
		return
	}

	revenuePercent, err := strconv.ParseFloat(strings.Replace(form.RevenuePercent, ",", ".", 1), 64)
	if err != nil || revenuePercent < 0 {
		form.Error = "Укажите корректный процент от выручки"
		h.renderFormError(w, form)
		return
	}

	created, err := h.employeeStore.Create(r.Context(), employee.Employee{
		Name:           form.Name,
		Shop:           form.Shop,
		ShiftRate:      shiftRate,
		RevenuePercent: revenuePercent,
	})
	if err != nil {
		http.Error(w, "не удалось сохранить сотрудника", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "employees-card", created); err != nil {
		http.Error(w, "ошибка отображения", http.StatusInternalServerError)
	}
}

func (h *Handler) employeeDetail(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	emp, err := h.employeeStore.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, employee.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "не удалось загрузить сотрудника", http.StatusInternalServerError)
		return
	}

	operations, err := h.operationStore.ListByEmployee(r.Context(), id)
	if err != nil {
		http.Error(w, "не удалось загрузить операции", http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	if err := h.templates.ExecuteTemplate(&buf, "employee-profile-content", employeeProfileData{
		Employee:   emp,
		Operations: operations,
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
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if _, err := h.employeeStore.GetByID(r.Context(), id); err != nil {
		if errors.Is(err, employee.ErrNotFound) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "не удалось загрузить сотрудника", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/shifts?employee_id="+strconv.FormatInt(id, 10), http.StatusSeeOther)
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
