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

type HistoryItem struct {
	Payment      Payment
	EmployeeName string
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

func (s *Store) Count(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM payments`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count payments: %w", err)
	}
	return count, nil
}

func (s *Store) LatestHistory(ctx context.Context) (HistoryItem, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT
			p.id, p.employee_id, p.period_from, p.period_to, p.amount, p.comment, p.created_at,
			p.shift_pay, p.revenue_bonus, p.bonus_total, p.advance_total, p.fine_total, p.flowers_total,
			e.name
		FROM payments p
		JOIN employees e ON e.id = p.employee_id
		ORDER BY p.created_at DESC, p.id DESC
		LIMIT 1
	`)

	item, err := scanHistoryItem(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return HistoryItem{}, ErrNotFound
		}
		return HistoryItem{}, fmt.Errorf("latest payment history: %w", err)
	}
	return item, nil
}

func (s *Store) ListHistory(ctx context.Context) ([]HistoryItem, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT
			p.id, p.employee_id, p.period_from, p.period_to, p.amount, p.comment, p.created_at,
			p.shift_pay, p.revenue_bonus, p.bonus_total, p.advance_total, p.fine_total, p.flowers_total,
			e.name
		FROM payments p
		JOIN employees e ON e.id = p.employee_id
		ORDER BY p.created_at DESC, p.id DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list payment history: %w", err)
	}
	defer rows.Close()

	var items []HistoryItem
	for rows.Next() {
		item, err := scanHistoryItem(rows)
		if err != nil {
			return nil, fmt.Errorf("scan payment history: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate payment history: %w", err)
	}
	return items, nil
}

func scanHistoryItem(scanner interface {
	Scan(dest ...any) error
}) (HistoryItem, error) {
	var item HistoryItem
	if err := scanner.Scan(
		&item.Payment.ID,
		&item.Payment.EmployeeID,
		&item.Payment.PeriodFrom,
		&item.Payment.PeriodTo,
		&item.Payment.Amount,
		&item.Payment.Comment,
		&item.Payment.CreatedAt,
		&item.Payment.Snapshot.ShiftPay,
		&item.Payment.Snapshot.RevenueBonus,
		&item.Payment.Snapshot.BonusTotal,
		&item.Payment.Snapshot.AdvanceTotal,
		&item.Payment.Snapshot.FineTotal,
		&item.Payment.Snapshot.FlowersTotal,
		&item.EmployeeName,
	); err != nil {
		return HistoryItem{}, err
	}
	return item, nil
}

func (s *Store) ListRecentByEmployee(ctx context.Context, employeeID int64, limit int) ([]Payment, error) {
	if limit <= 0 {
		limit = 5
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT `+selectColumns+`
		FROM payments
		WHERE employee_id = ?
		ORDER BY created_at DESC, id DESC
		LIMIT ?
	`, employeeID, limit)
	if err != nil {
		return nil, fmt.Errorf("list recent payments by employee: %w", err)
	}
	defer rows.Close()

	var payments []Payment
	for rows.Next() {
		p, err := scanPayment(rows)
		if err != nil {
			return nil, fmt.Errorf("scan recent payment: %w", err)
		}
		payments = append(payments, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recent payments: %w", err)
	}
	return payments, nil
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
