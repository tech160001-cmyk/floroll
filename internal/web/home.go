package web

import (
	"floroll/internal/operation"
	"floroll/internal/shift"
)

type homePageData struct {
	EmployeeCount  int
	ShiftCount     int
	OperationCount int
	PaymentCount   int

	LatestShift     *shift.Shift
	LatestOperation *operation.Operation
	LatestPayment   *paymentHistoryView
}
