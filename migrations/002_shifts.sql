CREATE TABLE IF NOT EXISTS shifts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    employee_id INTEGER NOT NULL REFERENCES employees(id),
    shift_date TEXT NOT NULL,
    shop TEXT NOT NULL,
    revenue REAL NOT NULL,
    shift_type TEXT NOT NULL CHECK (shift_type IN ('regular', 'substitute')),
    payment REAL NOT NULL
);
