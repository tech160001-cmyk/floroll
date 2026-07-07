package payment

type Payment struct {
	ID         int64
	EmployeeID int64
	PeriodFrom string
	PeriodTo   string
	Amount     float64
	Comment    string
	CreatedAt  string
	Snapshot   Snapshot
}
