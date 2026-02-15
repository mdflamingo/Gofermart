DO $$
BEGIN
    CREATE TYPE orderstatus AS ENUM ('NEW', 'PROCESSING', 'INVALID', 'PROCESSED');
EXCEPTION
    WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE orders (
    id INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    number VARCHAR(255) NOT NULL UNIQUE,
    user_id INT NOT NULL REFERENCES users(id),
    status orderstatus DEFAULT 'NEW' NOT NULL,
    accrual REAL DEFAULT 0 NOT NULL,
    uploaded_at TIMESTAMPTZ DEFAULT NOW() NOT NULL
);

CREATE INDEX idx_orders_number ON orders(number);
CREATE INDEX idx_orders_user_id ON orders(user_id);
CREATE INDEX idx_orders_status ON orders(status);
