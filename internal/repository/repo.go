package repository

import (
	"context"
	"cpamgt/internal/pkg/log"
	"time"

	"gorm.io/gorm"
)

type ContextKeyType struct{}

var ctxTxKey ContextKeyType = ContextKeyType{}

func GetContextKey() ContextKeyType {
	return ctxTxKey
}

type Repository struct {
	db     *gorm.DB
	logger *log.Logger
}

func NewRepository(
	logger *log.Logger,
	db *gorm.DB,
	// rdb *redis.Client,
	//
	//	mongo *mongo.Client,
) *Repository {
	return &Repository{
		db: db,
		//rdb:    rdb,
		//mongo:  mongo,
		logger: logger,
	}
}

type Transaction interface {
	Transaction(ctx context.Context, fn func(ctx context.Context) error) error
}

func NewTransaction(r *Repository) Transaction {
	return r
}

// DB return new gorm db Session or tx
// If you need to create a Transaction, you must call DB(ctx) and Transaction(ctx,fn)
func (r *Repository) DB(ctx context.Context) *gorm.DB {
	v := ctx.Value(ctxTxKey)
	if v != nil {
		if tx, ok := v.(*gorm.DB); ok {
			return tx
		}
	}
	// r.db.withContext(ctx) will create a new transaction if
	// the context is not a transaction.
	return r.db.WithContext(ctx)
}

func (r *Repository) Transaction(ctx context.Context, fn func(ctx context.Context) error) error {
	// diff: 对于嵌套事务，如果外部事务失败，内部事务也会失败
	return r.DB(ctx).Transaction(func(tx *gorm.DB) error {
		ctx = context.WithValue(ctx, ctxTxKey, tx)
		return fn(ctx)
	})
	// diff: 每次都是新建独立事务，如果最终外部事务失败，不会影响内部事务
	//return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
	//	ctx = context.WithValue(ctx, ctxTxKey, tx)
	//	return fn(ctx)
	//})
}

func TransactionExample() {
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	repo := NewRepository(nil, nil)

	err := repo.Transaction(ctx, func(ctx context.Context) error {
		db := repo.DB(ctx) // 如果存在事务, db 就是事务, 否则就是 db
		_ = db
		// do something with db
		// if error, return error
		// userRepo.DoSomething(ctx) -> will get db form ctx
		//
		return nil
	})
	if err != nil {
		panic(err)
	}
	// 嵌套事务 SavePoint
	err = repo.Transaction(ctx, func(ctx context.Context) error {
		db := repo.DB(ctx) // 如果存在事务, db 就是事务, 否则就是 db
		_ = db
		// do something with db
		// if error, return error
		// userRepo.DoSomething(ctx) -> will get db form ctx
		//
		err = repo.Transaction(ctx, func(ctx context.Context) error {
			// do something2
			return err
		})
		return err
		return nil
	})

}
