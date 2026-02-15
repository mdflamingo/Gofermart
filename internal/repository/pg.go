package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/mdflamingo/Gofermart/internal/models"
)

var ErrConflict = errors.New("conflict: duplicate entry")
var ErrNotFound = errors.New("obj not found")
var ErrInsufficientFunds = errors.New("insufficient funds")

type Order struct {
	Number     string
	Status     string
	Accrual    float64
	UploadedAt time.Time
}

type Balance struct {
	Current   float64
	Withdrawn float64
}

type Withdrawal struct {
	Order       string
	Sum         float64
	ProcessedAt time.Time
}

type OrderToUpdate struct {
	ID     int
	Number string
	UserID int
}

type DBStorage struct {
	pool *pgxpool.Pool
}

func NewDBStorage(dsn string) (*DBStorage, error) {
	ctx := context.Background()

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnLifetime = time.Hour
	config.MaxConnIdleTime = 30 * time.Minute
	config.HealthCheckPeriod = time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	ctxPing, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctxPing); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if err := runMigrations(dsn); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return &DBStorage{pool: pool}, nil
}

func (d *DBStorage) Close() error {
	d.pool.Close()
	return nil
}

func runMigrations(dsn string) error {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("failed to create migration driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file://migrations",
		"postgres",
		driver)
	if err != nil {
		return fmt.Errorf("failed to create migrate instance: %w", err)
	}

	err = m.Up()
	if err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

func (d *DBStorage) Ping(ctx context.Context) error {
	return d.pool.Ping(ctx)
}

func (d *DBStorage) SaveUser(user models.UserDB) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var userID int

	err := d.pool.QueryRow(ctx,
		`INSERT INTO users (login, password)
         VALUES ($1, $2)
		 RETURNING id`,
		user.Login, user.Password).Scan(&userID)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return 0, ErrConflict
		}
		return 0, fmt.Errorf("failed to save user: %w", err)
	}

	return userID, nil
}

func (d *DBStorage) GetUser(user models.UserDB) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var userID int

	err := d.pool.QueryRow(ctx, `SELECT id FROM users WHERE login = $1 and password = $2`, user.Login, user.Password).Scan(&userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, ErrNotFound
		}
		return 0, fmt.Errorf("failed to get user: %w", err)
	}
	return userID, nil
}

func (d *DBStorage) SaveOrder(order string, userID int) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var existingUserID int
	err := d.pool.QueryRow(ctx, `SELECT user_id FROM orders WHERE number = $1`, order).Scan(&existingUserID)
	if err == nil {
		return existingUserID, ErrConflict
	} else if !errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("failed to check order existence: %w", err)
	}

	_, err = d.pool.Exec(ctx, `INSERT INTO orders (number, user_id) VALUES ($1, $2)`, order, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to save order number: %w", err)
	}

	return userID, nil
}

func (d *DBStorage) GetOrders(userID int) ([]Order, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := d.pool.Query(ctx,
		`SELECT number, status, accrual, uploaded_at FROM orders WHERE user_id = $1 ORDER BY uploaded_at DESC`,
		userID)

	if err != nil {
		return nil, fmt.Errorf("database query error: %w", err)
	}
	defer rows.Close()

	var orders []Order

	for rows.Next() {
		var order Order

		if err := rows.Scan(&order.Number, &order.Status, &order.Accrual, &order.UploadedAt); err != nil {
			return nil, fmt.Errorf("data scan error: %w", err)
		}
		orders = append(orders, order)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows processing error: %w", err)
	}

	return orders, nil
}

func (d *DBStorage) GetBalance(userID int) (Balance, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var balance Balance

	err := d.pool.QueryRow(ctx,
		`SELECT current, withdrawn FROM balance WHERE user_id = $1`,
		userID).Scan(&balance.Current, &balance.Withdrawn)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Balance{}, ErrNotFound
		}
		return Balance{}, err
	}
	return balance, nil
}

func (d *DBStorage) SaveWithdrawal(userID int, order string, sum float64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	balance, err := d.GetBalance(userID)
	if err != nil {
		return err
	}
	if balance.Current < sum {
		return ErrInsufficientFunds
	}

	_, err = d.pool.Exec(ctx,
		`UPDATE balance SET current = current - $1, withdrawn = withdrawn + $1 WHERE user_id = $2`,
		sum, userID)
	if err != nil {
		return err
	}

	_, err = d.pool.Exec(ctx,
		`INSERT INTO withdrawals (user_id, "order", sum) VALUES ($1, $2, $3)`,
		userID, order, sum)
	return err
}

func (d *DBStorage) GetWithdrawals(userID int) ([]Withdrawal, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := d.pool.Query(ctx,
		`SELECT "order", sum, processed_at FROM withdrawals WHERE user_id = $1 ORDER BY processed_at DESC`,
		userID)
	if err != nil {
		return nil, fmt.Errorf("query execution error: %w", err)
	}
	defer rows.Close()

	var withdrawals []Withdrawal
	for rows.Next() {
		var withdrawal Withdrawal
		if err := rows.Scan(&withdrawal.Order, &withdrawal.Sum, &withdrawal.ProcessedAt); err != nil {
			return nil, fmt.Errorf("data scan error: %w", err)
		}
		withdrawals = append(withdrawals, withdrawal)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("rows processing error: %w", err)
	}

	return withdrawals, nil
}

func (d *DBStorage) GetOrder(orderNum string) (Order, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var order Order

	err := d.pool.QueryRow(ctx,
		`SELECT number, status, accrual, uploaded_at FROM orders WHERE number = $1`,
		orderNum).Scan(&order.Number, &order.Status, &order.Accrual, &order.UploadedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Order{}, ErrNotFound
		}
		return Order{}, err
	}
	return order, nil
}

func (d *DBStorage) InitBalance(userID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := d.pool.Exec(ctx,
		`INSERT INTO balance (user_id, current, withdrawn) VALUES ($1, 0, 0) ON CONFLICT (user_id) DO NOTHING`,
		userID)
	return err
}

func (d *DBStorage) GetOrdersToUpdate() ([]OrderToUpdate, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := d.pool.Query(ctx,
		`SELECT id, number, user_id FROM orders WHERE status IN ('NEW', 'PROCESSING')`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []OrderToUpdate
	for rows.Next() {
		var order OrderToUpdate
		if err := rows.Scan(&order.ID, &order.Number, &order.UserID); err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, rows.Err()
}

func (d *DBStorage) UpdateOrderStatus(orderID int, status string, accrual float64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := d.pool.Exec(ctx,
		`UPDATE orders SET status = $1, accrual = $2 WHERE id = $3`,
		status, accrual, orderID)
	return err
}

func (d *DBStorage) UpdateBalance(userID int, amount float64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := d.pool.Exec(ctx,
		`UPDATE balance SET current = current + $1 WHERE user_id = $2`,
		amount, userID)
	return err
}
