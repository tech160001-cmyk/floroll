package web

import (
	"bytes"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"floroll/internal/employee"
	"floroll/internal/operation"
)

type shiftsPageData struct {
	Shifts    []operation.Operation
	Employees []employee.Employee
	Form      shiftFormData
	ShowForm  bool
}

type shiftFormData struct {
	Error         string
	Employees     []employee.Employee
	EmployeeID    string
	Date          string
	Shop          string
	Revenue       string
	ShiftType     string
	CustomPayment string
	ShowPayment   bool
}

func (h *Handler) shifts(w http.ResponseWriter, r *http.Request) {
	list, err := h.operationStore.ListShifts(r.Context())
	if err != nil {
		http.Error(w, "не удалось загрузить смены", http.StatusInternalServerError)
		return
	}

	employees, err := h.employeeStore.List(r.Context())
	if err != nil {
		http.Error(w, "не удалось загрузить сотрудников", http.StatusInternalServerError)
		return
	}

	var preselectID int64
	if idStr := r.URL.Query().Get("employee_id"); idStr != "" {
		preselectID, _ = strconv.ParseInt(idStr, 10, 64)
	}

	showForm := r.URL.Query().Get("form") == "1" || preselectID > 0

	page := shiftsPageData{
		Shifts:    list,
		Employees: employees,
		ShowForm:  showForm,
	}

	if showForm {
		page.Form = shiftFormFromRequest(r, employees)
		if preselectID > 0 && page.Form.EmployeeID == "" {
			page.Form.EmployeeID = strconv.FormatInt(preselectID, 10)
			for _, e := range employees {
				if e.ID == preselectID {
					page.Form.Shop = e.Shop
					break
				}
			}
		}
	}

	var buf bytes.Buffer
	if err := h.templates.ExecuteTemplate(&buf, "shifts-content", page); err != nil {
		http.Error(w, "ошибка отображения страницы", http.StatusInternalServerError)
		return
	}

	h.renderPage(w, pageData{
		Title:   "Смены",
		Content: template.HTML(buf.String()),
	})
}

func (h *Handler) shiftsForm(w http.ResponseWriter, r *http.Request) {
	employees, err := h.employeeStore.List(r.Context())
	if err != nil {
		http.Error(w, "не удалось загрузить сотрудников", http.StatusInternalServerError)
		return
	}

	form := shiftFormFromRequest(r, employees)
	h.renderPartial(w, "shifts-form", form)
}

func (h *Handler) shiftsCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "неверные данные формы", http.StatusBadRequest)
		return
	}

	employees, err := h.employeeStore.List(r.Context())
	if err != nil {
		http.Error(w, "не удалось загрузить сотрудников", http.StatusInternalServerError)
		return
	}

	form := shiftFormFromRequest(r, employees)

	if len(employees) == 0 {
		form.Error = "Сначала добавьте сотрудников"
		h.renderShiftFormError(w, form)
		return
	}

	employeeID, err := strconv.ParseInt(form.EmployeeID, 10, 64)
	if err != nil || employeeID <= 0 {
		form.Error = "Выберите сотрудника"
		h.renderShiftFormError(w, form)
		return
	}

	if form.Date == "" {
		form.Error = "Укажите дату"
		h.renderShiftFormError(w, form)
		return
	}

	if _, err := time.Parse("2006-01-02", form.Date); err != nil {
		form.Error = "Неверный формат даты"
		h.renderShiftFormError(w, form)
		return
	}

	if form.Shop == "" {
		form.Error = "Укажите магазин"
		h.renderShiftFormError(w, form)
		return
	}

	revenue, err := parseAmount(form.Revenue)
	if err != nil || revenue < 0 {
		form.Error = "Укажите корректную выручку"
		h.renderShiftFormError(w, form)
		return
	}

	emp, err := h.employeeStore.GetByID(r.Context(), employeeID)
	if err != nil {
		form.Error = "Сотрудник не найден"
		h.renderShiftFormError(w, form)
		return
	}

	shiftKind := operation.ShiftKind(form.ShiftType)
	if shiftKind != operation.ShiftRegular && shiftKind != operation.ShiftSubstitute {
		shiftKind = operation.ShiftRegular
		form.ShiftType = string(operation.ShiftRegular)
	}

	var payment float64
	if shiftKind == operation.ShiftSubstitute {
		payment, err = parseAmount(form.CustomPayment)
		if err != nil || payment < 0 {
			form.ShowPayment = true
			form.Error = "Укажите оплату за подработку"
			h.renderShiftFormError(w, form)
			return
		}
	} else {
		payment = emp.ShiftRate
	}

	created, err := h.operationStore.Create(r.Context(), operation.Operation{
		EmployeeID: employeeID,
		Date:       form.Date,
		Type:       operation.TypeShift,
		Amount:     payment,
		Shop:       form.Shop,
		Revenue:    revenue,
		ShiftKind:  shiftKind,
	})
	if err != nil {
		http.Error(w, "не удалось сохранить смену", http.StatusInternalServerError)
		return
	}

	created.EmployeeName = emp.Name

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "shifts-row", created); err != nil {
		http.Error(w, "ошибка отображения", http.StatusInternalServerError)
	}
}

func shiftFormFromRequest(r *http.Request, employees []employee.Employee) shiftFormData {
	shiftType := r.FormValue("shift_type")
	if shiftType == "" {
		if r.URL.Query().Get("shift_type") != "" {
			shiftType = r.URL.Query().Get("shift_type")
		} else {
			shiftType = string(operation.ShiftRegular)
		}
	}

	employeeID := r.FormValue("employee_id")
	if employeeID == "" {
		employeeID = r.URL.Query().Get("employee_id")
	}

	date := r.FormValue("shift_date")
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	shop := strings.TrimSpace(r.FormValue("shop"))
	if shop == "" && employeeID != "" {
		for _, e := range employees {
			if strconv.FormatInt(e.ID, 10) == employeeID {
				shop = e.Shop
				break
			}
		}
	}

	return shiftFormData{
		Employees:     employees,
		EmployeeID:    employeeID,
		Date:          date,
		Shop:          shop,
		Revenue:       r.FormValue("revenue"),
		ShiftType:     shiftType,
		CustomPayment: r.FormValue("custom_payment"),
		ShowPayment:   shiftType == string(operation.ShiftSubstitute),
	}
}

func parseAmount(value string) (float64, error) {
	return strconv.ParseFloat(strings.Replace(strings.TrimSpace(value), ",", ".", 1), 64)
}

func (h *Handler) renderShiftFormError(w http.ResponseWriter, form shiftFormData) {
	w.Header().Set("HX-Retarget", "#modal-content")
	w.Header().Set("HX-Reswap", "innerHTML")
	h.renderPartial(w, "shifts-form", form)
}
