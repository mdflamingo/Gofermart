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

type Order struct {
	Number     string
	Status     string
	Accrual    int
	UploadedAt time.Time
}

type Balance struct {
	Balance   float64
	Withdrawn int
}

type Withdrawal struct {
	Order       string
	Sum         int
	ProcessedAt time.Time
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

func (d *DBStorage) SaveUser(user models.UserDB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := d.pool.Exec(ctx,
		`INSERT INTO users (login, password)
         VALUES ($1, $2)`,
		user.Login, user.Password)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return ErrConflict
		}
		return fmt.Errorf("failed to save URL: %w", err)
	}

	return nil
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

	var returnedUserID int

	err := d.pool.QueryRow(ctx,
		`INSERT INTO orders (number, user_id)
         VALUES ($1, $2)
         ON CONFLICT (number)
         DO UPDATE SET number = EXCLUDED.number
         RETURNING user_id`,
		order, userID).Scan(&returnedUserID)

	if err != nil {
		return 0, fmt.Errorf("failed to save order number: %w", err)
	}

	if returnedUserID != userID {
		return returnedUserID, ErrConflict
	}

	return returnedUserID, nil
}

func (d *DBStorage) GetOrders(userID int) ([]Order, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := d.pool.Query(ctx,
		`SELECT number, status, accrual, uploaded_at FROM orders WHERE user_id = $1 ORDER BY uploaded_at ASC`,
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

	if len(orders) == 0 {
		return []Order{}, nil
	}

	return orders, nil
}

func (d *DBStorage) GetBalance(userID int) (Balance, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	var balance Balance

	err := d.pool.QueryRow(ctx,
		`SELECT balance, withdrawn FROM balance WHERE user_id = $1 ORDER BY uploaded_at ASC`,
		userID).Scan(&balance.Balance, &balance.Withdrawn)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Balance{}, ErrNotFound
		}
		return Balance{}, err
	}
	return balance, nil
}

func (d *DBStorage) SaveWithdrawn(userID int, withdrawn int, balance float64) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := d.pool.Exec(ctx,
		`INSERT INTO balance (user_id, balance, withdrawn)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id) DO UPDATE
		 SET balance = EXCLUDED.balance,
		     withdrawn = EXCLUDED.withdrawn,
		     uploaded_at = NOW()`,
		userID, balance, withdrawn)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (d *DBStorage) GetWithdrawals(userID int) ([]Withdrawal, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := d.pool.Query(
		ctx,
		`SELECT o.number, b.balance, b.uploaded_at
		FROM balance as b
		LEFT JOIN orders as o ON o.id = b.order_id
		WHERE b.user_id = $1
		ORDER BY b.uploaded_at ASC`,
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

	if len(withdrawals) == 0 {
		return []Withdrawal{}, nil
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
