CREATE TABLE orders (
    id INTEGER PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    'number' VARCHAR NOT NULL UNIQUE,
    user_id BIGINT NOT NULL,
    uploaded_at TIMESTAMPTZ DEFAULT now() NOT NULL,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX idx_orders_order_num ON orders(order_num);
