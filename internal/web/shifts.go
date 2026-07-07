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
	"floroll/internal/shift"

	"github.com/go-chi/chi/v5"
)

type shiftsPageData struct {
	Shifts    []shift.Shift
	Employees []employee.Employee
	Form      shiftFormData
	ShowForm  bool
}

type shiftFormData struct {
	Error          string
	ID             int64
	IsEdit         bool
	Employees      []employee.Employee
	EmployeeID     string
	EmployeeName   string
	LockedEmployee bool
	Date           string
	Shop           string
	Revenue        string
	ShopRevenue    string
	Comment        string
}

func (h *Handler) shifts(w http.ResponseWriter, r *http.Request) {
	list, err := h.shiftStore.List(r.Context())
	if err != nil {
		http.Error(w, "не удалось загрузить смены", http.StatusInternalServerError)
		return
	}

	employees, err := h.employeeStore.List(r.Context())
	if err != nil {
		h.renderPageError(w, "Смены", "Не удалось загрузить сотрудников. Попробуйте обновить страницу.")
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
			page.Form.LockedEmployee = true
			for _, e := range employees {
				if e.ID == preselectID {
					page.Form.Shop = e.Shop
					page.Form.EmployeeName = e.Name
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
		h.renderModalError(w, "Не удалось загрузить сотрудников. Попробуйте ещё раз.")
		return
	}

	form := shiftFormFromRequest(r, employees)

	if employeeID := r.URL.Query().Get("employee_id"); employeeID != "" {
		form.EmployeeID = employeeID
		form.LockedEmployee = true
		found := false
		for _, e := range employees {
			if strconv.FormatInt(e.ID, 10) == employeeID {
				form.Shop = e.Shop
				form.EmployeeName = e.Name
				found = true
				break
			}
		}
		if !found {
			id, _ := strconv.ParseInt(employeeID, 10, 64)
			if emp, err := h.employeeStore.GetByID(r.Context(), id); err == nil && emp.IsArchived() {
				form.EmployeeName = emp.Name
				form.Shop = emp.Shop
				form.Error = "Архивному сотруднику нельзя добавить смену"
			}
		}
	}

	h.renderPartial(w, "shifts-form", form)
}

func (h *Handler) shiftsCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		h.renderShiftFormError(w, shiftFormData{Error: "Не удалось прочитать данные формы. Попробуйте ещё раз."})
		return
	}

	employees, err := h.employeeStore.List(r.Context())
	if err != nil {
		h.renderModalError(w, "Не удалось загрузить сотрудников. Попробуйте ещё раз.")
		return
	}

	form := shiftFormFromRequest(r, employees)
	form, sh, emp, ok := h.parseShiftForm(r, form)
	if !ok {
		h.renderShiftFormError(w, form)
		return
	}

	if len(employees) == 0 {
		form.Error = "Сначала добавьте сотрудников"
		h.renderShiftFormError(w, form)
		return
	}

	created, err := h.shiftStore.Create(r.Context(), sh)
	if err != nil {
		form.Error = "Не удалось сохранить смену. Попробуйте ещё раз."
		h.renderShiftFormError(w, form)
		return
	}

	created.EmployeeName = emp.Name

	h.triggerDashboardRefresh(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "shifts-row", created); err != nil {
		http.Error(w, "ошибка отображения", http.StatusInternalServerError)
	}
}

func (h *Handler) shiftEditForm(w http.ResponseWriter, r *http.Request) {
	sh, err := h.loadShiftByParam(w, r)
	if err != nil {
		return
	}

	employees, err := h.employeeStore.List(r.Context())
	if err != nil {
		h.renderModalError(w, "Не удалось загрузить сотрудников. Попробуйте ещё раз.")
		return
	}

	h.renderPartial(w, "shifts-form", shiftFormData{
		ID:             sh.ID,
		IsEdit:         true,
		Employees:      employees,
		EmployeeID:     strconv.FormatInt(sh.EmployeeID, 10),
		EmployeeName:   sh.EmployeeName,
		Date:           sh.Date,
		Shop:           sh.Shop,
		Revenue:        strconv.FormatFloat(sh.Revenue, 'f', -1, 64),
		ShopRevenue:    strconv.FormatFloat(sh.ShopRevenue, 'f', -1, 64),
		Comment:        sh.Comment,
		LockedEmployee: len(employees) == 0,
	})
}

func (h *Handler) shiftUpdate(w http.ResponseWriter, r *http.Request) {
	current, err := h.loadShiftByParam(w, r)
	if err != nil {
		return
	}

	employees, err := h.employeeStore.List(r.Context())
	if err != nil {
		h.renderModalError(w, "Не удалось загрузить сотрудников. Попробуйте ещё раз.")
		return
	}

	form := shiftFormFromRequest(r, employees)
	form.ID = current.ID
	form.IsEdit = true
	form, sh, emp, ok := h.parseShiftForm(r, form)
	if !ok {
		h.renderShiftFormError(w, form)
		return
	}
	sh.ID = current.ID

	updated, err := h.shiftStore.Update(r.Context(), sh)
	if err != nil {
		form.Error = "Не удалось сохранить смену. Попробуйте ещё раз."
		h.renderShiftFormError(w, form)
		return
	}
	updated.EmployeeName = emp.Name

	h.triggerDashboardRefresh(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "shifts-row", updated); err != nil {
		http.Error(w, "ошибка отображения", http.StatusInternalServerError)
	}
}

func (h *Handler) shiftDeleteConfirm(w http.ResponseWriter, r *http.Request) {
	sh, err := h.loadShiftByParam(w, r)
	if err != nil {
		return
	}
	h.renderPartial(w, "shift-delete-confirm", sh)
}

func (h *Handler) shiftDelete(w http.ResponseWriter, r *http.Request) {
	sh, err := h.loadShiftByParam(w, r)
	if err != nil {
		return
	}

	covered, err := h.paymentStore.ExistsForEmployeeDate(r.Context(), sh.EmployeeID, sh.Date)
	if err != nil {
		h.renderModalError(w, "Не удалось проверить выплаты. Попробуйте ещё раз.")
		return
	}
	if covered {
		err = h.shiftStore.Cancel(r.Context(), sh.ID)
	} else {
		err = h.shiftStore.Delete(r.Context(), sh.ID)
	}
	if err != nil {
		h.renderModalError(w, "Не удалось удалить смену. Попробуйте ещё раз.")
		return
	}

	h.triggerDashboardRefresh(w)
	w.WriteHeader(http.StatusOK)
}

func shiftFormFromRequest(r *http.Request, employees []employee.Employee) shiftFormData {
	employeeID := r.FormValue("employee_id")
	if employeeID == "" {
		employeeID = r.URL.Query().Get("employee_id")
	}

	date := r.FormValue("shift_date")
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	shop := strings.TrimSpace(r.FormValue("shop"))
	employeeName := strings.TrimSpace(r.FormValue("employee_name"))
	if shop == "" && employeeID != "" {
		for _, e := range employees {
			if strconv.FormatInt(e.ID, 10) == employeeID {
				shop = e.Shop
				employeeName = e.Name
				break
			}
		}
	}

	locked := r.FormValue("locked_employee") == "1" || r.URL.Query().Get("employee_id") != ""

	return shiftFormData{
		Employees:      employees,
		EmployeeID:     employeeID,
		EmployeeName:   employeeName,
		LockedEmployee: locked,
		Date:           date,
		Shop:           shop,
		Revenue:        r.FormValue("revenue"),
		ShopRevenue:    r.FormValue("shop_revenue"),
		Comment:        strings.TrimSpace(r.FormValue("comment")),
	}
}

func (h *Handler) renderShiftFormError(w http.ResponseWriter, form shiftFormData) {
	w.Header().Set("HX-Retarget", "#modal-content")
	w.Header().Set("HX-Reswap", "innerHTML")
	h.renderPartial(w, "shifts-form", form)
}

func (h *Handler) parseShiftForm(r *http.Request, form shiftFormData) (shiftFormData, shift.Shift, employee.Employee, bool) {
	employeeID, err := strconv.ParseInt(form.EmployeeID, 10, 64)
	if err != nil || employeeID <= 0 {
		form.Error = "Выберите сотрудника"
		return form, shift.Shift{}, employee.Employee{}, false
	}

	if form.Date == "" {
		form.Error = "Укажите дату"
		return form, shift.Shift{}, employee.Employee{}, false
	}

	if _, err := time.Parse("2006-01-02", form.Date); err != nil {
		form.Error = "Неверный формат даты"
		return form, shift.Shift{}, employee.Employee{}, false
	}

	if form.Shop == "" {
		form.Error = "Укажите магазин"
		return form, shift.Shift{}, employee.Employee{}, false
	}

	revenue, err := parseAmount(form.Revenue)
	if err != nil || revenue < 0 {
		form.Error = "Укажите корректную личную выручку"
		return form, shift.Shift{}, employee.Employee{}, false
	}

	shopRevenue, err := parseAmount(form.ShopRevenue)
	if err != nil || shopRevenue < 0 {
		form.Error = "Укажите корректную общую выручку магазина"
		return form, shift.Shift{}, employee.Employee{}, false
	}

	emp, err := h.employeeStore.GetByID(r.Context(), employeeID)
	if err != nil || emp.IsArchived() {
		form.Error = "Сотрудник не найден или в архиве"
		return form, shift.Shift{}, employee.Employee{}, false
	}

	return form, shift.Shift{
		EmployeeID:  employeeID,
		Date:        form.Date,
		Shop:        form.Shop,
		Revenue:     revenue,
		ShopRevenue: shopRevenue,
		Comment:     form.Comment,
		Payment:     emp.ShiftRate,
	}, emp, true
}

func (h *Handler) loadShiftByParam(w http.ResponseWriter, r *http.Request) (shift.Shift, error) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return shift.Shift{}, err
	}

	sh, err := h.shiftStore.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, shift.ErrNotFound) {
			http.NotFound(w, r)
			return shift.Shift{}, err
		}
		http.Error(w, "не удалось загрузить смену", http.StatusInternalServerError)
		return shift.Shift{}, err
	}

	return sh, nil
}
