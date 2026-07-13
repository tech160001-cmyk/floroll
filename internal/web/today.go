package web

import (
	"net/http"
	"time"
)

// todayKind is the single "what to do today" verdict shown in the hero.
// Exactly one kind is active at a time, chosen by priority.
type todayKind string

const (
	todayNoEmployees todayKind = "no_employees"
	todayReadyToPay  todayKind = "ready_to_pay"
	todayClosingSoon todayKind = "closing_soon"
	todayCalm        todayKind = "calm"
)

// closingSoonDays is how many days before period close we start nudging.
const closingSoonDays = 3

type todayPageData struct {
	DateLabel   string
	PeriodLabel string
	ClosingNote string

	Kind todayKind

	TotalToPay float64
	ReadyCount int

	AttentionCount int
}

func (d todayPageData) IsReadyToPay() bool  { return d.Kind == todayReadyToPay }
func (d todayPageData) IsClosingSoon() bool { return d.Kind == todayClosingSoon }
func (d todayPageData) IsCalm() bool        { return d.Kind == todayCalm }
func (d todayPageData) IsNoEmployees() bool { return d.Kind == todayNoEmployees }

// ShowShiftAction reports whether the quiet "Внести смену" action belongs on
// screen — always, unless there is no team yet.
func (d todayPageData) ShowShiftAction() bool { return d.Kind != todayNoEmployees }

// buildTodayData derives the morning glance from existing read-models only.
func (h *Handler) buildTodayData(r *http.Request) (todayPageData, error) {
	page, err := h.buildPayoutsPage(r.Context(), r)
	if err != nil {
		return todayPageData{}, err
	}

	now := time.Now()
	data := todayPageData{
		DateLabel:      formatTodayRu(now),
		PeriodLabel:    page.PeriodLabel,
		ClosingNote:    closingPhrase(page.Open.DaysLeft),
		TotalToPay:     page.TotalToPay,
		ReadyCount:     len(page.Ready),
		AttentionCount: len(page.Attention),
	}

	switch {
	case !page.HasEmployees():
		data.Kind = todayNoEmployees
	case len(page.Ready) > 0:
		data.Kind = todayReadyToPay
	case page.Open.DaysLeft <= closingSoonDays:
		data.Kind = todayClosingSoon
	default:
		data.Kind = todayCalm
	}

	return data, nil
}
