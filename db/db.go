package db

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
)

type Store struct {
	DB  *sql.DB
	RDB *redis.Client
}

func NewStore(dbURL, redisURL string) *Store {
	var db *sql.DB
	if dbURL != "" {
		if d, err := sql.Open("postgres", dbURL); err == nil {
			db = d
		}
	}

	var rdb *redis.Client
	if redisURL != "" {
		opts, err := redis.ParseURL(redisURL)
		if err != nil {
			opts = &redis.Options{
				Addr: redisURL,
			}
		}
		rdb = redis.NewClient(opts)
	}

	return &Store{
		DB:  db,
		RDB: rdb,
	}
}

func (s *Store) PingPostgres() bool {
	if s.DB == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return s.DB.PingContext(ctx) == nil
}

func (s *Store) PingRedis() bool {
	if s.RDB == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return s.RDB.Ping(ctx).Err() == nil
}
