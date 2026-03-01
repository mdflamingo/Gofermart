DROP TABLE IF EXISTS orders;
DROP INDEX IF EXISTS idx_orders_order_num;
DROP INDEX IF EXISTS idx_orders_user_id;
DROP INDEX IF EXISTS idx_orders_status;
DROP TYPE IF EXISTS orderstatus CASCADE;
