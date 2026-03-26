//go:build integration

package dbtest

import (
	"context"
	"fmt"
	"log"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"easybnk.gitlab.yandexcloud.net/backend/platform-core/pkg/db"
)

// --- Два репозитория ---

type userRepo struct {
	db.DbClient
}

func newUserRepo(c db.DbClient) *userRepo {
	return &userRepo{DbClient: c}
}

func (r *userRepo) Create(ctx context.Context, name string) error {
	_, err := r.Exec(ctx, "INSERT INTO users (name) VALUES ($1)", name)
	return err
}

func (r *userRepo) Count(ctx context.Context) (int, error) {
	var count int
	err := r.QueryRow(ctx, "SELECT count(*) FROM users").Scan(&count)
	return count, err
}

type orderRepo struct {
	db.DbClient
}

func newOrderRepo(c db.DbClient) *orderRepo {
	return &orderRepo{DbClient: c}
}

func (r *orderRepo) Create(ctx context.Context, userID int, amount int) error {
	_, err := r.Exec(ctx, "INSERT INTO orders (user_id, amount) VALUES ($1, $2)", userID, amount)
	return err
}

func (r *orderRepo) Count(ctx context.Context) (int, error) {
	var count int
	err := r.QueryRow(ctx, "SELECT count(*) FROM orders").Scan(&count)
	return count, err
}

// --- Сервис: два репозитория + DbClient ---

type orderService struct {
	db.DbClient
	users  *userRepo
	orders *orderRepo
}

func (s *orderService) CreateUserWithOrder(ctx context.Context, name string, amount int) error {
	return s.WithTransaction(ctx, func(txctx context.Context) error {
		if err := s.users.Create(txctx, name); err != nil {
			return fmt.Errorf("create user: %w", err)
		}
		if err := s.orders.Create(txctx, 1, amount); err != nil {
			return fmt.Errorf("create order: %w", err)
		}
		return nil
	})
}

func (s *orderService) CreateUserWithOrderAndFail(ctx context.Context, name string, amount int) error {
	return s.WithTransaction(ctx, func(txctx context.Context) error {
		if err := s.users.Create(txctx, name); err != nil {
			return fmt.Errorf("create user: %w", err)
		}
		if err := s.orders.Create(txctx, 1, amount); err != nil {
			return fmt.Errorf("create order: %w", err)
		}
		return fmt.Errorf("business logic error")
	})
}

// --- Инфра ---

func startPostgres(ctx context.Context) (*pgxpool.Pool, func(), error) {
	req := testcontainers.ContainerRequest{
		Image:        "postgres:17-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
			"POSTGRES_DB":       "test",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp"),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("start container: %w", err)
	}

	host, _ := container.Host(ctx)
	port, _ := container.MappedPort(ctx, "5432")

	dsn := fmt.Sprintf("postgres://test:test@%s:%s/test?sslmode=disable", host, port.Port())
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		container.Terminate(ctx) //nolint:errcheck
		return nil, nil, fmt.Errorf("connect: %w", err)
	}

	cleanup := func() {
		pool.Close()
		if err := container.Terminate(ctx); err != nil {
			log.Printf("terminate container: %v", err)
		}
	}

	return pool, cleanup, nil
}

func createTables(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE users (id SERIAL PRIMARY KEY, name TEXT NOT NULL);
		CREATE TABLE orders (id SERIAL PRIMARY KEY, user_id INT NOT NULL, amount INT NOT NULL);
	`)
	return err
}

// --- Тесты ---

func TestTransaction_TwoRepos_CommitOnSuccess(t *testing.T) {
	ctx := context.Background()
	pool, cleanup, err := startPostgres(ctx)
	require.NoError(t, err)
	defer cleanup()
	require.NoError(t, createTables(ctx, pool))

	client := db.NewDbClient(pool)
	svc := &orderService{
		users:    newUserRepo(client),
		orders:   newOrderRepo(client),
		DbClient: client,
	}

	err = svc.CreateUserWithOrder(ctx, "Alice", 100)
	require.NoError(t, err)

	userCount, err := svc.users.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, userCount)

	orderCount, err := svc.orders.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, orderCount)
}

func TestTransaction_TwoRepos_RollbackOnError(t *testing.T) {
	ctx := context.Background()
	pool, cleanup, err := startPostgres(ctx)
	require.NoError(t, err)
	defer cleanup()
	require.NoError(t, createTables(ctx, pool))

	client := db.NewDbClient(pool)
	svc := &orderService{
		users:    newUserRepo(client),
		orders:   newOrderRepo(client),
		DbClient: client,
	}

	err = svc.CreateUserWithOrderAndFail(ctx, "Alice", 100)
	require.Error(t, err)
	require.Contains(t, err.Error(), "business logic error")

	userCount, err := svc.users.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, userCount, "user должен откатиться")

	orderCount, err := svc.orders.Count(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, orderCount, "order должен откатиться")
}

func TestTransaction_NestedTransactionError(t *testing.T) {
	ctx := context.Background()
	pool, cleanup, err := startPostgres(ctx)
	require.NoError(t, err)
	defer cleanup()

	client := db.NewDbClient(pool)

	err = client.WithTransaction(ctx, func(txctx context.Context) error {
		return client.WithTransaction(txctx, func(_ context.Context) error {
			return nil
		})
	})
	require.ErrorIs(t, err, db.ErrNestedTransaction)
}
