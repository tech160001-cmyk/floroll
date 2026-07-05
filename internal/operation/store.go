package operation

import (
	"context"
	"database/sql"
	"fmt"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

const selectColumns = `
	id, employee_id, op_date, op_type, amount, comment,
	COALESCE(shop, ''), COALESCE(revenue, 0), COALESCE(shift_kind, '')
`

func (s *Store) ListByEmployee(ctx context.Context, employeeID int64) ([]Operation, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+selectColumns+`
		FROM operations
		WHERE employee_id = ?
		ORDER BY op_date DESC, id DESC
	`, employeeID)
	if err != nil {
		return nil, fmt.Errorf("list operations: %w", err)
	}
	defer rows.Close()

	return scanOperations(rows)
}

func (s *Store) ListShifts(ctx context.Context) ([]Operation, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT o.id, o.employee_id, o.op_date, o.op_type, o.amount, o.comment,
		       COALESCE(o.shop, ''), COALESCE(o.revenue, 0), COALESCE(o.shift_kind, ''),
		       e.name
		FROM operations o
		JOIN employees e ON e.id = o.employee_id
		WHERE o.op_type = 'shift'
		ORDER BY o.op_date DESC, o.id DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list shifts: %w", err)
	}
	defer rows.Close()

	return scanOperationsWithName(rows)
}

func (s *Store) Create(ctx context.Context, op Operation) (Operation, error) {
	var shop, shiftKind any
	var revenue any

	if op.IsShift() {
		shop = op.Shop
		revenue = op.Revenue
		shiftKind = string(op.ShiftKind)
	}

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO operations (employee_id, op_date, op_type, amount, comment, shop, revenue, shift_kind)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, op.EmployeeID, op.Date, string(op.Type), op.Amount, op.Comment, shop, revenue, shiftKind)
	if err != nil {
		return Operation{}, fmt.Errorf("create operation: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return Operation{}, fmt.Errorf("last insert id: %w", err)
	}

	op.ID = id
	return op, nil
}

func scanOperations(rows *sql.Rows) ([]Operation, error) {
	var ops []Operation
	for rows.Next() {
		op, err := scanOperation(rows)
		if err != nil {
			return nil, err
		}
		ops = append(ops, op)
	}
	return ops, rows.Err()
}

func scanOperationsWithName(rows *sql.Rows) ([]Operation, error) {
	var ops []Operation
	for rows.Next() {
		var op Operation
		var opType string
		if err := rows.Scan(
			&op.ID, &op.EmployeeID, &op.Date, &opType, &op.Amount, &op.Comment,
			&op.Shop, &op.Revenue, &op.ShiftKind, &op.EmployeeName,
		); err != nil {
			return nil, fmt.Errorf("scan operation: %w", err)
		}
		op.Type = Type(opType)
		ops = append(ops, op)
	}
	return ops, rows.Err()
}

func scanOperation(scanner interface {
	Scan(dest ...any) error
}) (Operation, error) {
	var op Operation
	var opType string
	if err := scanner.Scan(
		&op.ID, &op.EmployeeID, &op.Date, &opType, &op.Amount, &op.Comment,
		&op.Shop, &op.Revenue, &op.ShiftKind,
	); err != nil {
		return Operation{}, fmt.Errorf("scan operation: %w", err)
	}
	op.Type = Type(opType)
	return op, nil
}
