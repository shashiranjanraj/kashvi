package orm

import (
	"time"

	"github.com/shashiranjanraj/kashvi/pkg/cache"
	"github.com/shashiranjanraj/kashvi/pkg/database"
	"gorm.io/gorm"
)

type Query struct {
	db *gorm.DB
}

func DB() *Query {
	return &Query{db: database.DB}
}

func (q *Query) Model(v interface{}) *Query {
	return &Query{db: q.db.Model(v)}
}

func (q *Query) Where(query string, args ...interface{}) *Query {
	return &Query{db: q.db.Where(query, args...)}
}

func (q *Query) Get(dest interface{}) error {
	return q.db.Find(dest).Error
}

func (q *Query) First(dest interface{}) error {
	return q.db.First(dest).Error
}

func (q *Query) Cache(key string, ttl time.Duration, dest interface{}) error {
	if cache.Get(key, dest) {
		return nil
	}

	err := q.db.Find(dest).Error
	if err != nil {
		return err
	}

	cache.Set(key, dest, ttl)
	return nil
}
