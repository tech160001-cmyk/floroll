package shift

type Shift struct {
	ID           int64
	EmployeeID   int64
	EmployeeName string
	Date         string
	Shop         string
	Revenue      float64
	ShopRevenue  float64
	Comment      string
	Payment      float64
	CancelledAt  string
}

func (s Shift) IsCancelled() bool {
	return s.CancelledAt != ""
}
