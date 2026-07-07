package operation

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("operation not found")

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

const selectColumns = `
	id, employee_id, op_date, operation_type, amount, comment, COALESCE(cancelled_at, '')
`

func (s *Store) ListByEmployee(ctx context.Context, employeeID int64) ([]Operation, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+selectColumns+`
		FROM operations
		WHERE employee_id = ? AND cancelled_at IS NULL
		ORDER BY op_date DESC, id DESC
	`, employeeID)
	if err != nil {
		return nil, fmt.Errorf("list operations: %w", err)
	}
	defer rows.Close()

	return scanOperations(rows)
}

func (s *Store) ListByEmployeePeriod(ctx context.Context, employeeID int64, from, to string) ([]Operation, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+selectColumns+`
		FROM operations
		WHERE employee_id = ? AND op_date >= ? AND op_date <= ? AND cancelled_at IS NULL
		ORDER BY op_date ASC, id ASC
	`, employeeID, from, to)
	if err != nil {
		return nil, fmt.Errorf("list operations by period: %w", err)
	}
	defer rows.Close()

	return scanOperations(rows)
}

func (s *Store) GetByID(ctx context.Context, id int64) (Operation, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+selectColumns+`
		FROM operations
		WHERE id = ?
	`, id)

	op, err := scanOperation(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Operation{}, ErrNotFound
		}
		return Operation{}, err
	}
	return op, nil
}

func (s *Store) Create(ctx context.Context, op Operation) (Operation, error) {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO operations (employee_id, op_date, operation_type, amount, comment)
		VALUES (?, ?, ?, ?, ?)
	`, op.EmployeeID, op.Date, string(op.Type), op.Amount, op.Comment)
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

func (s *Store) Update(ctx context.Context, op Operation) (Operation, error) {
	result, err := s.db.ExecContext(ctx, `
		UPDATE operations
		SET employee_id = ?, op_date = ?, operation_type = ?, amount = ?, comment = ?
		WHERE id = ? AND cancelled_at IS NULL
	`, op.EmployeeID, op.Date, string(op.Type), op.Amount, op.Comment, op.ID)
	if err != nil {
		return Operation{}, fmt.Errorf("update operation: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return Operation{}, fmt.Errorf("rows affected: %w", err)
	}
	if affected == 0 {
		return Operation{}, ErrNotFound
	}

	return s.GetByID(ctx, op.ID)
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM operations WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete operation: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *Store) Cancel(ctx context.Context, id int64) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE operations
		SET cancelled_at = datetime('now')
		WHERE id = ? AND cancelled_at IS NULL
	`, id)
	if err != nil {
		return fmt.Errorf("cancel operation: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if affected == 0 {
		return ErrNotFound
	}

	return nil
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

func scanOperation(scanner interface {
	Scan(dest ...any) error
}) (Operation, error) {
	var op Operation
	var opType string
	if err := scanner.Scan(
		&op.ID, &op.EmployeeID, &op.Date, &opType, &op.Amount, &op.Comment, &op.CancelledAt,
	); err != nil {
		return Operation{}, fmt.Errorf("scan operation: %w", err)
	}
	op.Type = Type(opType)
	if op.Type == "shift" {
		return Operation{}, fmt.Errorf("scan operation: unexpected shift row")
	}
	return op, nil
}
