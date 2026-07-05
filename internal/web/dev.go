package web

import (
	"bytes"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type devIconsPageData struct {
	Icons []string
	Sizes []int
}

func (h *Handler) devIcons(w http.ResponseWriter, r *http.Request) {
	iconsDir := filepath.Join(projectRoot(), "web", "static", "icons")
	entries, err := os.ReadDir(iconsDir)
	if err != nil {
		http.Error(w, "не удалось загрузить иконки", http.StatusInternalServerError)
		return
	}

	var icons []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(strings.ToLower(name), ".svg") {
			icons = append(icons, name)
		}
	}
	sort.Strings(icons)

	var buf bytes.Buffer
	if err := h.templates.ExecuteTemplate(&buf, "dev-icons-content", devIconsPageData{
		Icons: icons,
		Sizes: []int{24, 32, 48, 64},
	}); err != nil {
		http.Error(w, "ошибка отображения страницы", http.StatusInternalServerError)
		return
	}

	h.renderPage(w, pageData{
		Title:   "Icons",
		Content: template.HTML(buf.String()),
	})
}
