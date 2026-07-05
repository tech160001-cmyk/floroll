CREATE TABLE IF NOT EXISTS operations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    employee_id INTEGER NOT NULL REFERENCES employees(id),
    op_date TEXT NOT NULL,
    op_type TEXT NOT NULL CHECK (op_type IN ('fine', 'advance', 'debt', 'bonus')),
    amount REAL NOT NULL,
    comment TEXT NOT NULL DEFAULT ''
);
