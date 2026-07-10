package web

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"floroll/internal/payroll"
)

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
