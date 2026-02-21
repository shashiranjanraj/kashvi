package orm

import (
	"sync"
	"time"

	"github.com/shashiranjanraj/kashvi/pkg/database"
	"gorm.io/gorm"
)

// Query is a chainable, immutable query builder wrapping gorm.DB.
type Query struct {
	db *gorm.DB
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
	return &Query{db: database.DB}
}

// Model sets the model for the query (table resolution).
func (q *Query) Model(v interface{}) *Query {
	return &Query{db: q.db.Model(v)}
}

// Where appends a WHERE clause.
func (q *Query) Where(query string, args ...interface{}) *Query {
	return &Query{db: q.db.Where(query, args...)}
}

// OrderBy appends an ORDER BY clause. dir should be "asc" or "desc".
func (q *Query) OrderBy(col, dir string) *Query {
	return &Query{db: q.db.Order(col + " " + dir)}
}

// Select limits the fetched columns.
func (q *Query) Select(fields ...string) *Query {
	args := make([]interface{}, len(fields)-1)
	for i, f := range fields[1:] {
		args[i] = f
	}
	return &Query{db: q.db.Select(fields[0], args...)}
}

// Joins adds a JOIN clause.
func (q *Query) Joins(query string, args ...interface{}) *Query {
	return &Query{db: q.db.Joins(query, args...)}
}

// With eager-loads the named association (GORM Preload).
func (q *Query) With(assoc string) *Query {
	return &Query{db: q.db.Preload(assoc)}
}

// Paginate applies OFFSET/LIMIT for page-based pagination.
func (q *Query) Paginate(page, limit int) *Query {
	page, limit = normalizePagination(page, limit)
	offset := (page - 1) * limit
	return &Query{db: q.db.Offset(offset).Limit(limit)}
}

// ---------- Read ----------

// Get fetches all matching rows into dest.
func (q *Query) Get(dest interface{}) error {
	return q.db.Find(dest).Error
}

// First fetches the first matching row into dest.
func (q *Query) First(dest interface{}) error {
	return q.db.First(dest).Error
}

// GetWithPagination fetches rows with pagination metadata.
func (q *Query) GetWithPagination(dest interface{}, page, limit int) (Pagination, error) {
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
	// Import-cycle-safe: import cache inline only through the registered interface.
	// Direct cache use is done via the CacheStore variable below (set at boot).
	if CacheStore != nil && CacheStore.Get(key, dest) {
		return nil
	}

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
	return q.db.Create(value).Error
}

// Save upserts value (creates if no primary key, updates otherwise).
func (q *Query) Save(value interface{}) error {
	return q.db.Save(value).Error
}

// Update sets a single column to value on the current query scope.
func (q *Query) Update(col string, value interface{}) error {
	return q.db.Update(col, value).Error
}

// Updates sets multiple columns from a map or struct.
func (q *Query) Updates(values interface{}) error {
	return q.db.Updates(values).Error
}

// Delete soft-deletes (or hard-deletes if no DeletedAt field) matching rows.
func (q *Query) Delete(value interface{}, conds ...interface{}) error {
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
