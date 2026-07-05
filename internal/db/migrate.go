package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
)

func Migrate(database *sql.DB) error {
	root := "."
	if env := os.Getenv("FLOROLL_ROOT"); env != "" {
		root = env
	}

	files := []string{
		"001_employees.sql",
		"002_shifts.sql",
		"003_operations.sql",
		"004_unify_operations.sql",
	}

	for _, file := range files {
		path := filepath.Join(root, "migrations", file)
		sqlBytes, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", file, err)
		}

		if _, err := database.Exec(string(sqlBytes)); err != nil {
			return fmt.Errorf("run migration %s: %w", file, err)
		}
	}

	if err := unifyOperationsIfNeeded(database); err != nil {
		return fmt.Errorf("unify operations: %w", err)
	}

	return nil
}

func unifyOperationsIfNeeded(database *sql.DB) error {
	var count int
	err := database.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('operations') WHERE name = 'shop'
	`).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	tx, err := database.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`
		CREATE TABLE operations_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			employee_id INTEGER NOT NULL REFERENCES employees(id),
			op_date TEXT NOT NULL,
			op_type TEXT NOT NULL CHECK (op_type IN ('shift', 'fine', 'advance', 'debt', 'bonus')),
			amount REAL NOT NULL,
			comment TEXT NOT NULL DEFAULT '',
			shop TEXT,
			revenue REAL,
			shift_kind TEXT
		)
	`); err != nil {
		return err
	}

	if _, err := tx.Exec(`
		INSERT INTO operations_new (id, employee_id, op_date, op_type, amount, comment)
		SELECT id, employee_id, op_date, op_type, amount, comment FROM operations
	`); err != nil {
		return err
	}

	if tableExists(tx, "shifts") {
		if _, err := tx.Exec(`
			INSERT INTO operations_new (employee_id, op_date, op_type, amount, comment, shop, revenue, shift_kind)
			SELECT employee_id, shift_date, 'shift', payment, '', shop, revenue, shift_type FROM shifts
		`); err != nil {
			return err
		}
		if _, err := tx.Exec(`DROP TABLE shifts`); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(`DROP TABLE operations`); err != nil {
		return err
	}

	if _, err := tx.Exec(`ALTER TABLE operations_new RENAME TO operations`); err != nil {
		return err
	}

	return tx.Commit()
}

func tableExists(tx *sql.Tx, name string) bool {
	var count int
	err := tx.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?
	`, name).Scan(&count)
	return err == nil && count > 0
}
