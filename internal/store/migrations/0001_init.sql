-- 0001_init.sql: initial schema of Konzept.md §3.
--
-- Amounts are positive Rappen, the sign is added by the display. Dates are
-- calendar days in the format YYYY-MM-DD without a time.

CREATE TABLE accounts (
    id         INTEGER PRIMARY KEY,
    name       TEXT    NOT NULL UNIQUE,
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_active  INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE entries (
    id           INTEGER PRIMARY KEY,
    account_id   INTEGER NOT NULL REFERENCES accounts(id),
    name         TEXT    NOT NULL,
    amount_minor INTEGER NOT NULL CHECK (amount_minor > 0),
    start_date   TEXT    NOT NULL,
    unit         TEXT    NOT NULL CHECK (unit IN ('once', 'week', 'month', 'year')),
    interval     INTEGER NOT NULL DEFAULT 1 CHECK (interval > 0),
    ends_on      TEXT    NULL,
    notes        TEXT    NOT NULL DEFAULT ''
);

CREATE INDEX idx_entries_account_id ON entries(account_id);

-- The default accounts of Konzept.md §3. They are part of the migration, so
-- they exist exactly once in a new database and never come back after they
-- have been deleted.

INSERT INTO accounts (name, sort_order, is_active) VALUES
    ('Lifestyle', 1, 1),
    ('Sparen', 2, 1),
    ('Steuern', 3, 1);
