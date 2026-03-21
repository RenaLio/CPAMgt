package repository

import (
	"context"
	"cpamgt/internal/config"
	"cpamgt/pkg/log"
	"fmt"

	"time"

	"github.com/glebarez/sqlite"
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewDB(conf *config.Config, logger *log.Logger) *gorm.DB {
	var (
		db  *gorm.DB
		err error
	)

	driver := conf.Data.DB.User.Driver
	dsn := conf.Data.DB.User.DSN

	// GORM doc: https://gorm.io/docs/connecting_to_the_database.html
	switch driver {
	case "mysql":
		db, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
			//Logger: logger,
		})
	case "postgres":
		db, err = gorm.Open(postgres.New(postgres.Config{
			DSN:                  dsn,
			PreferSimpleProtocol: true, // disables implicit prepared statement usage
		}), &gorm.Config{})
	case "sqlite":
		db, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	default:
		panic("unknown db driver")
	}
	if err != nil {
		panic(err)
	}
	db = db.Debug()

	// Connection Pool config
	sqlDB, err := db.DB()
	if err != nil {
		panic(err)
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)
	return db
}

func NewRedis(conf *viper.Viper) *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     conf.GetString("data.redis.addr"),
		Password: conf.GetString("data.redis.password"),
		DB:       conf.GetInt("data.redis.db"),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		panic(fmt.Sprintf("redis error: %s", err.Error()))
	}

	return rdb
}

//func NewMongo(conf *viper.Viper) (*mongo.Client, func(), error) {
//	// https://www.mongodb.com/zh-cn/docs/drivers/go/current/
//	uri := conf.GetString("data.mongo.uri")
//	client, err := mongo.Connect(context.TODO(), options.Client().
//		ApplyURI(uri))
//	if err != nil {
//		panic(fmt.Sprintf("mongo client error: %s", err.Error()))
//	}
//
//	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
//	defer cancel()
//
//	err = client.Ping(ctx, nil)
//	if err != nil {
//		panic(fmt.Sprintf("mongo ping error: %s", err.Error()))
//	}
//
//	return client, func() {
//		err = client.Disconnect(ctx)
//		if err != nil {
//			panic(fmt.Sprintf("mongo disconnect error: %s", err.Error()))
//		}
//	}, err
//}
