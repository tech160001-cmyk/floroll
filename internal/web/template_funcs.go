package web

import (
	"strconv"
	"strings"
	"time"
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
