package web

import (
	"bytes"
	"database/sql"
	"html/template"
	"net/http"
	"path/filepath"

	"floroll/internal/employee"
	"floroll/internal/operation"
	"floroll/internal/payment"
	"floroll/internal/shift"
)

type pageData struct {
	Title   string
	Content template.HTML
}

type Handler struct {
	templates      *template.Template
	employeeStore  *employee.Store
	operationStore *operation.Store
	shiftStore     *shift.Store
	paymentStore   *payment.Store
}

func NewHandler(db *sql.DB) (*Handler, error) {
	root := projectRoot()
	pattern := filepath.Join(root, "web", "templates", "*.html")

	templates := template.New("").Funcs(template.FuncMap{
		"shiftCountLabel":   shiftCountLabel,
		"formatPaymentDate": formatPaymentDate,
	})
	templates, err := templates.ParseGlob(pattern)
	if err != nil {
		return nil, err
	}

	return &Handler{
		templates:      templates,
		employeeStore:  employee.NewStore(db),
		operationStore: operation.NewStore(db),
		shiftStore:     shift.NewStore(db),
		paymentStore:   payment.NewStore(db),
	}, nil
}

func (h *Handler) home(w http.ResponseWriter, r *http.Request) {
	summary, err := h.dashboardSummaryData(r)
	if err != nil {
		http.Error(w, "не удалось загрузить данные", http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	if err := h.templates.ExecuteTemplate(&buf, "home-content", summary); err != nil {
		http.Error(w, "ошибка отображения страницы", http.StatusInternalServerError)
		return
	}

	h.renderPage(w, pageData{
		Title:   "Главная",
		Content: template.HTML(buf.String()),
	})
}

func (h *Handler) dashboardSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := h.dashboardSummaryData(r)
	if err != nil {
		http.Error(w, "не удалось загрузить данные", http.StatusInternalServerError)
		return
	}
	h.renderPartial(w, "dashboard-summary", summary)
}

func (h *Handler) dashboardSummaryData(r *http.Request) (homePageData, error) {
	count, err := h.employeeStore.Count(r.Context())
	if err != nil {
		return homePageData{}, err
	}
	return homePageData{EmployeeCount: count, TotalPay: 0}, nil
}

func (h *Handler) triggerDashboardRefresh(w http.ResponseWriter) {
	w.Header().Add("HX-Trigger", "dashboard-refresh")
}

func (h *Handler) renderModalError(w http.ResponseWriter, message string) {
	w.Header().Set("HX-Retarget", "#modal-content")
	w.Header().Set("HX-Reswap", "innerHTML")
	h.renderPartial(w, "action-error-card", payrollErrorData{
		Title:   "Не удалось выполнить действие",
		Message: message,
	})
}

func (h *Handler) render(w http.ResponseWriter, contentTemplate string, data pageData) {
	var buf bytes.Buffer
	if err := h.templates.ExecuteTemplate(&buf, contentTemplate, data); err != nil {
		http.Error(w, "ошибка отображения страницы", http.StatusInternalServerError)
		return
	}

	data.Content = template.HTML(buf.String())
	h.renderPage(w, data)
}

func (h *Handler) renderPage(w http.ResponseWriter, data pageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := h.templates.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, "ошибка отображения страницы", http.StatusInternalServerError)
	}
}
