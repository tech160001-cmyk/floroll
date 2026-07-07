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
		"005_operation_types.sql",
		"006_split_shifts.sql",
		"007_payments_period.sql",
		"008_payments_snapshot.sql",
		"009_soft_archive.sql",
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

	if err := updateOperationTypesIfNeeded(database); err != nil {
		return fmt.Errorf("update operation types: %w", err)
	}

	if err := splitShiftsFromOperationsIfNeeded(database); err != nil {
		return fmt.Errorf("split shifts: %w", err)
	}

	if err := updatePaymentsSchemaIfNeeded(database); err != nil {
		return fmt.Errorf("update payments schema: %w", err)
	}

	if err := addPaymentSnapshotIfNeeded(database); err != nil {
		return fmt.Errorf("add payment snapshot: %w", err)
	}

	if err := addSoftArchiveColumnsIfNeeded(database); err != nil {
		return fmt.Errorf("add soft archive columns: %w", err)
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

func updateOperationTypesIfNeeded(database *sql.DB) error {
	var count int
	err := database.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('operations') WHERE name = 'operation_type'
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
			operation_type TEXT NOT NULL,
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
		INSERT INTO operations_new (
			id, employee_id, op_date, operation_type, amount, comment, shop, revenue, shift_kind
		)
		SELECT
			id,
			employee_id,
			op_date,
			CASE op_type WHEN 'debt' THEN 'flowers' ELSE op_type END,
			amount,
			comment,
			shop,
			revenue,
			shift_kind
		FROM operations
	`); err != nil {
		return err
	}

	if _, err := tx.Exec(`DROP TABLE operations`); err != nil {
		return err
	}

	if _, err := tx.Exec(`ALTER TABLE operations_new RENAME TO operations`); err != nil {
		return err
	}

	return tx.Commit()
}

func splitShiftsFromOperationsIfNeeded(database *sql.DB) error {
	var count int
	err := database.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='payments'
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

	hasOperationType, err := columnExists(tx, "operations", "operation_type")
	if err != nil {
		return err
	}

	if _, err := tx.Exec(`DROP TABLE IF EXISTS shifts`); err != nil {
		return err
	}

	if _, err := tx.Exec(`
		CREATE TABLE shifts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			employee_id INTEGER NOT NULL REFERENCES employees(id),
			shift_date TEXT NOT NULL,
			shop TEXT NOT NULL,
			revenue REAL NOT NULL,
			shop_revenue REAL NOT NULL DEFAULT 0,
			comment TEXT NOT NULL DEFAULT '',
			payment REAL NOT NULL DEFAULT 0
		)
	`); err != nil {
		return err
	}

	if hasOperationType {
		if _, err := tx.Exec(`
			INSERT INTO shifts (employee_id, shift_date, shop, revenue, shop_revenue, comment, payment)
			SELECT
				employee_id,
				op_date,
				COALESCE(shop, ''),
				COALESCE(revenue, 0),
				0,
				comment,
				amount
			FROM operations
			WHERE operation_type = 'shift'
		`); err != nil {
			return err
		}
	}

	if _, err := tx.Exec(`
		CREATE TABLE operations_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			employee_id INTEGER NOT NULL REFERENCES employees(id),
			op_date TEXT NOT NULL,
			operation_type TEXT NOT NULL,
			amount REAL NOT NULL,
			comment TEXT NOT NULL DEFAULT ''
		)
	`); err != nil {
		return err
	}

	if hasOperationType {
		if _, err := tx.Exec(`
			INSERT INTO operations_new (id, employee_id, op_date, operation_type, amount, comment)
			SELECT id, employee_id, op_date, operation_type, amount, comment
			FROM operations
			WHERE operation_type != 'shift'
		`); err != nil {
			return err
		}
	} else {
		hasOpType, err := columnExists(tx, "operations", "op_type")
		if err != nil {
			return err
		}
		if hasOpType {
			if _, err := tx.Exec(`
				INSERT INTO operations_new (id, employee_id, op_date, operation_type, amount, comment)
				SELECT
					id,
					employee_id,
					op_date,
					CASE op_type WHEN 'debt' THEN 'flowers' ELSE op_type END,
					amount,
					comment
				FROM operations
				WHERE op_type != 'shift'
			`); err != nil {
				return err
			}
		}
	}

	if _, err := tx.Exec(`DROP TABLE IF EXISTS operations`); err != nil {
		return err
	}

	if _, err := tx.Exec(`ALTER TABLE operations_new RENAME TO operations`); err != nil {
		return err
	}

	if _, err := tx.Exec(`
		CREATE TABLE payments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			employee_id INTEGER NOT NULL REFERENCES employees(id),
			period_from TEXT NOT NULL,
			period_to TEXT NOT NULL,
			amount REAL NOT NULL,
			comment TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL DEFAULT (datetime('now')),
			shift_pay REAL NOT NULL DEFAULT 0,
			revenue_bonus REAL NOT NULL DEFAULT 0,
			bonus_total REAL NOT NULL DEFAULT 0,
			advance_total REAL NOT NULL DEFAULT 0,
			fine_total REAL NOT NULL DEFAULT 0,
			flowers_total REAL NOT NULL DEFAULT 0
		)
	`); err != nil {
		return err
	}

	return tx.Commit()
}

func updatePaymentsSchemaIfNeeded(database *sql.DB) error {
	var tableCount int
	if err := database.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='payments'
	`).Scan(&tableCount); err != nil {
		return err
	}
	if tableCount == 0 {
		return nil
	}

	var count int
	if err := database.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('payments') WHERE name = 'period_from'
	`).Scan(&count); err != nil {
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

	hasPaymentDate, err := columnExists(tx, "payments", "payment_date")
	if err != nil {
		return err
	}

	if _, err := tx.Exec(`
		CREATE TABLE payments_new (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			employee_id INTEGER NOT NULL REFERENCES employees(id),
			period_from TEXT NOT NULL,
			period_to TEXT NOT NULL,
			amount REAL NOT NULL,
			comment TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL DEFAULT (datetime('now'))
		)
	`); err != nil {
		return err
	}

	if hasPaymentDate {
		hasConfirmedAt, err := columnExists(tx, "payments", "confirmed_at")
		if err != nil {
			return err
		}
		if hasConfirmedAt {
			if _, err := tx.Exec(`
				INSERT INTO payments_new (id, employee_id, period_from, period_to, amount, comment, created_at)
				SELECT id, employee_id, payment_date, payment_date, amount, comment, confirmed_at
				FROM payments
			`); err != nil {
				return err
			}
		} else {
			if _, err := tx.Exec(`
				INSERT INTO payments_new (id, employee_id, period_from, period_to, amount, comment, created_at)
				SELECT id, employee_id, payment_date, payment_date, amount, comment, datetime('now')
				FROM payments
			`); err != nil {
				return err
			}
		}
	}

	if _, err := tx.Exec(`DROP TABLE payments`); err != nil {
		return err
	}

	if _, err := tx.Exec(`ALTER TABLE payments_new RENAME TO payments`); err != nil {
		return err
	}

	return tx.Commit()
}

func addPaymentSnapshotIfNeeded(database *sql.DB) error {
	var tableCount int
	if err := database.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='payments'
	`).Scan(&tableCount); err != nil {
		return err
	}
	if tableCount == 0 {
		return nil
	}

	var count int
	if err := database.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info('payments') WHERE name = 'shift_pay'
	`).Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	columns := []string{
		`ALTER TABLE payments ADD COLUMN shift_pay REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE payments ADD COLUMN revenue_bonus REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE payments ADD COLUMN bonus_total REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE payments ADD COLUMN advance_total REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE payments ADD COLUMN fine_total REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE payments ADD COLUMN flowers_total REAL NOT NULL DEFAULT 0`,
	}
	for _, stmt := range columns {
		if _, err := database.Exec(stmt); err != nil {
			return err
		}
	}

	return nil
}

func addSoftArchiveColumnsIfNeeded(database *sql.DB) error {
	columns := []struct {
		table  string
		column string
		stmt   string
	}{
		{"employees", "archived_at", `ALTER TABLE employees ADD COLUMN archived_at TEXT`},
		{"shifts", "cancelled_at", `ALTER TABLE shifts ADD COLUMN cancelled_at TEXT`},
		{"operations", "cancelled_at", `ALTER TABLE operations ADD COLUMN cancelled_at TEXT`},
	}

	for _, col := range columns {
		exists, err := columnExistsDB(database, col.table, col.column)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		if _, err := database.Exec(col.stmt); err != nil {
			return err
		}
	}

	return nil
}

func columnExistsDB(database *sql.DB, table, column string) (bool, error) {
	var count int
	err := database.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?
	`, table, column).Scan(&count)
	return count > 0, err
}

func columnExists(tx *sql.Tx, table, column string) (bool, error) {
	var count int
	err := tx.QueryRow(`
		SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?
	`, table, column).Scan(&count)
	return count > 0, err
}

func tableExists(tx *sql.Tx, name string) bool {
	var count int
	err := tx.QueryRow(`
		SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?
	`, name).Scan(&count)
	return err == nil && count > 0
}
