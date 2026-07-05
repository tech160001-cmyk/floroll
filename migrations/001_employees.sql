CREATE TABLE IF NOT EXISTS employees (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    shop TEXT NOT NULL,
    shift_rate REAL NOT NULL,
    revenue_percent REAL NOT NULL
);
