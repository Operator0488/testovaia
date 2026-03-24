package db

import "context"

// BaseRepository — базовая структура для репозиториев.
// Хранит пул соединений и при каждом запросе проверяет контекст
// на наличие транзакции.
//
// Использование:
//
//	type UserRepository struct {
//	    db.BaseRepository
//	}
//
//	func NewUserRepository(q db.Querier) *UserRepository {
//	    return &UserRepository{BaseRepository: db.NewBaseRepository(q)}
//	}
//
//	func (r *UserRepository) Create(ctx context.Context, user *User) error {
//	    _, err := r.Querier(ctx).Exec(ctx, "INSERT INTO users (name) VALUES ($1)", user.Name)
//	    return err
//	}
type BaseRepository struct {
	pool Querier
}

// NewBaseRepository создаёт BaseRepository с указанным Querier (обычно — пул соединений).
func NewBaseRepository(q Querier) BaseRepository {
	return BaseRepository{pool: q}
}

// Querier возвращает транзакцию из контекста, если она есть, иначе — пул.
func (r *BaseRepository) Querier(ctx context.Context) Querier {
	if tx := TxFromContext(ctx); tx != nil {
		return tx
	}
	return r.pool
}
