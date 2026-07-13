package web

import (
	"fmt"
	"html/template"
	"strconv"
	"strings"
	"time"

	"floroll/internal/payroll"
)

func templateFuncMap() template.FuncMap {
	return template.FuncMap{
		"shiftCountLabel":     shiftCountLabel,
		"formatPaymentDate":   formatPaymentDate,
		"formatMoney":         formatMoney,
		"payrollDueSubtitle":  payrollDueSubtitle,
		"employeeCountPhrase": employeeCountPhrase,
	}
}

func shiftCountLabel(n int) string {
	mod := n % 100
	if mod >= 11 && mod <= 19 {
		return "смен"
	}
	switch n % 10 {
	case 1:
		return "смена"
	case 2, 3, 4:
		return "смены"
	default:
		return "смен"
	}
}

var paymentMonthsRu = []string{
	"",
	"января",
	"февраля",
	"марта",
	"апреля",
	"мая",
	"июня",
	"июля",
	"августа",
	"сентября",
	"октября",
	"ноября",
	"декабря",
}

func formatPaymentDate(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "—"
	}

	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, value); err == nil {
			return formatDateRu(t)
		}
	}
	return value
}

func formatDateRu(t time.Time) string {
	return strconv.Itoa(t.Day()) + " " + paymentMonthsRu[int(t.Month())] + " " + strconv.Itoa(t.Year())
}

var weekdaysRu = map[time.Weekday]string{
	time.Monday:    "Понедельник",
	time.Tuesday:   "Вторник",
	time.Wednesday: "Среда",
	time.Thursday:  "Четверг",
	time.Friday:    "Пятница",
	time.Saturday:  "Суббота",
	time.Sunday:    "Воскресенье",
}

// formatTodayRu renders "Понедельник, 13 июля" for the Today header.
func formatTodayRu(t time.Time) string {
	return weekdaysRu[t.Weekday()] + ", " + strconv.Itoa(t.Day()) + " " + paymentMonthsRu[int(t.Month())]
}

// closingPhrase describes how much of the open period is left, calmly.
func closingPhrase(daysLeft int) string {
	switch {
	case daysLeft <= 0:
		return "закрывается сегодня"
	case daysLeft == 1:
		return "закрывается завтра"
	default:
		return "закрывается через " + strconv.Itoa(daysLeft) + " " + dayWordRu(daysLeft)
	}
}

func dayWordRu(n int) string {
	mod100 := n % 100
	if mod100 >= 11 && mod100 <= 14 {
		return "дней"
	}
	switch n % 10 {
	case 1:
		return "день"
	case 2, 3, 4:
		return "дня"
	default:
		return "дней"
	}
}

func formatMoney(amount float64) string {
	sign := ""
	value := amount
	if value < 0 {
		sign = "− "
		value = -value
	}

	whole := int64(value + 0.0001)
	text := strconv.FormatInt(whole, 10)
	if len(text) <= 3 {
		return sign + text + " ₽"
	}

	var parts []string
	for len(text) > 3 {
		parts = append([]string{text[len(text)-3:]}, parts...)
		text = text[:len(text)-3]
	}
	if text != "" {
		parts = append([]string{text}, parts...)
	}
	return sign + strings.Join(parts, " ") + " ₽"
}

func payrollDueSubtitle(summary payroll.DueSummary) string {
	var parts []string
	if summary.AttentionCount > 0 {
		parts = append(parts, fmt.Sprintf("Ещё %s %s", employeeCountPhrase(summary.AttentionCount), "требуют внимания"))
	}
	if summary.IdleCount > 0 {
		parts = append(parts, fmt.Sprintf("Ещё %s %s", employeeCountPhrase(summary.IdleCount), "без смен за период"))
	}
	if len(parts) == 0 {
		if summary.PaidCount > 0 && summary.ReadyCount == 0 {
			return "Все выплаты за период подтверждены"
		}
		return ""
	}
	return strings.Join(parts, ". ")
}

func employeeCountPhrase(n int) string {
	mod := n % 100
	if mod >= 11 && mod <= 19 {
		return fmt.Sprintf("%d сотрудников", n)
	}
	switch n % 10 {
	case 1:
		return fmt.Sprintf("%d сотрудник", n)
	case 2, 3, 4:
		return fmt.Sprintf("%d сотрудника", n)
	default:
		return fmt.Sprintf("%d сотрудников", n)
	}
}
