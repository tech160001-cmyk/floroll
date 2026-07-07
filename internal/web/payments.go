package web

import (
	"bytes"
	"html/template"
	"net/http"

	"floroll/internal/payment"
	"floroll/internal/payroll"
)

type paymentsPageData struct {
	Items []paymentHistoryView
}

type paymentHistoryView struct {
	Payment      payment.Payment
	EmployeeName string
	PeriodLabel  string
}

func (h *Handler) payments(w http.ResponseWriter, r *http.Request) {
	items, err := h.paymentStore.ListHistory(r.Context())
	if err != nil {
		h.renderPageError(w, "История выплат", "Не удалось загрузить историю выплат. Попробуйте обновить страницу.")
		return
	}

	viewItems := make([]paymentHistoryView, 0, len(items))
	for _, item := range items {
		viewItems = append(viewItems, paymentHistoryView{
			Payment:      item.Payment,
			EmployeeName: item.EmployeeName,
			PeriodLabel: payroll.Period{
				From: item.Payment.PeriodFrom,
				To:   item.Payment.PeriodTo,
			}.Label(),
		})
	}

	var buf bytes.Buffer
	if err := h.templates.ExecuteTemplate(&buf, "payments-content", paymentsPageData{
		Items: viewItems,
	}); err != nil {
		h.renderPageError(w, "История выплат", "Не удалось отобразить историю выплат.")
		return
	}

	h.renderPage(w, pageData{
		Title:   "История выплат",
		Content: template.HTML(buf.String()),
	})
}
