package web

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (h *Handler) Router() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	staticDir := filepath.Join(projectRoot(), "web", "static")
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))

	r.Get("/", h.home)
	r.Get("/operations/new", h.operationNew)
	r.Get("/employees", h.employees)
	r.Get("/employees/form", h.employeesForm)
	r.Post("/employees", h.employeesCreate)
	r.Get("/employees/{id}", h.employeeDetail)
	r.Get("/employees/{id}/operations", h.employeeOperations)
	r.Get("/employees/{id}/operations/form", h.employeeOperationsForm)
	r.Post("/employees/{id}/operations", h.employeeOperationsCreate)
	r.Get("/employees/{id}/shifts/new", h.employeeShiftNew)
	r.Get("/shifts", h.shifts)
	r.Get("/shifts/form", h.shiftsForm)
	r.Post("/shifts", h.shiftsCreate)
	r.Get("/payroll", h.payroll)

	return r
}

func projectRoot() string {
	if root := os.Getenv("FLOROLL_ROOT"); root != "" {
		return root
	}
	return "."
}
