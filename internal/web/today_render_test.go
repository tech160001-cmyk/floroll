package web

import (
	"bytes"
	"html/template"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func testTemplates(t *testing.T) *template.Template {
	t.Helper()
	_, thisFile, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(thisFile), "..", "..")
	pattern := filepath.Join(root, "web", "templates", "*.html")

	tmpl, err := template.New("").Funcs(templateFuncMap()).ParseGlob(pattern)
	if err != nil {
		t.Fatalf("parse templates: %v", err)
	}
	return tmpl
}

func TestTodayStateRenders(t *testing.T) {
	tmpl := testTemplates(t)

	cases := map[string]todayPageData{
		"no_employees": {DateLabel: "Понедельник, 13 июля", Kind: todayNoEmployees},
		"ready_to_pay": {
			DateLabel: "Понедельник, 13 июля", PeriodLabel: "1–15 июля",
			Kind: todayReadyToPay, TotalToPay: 38200, ReadyCount: 3, AttentionCount: 1,
		},
		"closing_soon": {
			DateLabel: "Понедельник, 13 июля", PeriodLabel: "1–15 июля",
			ClosingNote: "закрывается через 2 дня", Kind: todayClosingSoon,
		},
		"calm": {
			DateLabel: "Понедельник, 13 июля", PeriodLabel: "1–15 июля",
			ClosingNote: "закрывается через 9 дней", Kind: todayCalm,
		},
	}

	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := tmpl.ExecuteTemplate(&buf, "home-content", data); err != nil {
				t.Fatalf("execute home-content: %v", err)
			}
			if strings.TrimSpace(buf.String()) == "" {
				t.Fatal("rendered empty output")
			}
		})
	}
}

func TestTodayReadyToPayShowsAmountOnce(t *testing.T) {
	tmpl := testTemplates(t)
	data := todayPageData{
		DateLabel: "Понедельник, 13 июля", PeriodLabel: "1–15 июля",
		Kind: todayReadyToPay, TotalToPay: 38200, ReadyCount: 3,
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "home-content", data); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if n := strings.Count(buf.String(), "38 200 ₽"); n != 1 {
		t.Fatalf("expected hero amount once, got %d", n)
	}
}
