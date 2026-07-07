package payment

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("payment not found")

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

const selectColumns = `
	id, employee_id, period_from, period_to, amount, comment, created_at,
	shift_pay, revenue_bonus, bonus_total, advance_total, fine_total, flowers_total
`

func (s *Store) FindByEmployeePeriod(ctx context.Context, employeeID int64, from, to string) (Payment, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT `+selectColumns+`
		FROM payments
		WHERE employee_id = ? AND period_from = ? AND period_to = ?
	`, employeeID, from, to)

	p, err := scanPayment(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Payment{}, ErrNotFound
		}
		return Payment{}, fmt.Errorf("find payment: %w", err)
	}
	return p, nil
}

func (s *Store) Create(ctx context.Context, p Payment) (Payment, error) {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO payments (
			employee_id, period_from, period_to, amount, comment,
			shift_pay, revenue_bonus, bonus_total, advance_total, fine_total, flowers_total
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		p.EmployeeID, p.PeriodFrom, p.PeriodTo, p.Amount, p.Comment,
		p.Snapshot.ShiftPay, p.Snapshot.RevenueBonus, p.Snapshot.BonusTotal,
		p.Snapshot.AdvanceTotal, p.Snapshot.FineTotal, p.Snapshot.FlowersTotal,
	)
	if err != nil {
		return Payment{}, fmt.Errorf("create payment: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return Payment{}, fmt.Errorf("last insert id: %w", err)
	}

	p.ID = id
	if p.CreatedAt == "" {
		row := s.db.QueryRowContext(ctx, `
			SELECT `+selectColumns+` FROM payments WHERE id = ?
		`, id)
		p, err = scanPayment(row)
		if err != nil {
			return Payment{}, fmt.Errorf("read payment: %w", err)
		}
	}
	return p, nil
}

func (s *Store) ExistsForEmployeeDate(ctx context.Context, employeeID int64, date string) (bool, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM payments
		WHERE employee_id = ? AND period_from <= ? AND period_to >= ?
	`, employeeID, date, date).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check payment coverage: %w", err)
	}
	return count > 0, nil
}

func scanPayment(scanner interface {
	Scan(dest ...any) error
}) (Payment, error) {
	var p Payment
	if err := scanner.Scan(
		&p.ID, &p.EmployeeID, &p.PeriodFrom, &p.PeriodTo, &p.Amount, &p.Comment, &p.CreatedAt,
		&p.Snapshot.ShiftPay, &p.Snapshot.RevenueBonus, &p.Snapshot.BonusTotal,
		&p.Snapshot.AdvanceTotal, &p.Snapshot.FineTotal, &p.Snapshot.FlowersTotal,
	); err != nil {
		return Payment{}, err
	}
	return p, nil
}
