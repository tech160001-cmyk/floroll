package employee

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("employee not found")

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) List(ctx context.Context) ([]Employee, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, shop, shift_rate, revenue_percent, COALESCE(archived_at, '')
		FROM employees
		WHERE archived_at IS NULL
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("list employees: %w", err)
	}
	defer rows.Close()

	var employees []Employee
	for rows.Next() {
		var e Employee
		if err := rows.Scan(&e.ID, &e.Name, &e.Shop, &e.ShiftRate, &e.RevenuePercent, &e.ArchivedAt); err != nil {
			return nil, fmt.Errorf("scan employee: %w", err)
		}
		employees = append(employees, e)
	}

	return employees, rows.Err()
}

func (s *Store) Count(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM employees WHERE archived_at IS NULL`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count employees: %w", err)
	}
	return count, nil
}

func (s *Store) Create(ctx context.Context, e Employee) (Employee, error) {
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO employees (name, shop, shift_rate, revenue_percent)
		VALUES (?, ?, ?, ?)
	`, e.Name, e.Shop, e.ShiftRate, e.RevenuePercent)
	if err != nil {
		return Employee{}, fmt.Errorf("create employee: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return Employee{}, fmt.Errorf("last insert id: %w", err)
	}

	e.ID = id
	return e, nil
}

func (s *Store) GetByID(ctx context.Context, id int64) (Employee, error) {
	var e Employee
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, shop, shift_rate, revenue_percent, COALESCE(archived_at, '')
		FROM employees
		WHERE id = ?
	`, id).Scan(&e.ID, &e.Name, &e.Shop, &e.ShiftRate, &e.RevenuePercent, &e.ArchivedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Employee{}, ErrNotFound
		}
		return Employee{}, fmt.Errorf("get employee: %w", err)
	}

	return e, nil
}

func (s *Store) Update(ctx context.Context, e Employee) (Employee, error) {
	result, err := s.db.ExecContext(ctx, `
		UPDATE employees
		SET name = ?, shop = ?, shift_rate = ?, revenue_percent = ?
		WHERE id = ?
	`, e.Name, e.Shop, e.ShiftRate, e.RevenuePercent, e.ID)
	if err != nil {
		return Employee{}, fmt.Errorf("update employee: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return Employee{}, fmt.Errorf("rows affected: %w", err)
	}
	if affected == 0 {
		return Employee{}, ErrNotFound
	}

	return s.GetByID(ctx, e.ID)
}

func (s *Store) Archive(ctx context.Context, id int64) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE employees
		SET archived_at = datetime('now')
		WHERE id = ? AND archived_at IS NULL
	`, id)
	if err != nil {
		return fmt.Errorf("archive employee: %w", err)
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
