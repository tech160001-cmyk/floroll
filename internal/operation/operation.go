package operation

type Type string

const (
	TypeShift   Type = "shift"
	TypeFine    Type = "fine"
	TypeAdvance Type = "advance"
	TypeDebt    Type = "debt"
	TypeBonus   Type = "bonus"
)

func (t Type) Label() string {
	switch t {
	case TypeShift:
		return "Смена"
	case TypeFine:
		return "Штраф"
	case TypeAdvance:
		return "Аванс"
	case TypeDebt:
		return "Товар в долг"
	case TypeBonus:
		return "Бонус"
	default:
		return string(t)
	}
}

func FormTypes() []Type {
	return []Type{TypeFine, TypeAdvance, TypeDebt, TypeBonus}
}

type ShiftKind string

const (
	ShiftRegular    ShiftKind = "regular"
	ShiftSubstitute ShiftKind = "substitute"
)

func (k ShiftKind) Label() string {
	switch k {
	case ShiftSubstitute:
		return "Подработка"
	default:
		return "Обычная"
	}
}

type Operation struct {
	ID           int64
	EmployeeID   int64
	EmployeeName string
	Date         string
	Type         Type
	Amount       float64
	Comment      string
	Shop         string
	Revenue      float64
	ShiftKind    ShiftKind
}

func (o Operation) IsShift() bool {
	return o.Type == TypeShift
}
