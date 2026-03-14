package orm

import (
	"errors"
	"sync"
	"time"

	"github.com/shashiranjanraj/kashvi/pkg/database"
	"github.com/shashiranjanraj/kashvi/pkg/logger"
	"gorm.io/gorm"
)

// Query is a chainable, immutable query builder wrapping gorm.DB.
type Query struct {
	db           *gorm.DB
	Error        error
	RowsAffected int64
}

// Pagination holds metadata for a paginated response.
type Pagination struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
	HasNext    bool  `json:"has_next"`
	HasPrev    bool  `json:"has_prev"`
}

// DB returns a fresh Query backed by the global database connection.
func DB() *Query {
	if database.DB == nil {
		return &Query{Error: errors.New("database connection not initialized")}
	}
	return &Query{db: database.DB}
}

// String implements fmt.Stringer to ensure the query doesn't print as a pointer address.
func (q *Query) String() string {
	if q.Error != nil {
		return q.Error.Error()
	}
	return "orm.Query"
}

// Model sets the model for the query (table resolution).
func (q *Query) Model(v interface{}) *Query {
	if q.db == nil || q.Error != nil {
		return q
	}
	db := q.db.Model(v)
	return &Query{db: db, Error: db.Error}
}

// Where appends a WHERE clause.
func (q *Query) Where(query string, args ...interface{}) *Query {
	if q.db == nil || q.Error != nil {
		return q
	}
	db := q.db.Where(query, args...)
	return &Query{db: db, Error: db.Error}
}

// OrderBy appends an ORDER BY clause. dir should be "asc" or "desc".
func (q *Query) OrderBy(col, dir string) *Query {
	if q.db == nil || q.Error != nil {
		return q
	}
	db := q.db.Order(col + " " + dir)
	return &Query{db: db, Error: db.Error}
}

// Select limits the fetched columns.
func (q *Query) Select(fields ...string) *Query {
	if q.db == nil || q.Error != nil {
		return q
	}
	args := make([]interface{}, len(fields)-1)
	for i, f := range fields[1:] {
		args[i] = f
	}
	db := q.db.Select(fields[0], args...)
	return &Query{db: db, Error: db.Error}
}

// Joins adds a JOIN clause.
func (q *Query) Joins(query string, args ...interface{}) *Query {
	if q.db == nil || q.Error != nil {
		return q
	}
	db := q.db.Joins(query, args...)
	return &Query{db: db, Error: db.Error}
}

// With eager-loads the named association (GORM Preload).
func (q *Query) With(assoc string) *Query {
	if q.db == nil || q.Error != nil {
		return q
	}
	db := q.db.Preload(assoc)
	return &Query{db: db, Error: db.Error}
}

// Paginate applies OFFSET/LIMIT for page-based pagination.
func (q *Query) Paginate(page, limit int) *Query {
	if q.db == nil || q.Error != nil {
		return q
	}
	page, limit = normalizePagination(page, limit)
	offset := (page - 1) * limit
	db := q.db.Offset(offset).Limit(limit)
	return &Query{db: db, Error: db.Error}
}

// ---------- Read ----------

// Get fetches all matching rows into dest.
func (q *Query) Get(dest interface{}) error {
	if q.Error != nil {
		return q.Error
	}
	if q.db == nil {
		return errors.New("database connection not initialized")
	}
	return q.db.Find(dest).Error
}

// First fetches the first matching row into dest.
func (q *Query) First(dest interface{}) error {
	if q.Error != nil {
		return q.Error
	}
	if q.db == nil {
		return errors.New("database connection not initialized")
	}
	return q.db.First(dest).Error
}

// GetWithPagination fetches rows with pagination metadata.
func (q *Query) GetWithPagination(dest interface{}, page, limit int) (Pagination, error) {
	if q.Error != nil {
		return Pagination{}, q.Error
	}
	if q.db == nil {
		return Pagination{}, errors.New("database connection not initialized")
	}
	page, limit = normalizePagination(page, limit)

	var total int64
	if err := q.db.Count(&total).Error; err != nil {
		return Pagination{}, err
	}

	if err := q.Paginate(page, limit).Get(dest); err != nil {
		return Pagination{}, err
	}

	totalPages := int((total + int64(limit) - 1) / int64(limit))

	return Pagination{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}, nil
}

// Cache tries the cache first; on miss it executes the query and stores the result.
func (q *Query) Cache(key string, ttl time.Duration, dest interface{}) error {
	if q.Error != nil {
		return q.Error
	}
	if q.db == nil {
		return errors.New("database connection not initialized")
	}
	// Import-cycle-safe: import cache inline only through the registered interface.
	// Direct cache use is done via the CacheStore variable below (set at boot).
	if CacheStore != nil && CacheStore.Get(key, dest) {
		logger.Debug("orm: cache hit", "key", key)
		return nil
	}
	logger.Debug("orm: cache miss", "key", key)

	if err := q.db.Find(dest).Error; err != nil {
		return err
	}

	if CacheStore != nil {
		CacheStore.Set(key, dest, ttl)
	}
	return nil
}

// ---------- Write ----------

// Create inserts value into the database.
func (q *Query) Create(value interface{}) error {
	if q.db == nil || q.Error != nil {
		return q.Error
	}
	return q.db.Create(value).Error
}

// Save upserts value (creates if no primary key, updates otherwise).
func (q *Query) Save(value interface{}) error {
	if q.db == nil || q.Error != nil {
		return q.Error
	}
	return q.db.Save(value).Error
}

// Update sets a single column to value on the current query scope.
func (q *Query) Update(col string, value interface{}) error {
	if q.db == nil || q.Error != nil {
		return q.Error
	}
	return q.db.Update(col, value).Error
}

// Updates sets multiple columns from a map or struct.
func (q *Query) Updates(values interface{}) error {
	if q.db == nil || q.Error != nil {
		return q.Error
	}
	return q.db.Updates(values).Error
}

// Delete soft-deletes (or hard-deletes if no DeletedAt field) matching rows.
func (q *Query) Delete(value interface{}, conds ...interface{}) error {
	if q.db == nil || q.Error != nil {
		return q.Error
	}
	return q.db.Delete(value, conds...).Error
}

// ---------- Parallel ----------

// ParallelFunc is a query task that returns an error.
type ParallelFunc func() error

// Parallel runs all provided query functions concurrently and returns the first
// non-nil error encountered (all are still waited for).
func Parallel(fns ...ParallelFunc) error {
	var (
		wg    sync.WaitGroup
		mu    sync.Mutex
		first error
	)

	for _, fn := range fns {
		wg.Add(1)
		go func(f ParallelFunc) {
			defer wg.Done()
			if err := f(); err != nil {
				mu.Lock()
				if first == nil {
					first = err
				}
				mu.Unlock()
			}
		}(fn)
	}

	wg.Wait()
	return first
}

// ---------- helpers ----------

func normalizePagination(page, limit int) (int, int) {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}
	return page, limit
}

// ---------- Cache bridge (breaks import cycle) ----------

// Cacher is a minimal interface for the cache layer, so orm does not directly
// import pkg/cache (which would create a cycle via pkg/database).
type Cacher interface {
	Get(key string, dest interface{}) bool
	Set(key string, value interface{}, ttl time.Duration) error
}

// CacheStore is set at boot time (e.g. in internal/kernel/http.go) to wire up
// the real Redis cache without creating an import cycle.
var CacheStore Cacher
