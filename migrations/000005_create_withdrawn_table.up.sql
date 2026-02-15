CREATE TABLE withdrawals (
    id INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    user_id INT NOT NULL REFERENCES users(id),
    "order" VARCHAR(255) NOT NULL,
    sum DECIMAL(10,2) NOT NULL,
    processed_at TIMESTAMP DEFAULT NOW() NOT NULL
);

CREATE INDEX idx_withdrawals_user_id ON withdrawals(user_id);
