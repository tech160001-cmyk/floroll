package web

import (
	"bytes"
	"database/sql"
	"html/template"
	"net/http"
	"path/filepath"

	"floroll/internal/employee"
	"floroll/internal/operation"
)

type pageData struct {
	Title   string
	Content template.HTML
}

type Handler struct {
	templates      *template.Template
	employeeStore  *employee.Store
	operationStore *operation.Store
}

func NewHandler(db *sql.DB) (*Handler, error) {
	root := projectRoot()
	pattern := filepath.Join(root, "web", "templates", "*.html")

	templates, err := template.ParseGlob(pattern)
	if err != nil {
		return nil, err
	}

	return &Handler{
		templates:      templates,
		employeeStore:  employee.NewStore(db),
		operationStore: operation.NewStore(db),
	}, nil
}

func (h *Handler) home(w http.ResponseWriter, r *http.Request) {
	count, err := h.employeeStore.Count(r.Context())
	if err != nil {
		http.Error(w, "не удалось загрузить данные", http.StatusInternalServerError)
		return
	}

	var buf bytes.Buffer
	if err := h.templates.ExecuteTemplate(&buf, "home-content", homePageData{
		EmployeeCount: count,
		TotalPay:      0,
	}); err != nil {
		http.Error(w, "ошибка отображения страницы", http.StatusInternalServerError)
		return
	}

	h.renderPage(w, pageData{
		Title:   "Главная",
		Content: template.HTML(buf.String()),
	})
}

func (h *Handler) operationNew(w http.ResponseWriter, r *http.Request) {
	h.render(w, "stub-content", pageData{Title: "Добавить операцию"})
}

func (h *Handler) payroll(w http.ResponseWriter, r *http.Request) {
	h.render(w, "stub-content", pageData{Title: "Зарплата"})
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
