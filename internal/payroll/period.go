package payroll

import (
	"fmt"
	"time"
)

type Preset string

const (
	PresetFirstHalf  Preset = "first_half"
	PresetSecondHalf Preset = "second_half"
	PresetCustom     Preset = "custom"
)

type Period struct {
	From string
	To   string
}

func (p Preset) IsValid() bool {
	switch p {
	case PresetFirstHalf, PresetSecondHalf, PresetCustom:
		return true
	default:
		return false
	}
}

func ResolvePeriod(preset Preset, month, from, to string) (Period, error) {
	switch preset {
	case PresetFirstHalf, PresetSecondHalf:
		if month == "" {
			return Period{}, fmt.Errorf("укажите месяц")
		}
		t, err := time.Parse("2006-01", month)
		if err != nil {
			return Period{}, fmt.Errorf("неверный формат месяца")
		}
		if preset == PresetFirstHalf {
			return Period{
				From: t.Format("2006-01-02"),
				To:   time.Date(t.Year(), t.Month(), 15, 0, 0, 0, 0, time.UTC).Format("2006-01-02"),
			}, nil
		}
		lastDay := time.Date(t.Year(), t.Month()+1, 0, 0, 0, 0, 0, time.UTC)
		return Period{
			From: time.Date(t.Year(), t.Month(), 16, 0, 0, 0, 0, time.UTC).Format("2006-01-02"),
			To:   lastDay.Format("2006-01-02"),
		}, nil
	case PresetCustom:
		if from == "" || to == "" {
			return Period{}, fmt.Errorf("укажите даты периода")
		}
		fromDate, err := time.Parse("2006-01-02", from)
		if err != nil {
			return Period{}, fmt.Errorf("неверная дата начала")
		}
		toDate, err := time.Parse("2006-01-02", to)
		if err != nil {
			return Period{}, fmt.Errorf("неверная дата окончания")
		}
		if toDate.Before(fromDate) {
			return Period{}, fmt.Errorf("дата окончания раньше даты начала")
		}
		return Period{From: from, To: to}, nil
	default:
		return Period{}, fmt.Errorf("выберите период")
	}
}

var monthsRu = []string{
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

func (p Period) Label() string {
	from, err := time.Parse("2006-01-02", p.From)
	if err != nil {
		return p.From + " — " + p.To
	}
	to, err := time.Parse("2006-01-02", p.To)
	if err != nil {
		return p.From + " — " + p.To
	}

	if from.Year() == to.Year() && from.Month() == to.Month() {
		if from.Day() == 1 && to.Day() == 15 {
			return fmt.Sprintf("1–15 %s %d", monthsRu[int(from.Month())], from.Year())
		}
		lastDay := time.Date(from.Year(), from.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
		if from.Day() == 16 && to.Day() == lastDay {
			return fmt.Sprintf("16–%d %s %d", lastDay, monthsRu[int(from.Month())], from.Year())
		}
		return fmt.Sprintf("%d–%d %s %d", from.Day(), to.Day(), monthsRu[int(from.Month())], from.Year())
	}

	return fmt.Sprintf("%d %s %d — %d %s %d",
		from.Day(), monthsRu[int(from.Month())], from.Year(),
		to.Day(), monthsRu[int(to.Month())], to.Year(),
	)
}

func DefaultPreset(now time.Time) Preset {
	if now.Day() <= 15 {
		return PresetFirstHalf
	}
	return PresetSecondHalf
}

func PeriodFromDates(from, to string) (Period, error) {
	if from == "" || to == "" {
		return Period{}, fmt.Errorf("укажите даты периода")
	}
	fromDate, err := time.Parse("2006-01-02", from)
	if err != nil {
		return Period{}, fmt.Errorf("неверная дата начала")
	}
	toDate, err := time.Parse("2006-01-02", to)
	if err != nil {
		return Period{}, fmt.Errorf("неверная дата окончания")
	}
	if toDate.Before(fromDate) {
		return Period{}, fmt.Errorf("дата окончания раньше даты начала")
	}
	return Period{From: from, To: to}, nil
}
