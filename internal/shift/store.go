package shift

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("shift not found")

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

const selectColumns = `
	s.id, s.employee_id, s.shift_date, s.shop, s.revenue, s.shop_revenue,
	s.comment, s.payment, COALESCE(s.cancelled_at, ''), COALESCE(e.name, '')
`

func (s *Store) List(ctx context.Context) ([]Shift, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+selectColumns+`
		FROM shifts s
		LEFT JOIN employees e ON e.id = s.employee_id
		WHERE s.cancelled_at IS NULL
		ORDER BY s.shift_date DESC, s.id DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list shifts: %w", err)
	}
	defer rows.Close()

	return scanShifts(rows)
}

func (s *Store) ListByEmployeePeriod(ctx context.Context, employeeID int64, from, to string) ([]Shift, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT `+selectColumns+`
		FROM shifts s
		LEFT JOIN employees e ON e.id = s.employee_id
		WHERE s.employee_id = ? AND s.shift_date >= ? AND s.shift_date <= ? AND s.cancelled_at IS NULL
		ORDER BY s.shift_date ASC, s.id ASC
	`, employeeID, from, to)
	if err != nil {
		return nil, fmt.Errorf("list shifts by period: %w", err)
	}
	defer rows.Close()

	return scanShifts(rows)
}

func (s *Store) GetByID(ctx context.Context, id int64) (Shift, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+selectColumns+`
		FROM shifts s
		LEFT JOIN employees e ON e.id = s.employee_id
		WHERE s.id = ?
	`, id)

	sh, err := scanShift(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Shift{}, ErrNotFound
		}
		return Shift{}, err
	}
	return sh, nil
}

func (s *Store) Create(ctx context.Context, sh Shift) (Shift, error) {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO shifts (employee_id, shift_date, shop, revenue, shop_revenue, comment, payment)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, sh.EmployeeID, sh.Date, sh.Shop, sh.Revenue, sh.ShopRevenue, sh.Comment, sh.Payment)
	if err != nil {
		return Shift{}, fmt.Errorf("create shift: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return Shift{}, fmt.Errorf("last insert id: %w", err)
	}

	sh.ID = id
	return sh, nil
}

func (s *Store) Update(ctx context.Context, sh Shift) (Shift, error) {
	result, err := s.db.ExecContext(ctx, `
		UPDATE shifts
		SET employee_id = ?, shift_date = ?, shop = ?, revenue = ?, shop_revenue = ?, comment = ?, payment = ?
		WHERE id = ? AND cancelled_at IS NULL
	`, sh.EmployeeID, sh.Date, sh.Shop, sh.Revenue, sh.ShopRevenue, sh.Comment, sh.Payment, sh.ID)
	if err != nil {
		return Shift{}, fmt.Errorf("update shift: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return Shift{}, fmt.Errorf("rows affected: %w", err)
	}
	if affected == 0 {
		return Shift{}, ErrNotFound
	}

	return s.GetByID(ctx, sh.ID)
}

func (s *Store) Delete(ctx context.Context, id int64) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM shifts WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete shift: %w", err)
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
		UPDATE shifts
		SET cancelled_at = datetime('now')
		WHERE id = ? AND cancelled_at IS NULL
	`, id)
	if err != nil {
		return fmt.Errorf("cancel shift: %w", err)
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

func scanShifts(rows *sql.Rows) ([]Shift, error) {
	var shifts []Shift
	for rows.Next() {
		sh, err := scanShift(rows)
		if err != nil {
			return nil, err
		}
		shifts = append(shifts, sh)
	}
	return shifts, rows.Err()
}

func scanShift(scanner interface {
	Scan(dest ...any) error
}) (Shift, error) {
	var sh Shift
	if err := scanner.Scan(
		&sh.ID, &sh.EmployeeID, &sh.Date, &sh.Shop, &sh.Revenue, &sh.ShopRevenue,
		&sh.Comment, &sh.Payment, &sh.CancelledAt, &sh.EmployeeName,
	); err != nil {
		return Shift{}, fmt.Errorf("scan shift: %w", err)
	}
	return sh, nil
}
