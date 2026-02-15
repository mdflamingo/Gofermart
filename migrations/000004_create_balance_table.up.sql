CREATE TABLE balance (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL UNIQUE REFERENCES users(id),
    current REAL DEFAULT 0 NOT NULL,
    withdrawn REAL DEFAULT 0 NOT NULL
);

CREATE INDEX idx_balances_user_id ON balances(user_id);
