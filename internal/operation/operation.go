package operation

type Type string

const (
	TypeAdvance Type = "advance"
	TypeFine    Type = "fine"
	TypeFlowers Type = "flowers"
	TypeBonus   Type = "bonus"
	TypeOther   Type = "other"
)

func FormTypes() []Type {
	return []Type{
		TypeAdvance,
		TypeFine,
		TypeFlowers,
		TypeBonus,
		TypeOther,
	}
}

func (t Type) Label() string {
	switch t {
	case TypeAdvance:
		return "Аванс"
	case TypeFine:
		return "Штраф"
	case TypeFlowers, "debt":
		return "Удержание за цветы"
	case TypeBonus:
		return "Премия"
	case TypeOther:
		return "Другое"
	default:
		return string(t)
	}
}

func (t Type) IsFormType() bool {
	for _, valid := range FormTypes() {
		if t == valid {
			return true
		}
	}
	return false
}

type Operation struct {
	ID           int64
	EmployeeID   int64
	EmployeeName string
	Date         string
	Type         Type
	Amount       float64
	Comment      string
	CancelledAt  string
}

func (o Operation) IsCancelled() bool {
	return o.CancelledAt != ""
}

func (t Type) TimelineClass() string {
	switch t {
	case TypeFlowers, "debt":
		return "flowers"
	case TypeAdvance, TypeFine, TypeBonus, TypeOther:
		return string(t)
	default:
		return "other"
	}
}
