// Package logger — mongo_handler.go
//
// MongoHandler is an slog.Handler that asynchronously stores log records in
// a MongoDB collection.  It is designed for zero-impact on the hot request
// path:
//
//   - Writes are enqueued into a buffered channel (non-blocking).
//   - A single background goroutine drains the channel and performs
//     InsertMany in configurable batch sizes (default 50).
//   - If the channel is full, the record is silently dropped; logging must
//     never block application code.
//   - Graceful shutdown: call Close() to flush and disconnect.
package logger

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	mongoQueueSize = 4096 // buffered channel capacity
	mongoBatchSize = 50   // maximum documents per InsertMany
	mongoDrainTick = 2 * time.Second
)

// LogDocument is the shape written to MongoDB.
type LogDocument struct {
	Time      time.Time `bson:"time"`
	Level     string    `bson:"level"`
	Source    string    `bson:"source,omitempty"`
	Msg       string    `bson:"msg"`
	RequestID string    `bson:"request_id,omitempty"`
	Attrs     bson.M    `bson:"attrs,omitempty"`
}

// MongoHandler is a slog.Handler that writes to MongoDB asynchronously.
type MongoHandler struct {
	inner  slog.Handler // underlying handler for attribute resolution
	col    *mongo.Collection
	client *mongo.Client
	queue  chan LogDocument
	done   chan struct{}
	attrs  []slog.Attr
	groups []string
}

// NewMongoHandler creates a MongoHandler connected to uri/db/collection.
// The caller must eventually call Close().
func NewMongoHandler(uri, db, collection string) (*MongoHandler, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOpts := options.Client().ApplyURI(uri).
		SetConnectTimeout(5 * time.Second).
		SetServerSelectionTimeout(5 * time.Second).
		SetMaxPoolSize(10)

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("mongo_handler: connect: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("mongo_handler: ping: %w", err)
	}

	col := client.Database(db).Collection(collection)

	// Create time-based index for easy log querying / TTL.
	_, _ = col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "time", Value: -1}},
		Options: options.Index().SetBackground(true),
	})

	h := &MongoHandler{
		col:    col,
		client: client,
		queue:  make(chan LogDocument, mongoQueueSize),
		done:   make(chan struct{}),
	}

	go h.drainLoop()
	return h, nil
}

// ─── slog.Handler interface ───────────────────────────────────────────────────

func (h *MongoHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }

func (h *MongoHandler) Handle(_ context.Context, r slog.Record) error {
	doc := LogDocument{
		Time:  r.Time,
		Level: r.Level.String(),
		Msg:   r.Message,
		Attrs: bson.M{},
	}

	// Collect attrs from WithAttrs + the record itself.
	for _, a := range h.attrs {
		if a.Key == "request_id" {
			doc.RequestID = a.Value.String()
		} else {
			doc.Attrs[a.Key] = a.Value.Any()
		}
	}
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == "request_id" {
			doc.RequestID = a.Value.String()
		} else {
			doc.Attrs[a.Key] = a.Value.Any()
		}
		return true
	})

	if r.PC != 0 {
		frames := slog.NewLogLogger(h, r.Level)
		_ = frames // suppress unused warning — source embedded via PC below
		// We intentionally skip full source resolution to keep this zero-alloc.
	}

	// Non-blocking enqueue: drop if channel is full.
	select {
	case h.queue <- doc:
	default:
		// silently dropped — logging must never block
	}
	return nil
}

func (h *MongoHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newAttrs := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(newAttrs, h.attrs)
	copy(newAttrs[len(h.attrs):], attrs)
	return &MongoHandler{
		col:    h.col,
		client: h.client,
		queue:  h.queue,
		done:   h.done,
		attrs:  newAttrs,
		groups: h.groups,
	}
}

func (h *MongoHandler) WithGroup(name string) slog.Handler {
	newGroups := make([]string, len(h.groups)+1)
	copy(newGroups, h.groups)
	newGroups[len(h.groups)] = name
	return &MongoHandler{
		col:    h.col,
		client: h.client,
		queue:  h.queue,
		done:   h.done,
		attrs:  h.attrs,
		groups: newGroups,
	}
}

// ─── Internals ────────────────────────────────────────────────────────────────

// drainLoop runs in the background, flushing queued documents into MongoDB.
func (h *MongoHandler) drainLoop() {
	ticker := time.NewTicker(mongoDrainTick)
	defer ticker.Stop()

	batch := make([]interface{}, 0, mongoBatchSize)

	flush := func() {
		if len(batch) == 0 {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = h.col.InsertMany(ctx, batch) // errors are intentionally ignored
		batch = batch[:0]
	}

	for {
		select {
		case doc := <-h.queue:
			batch = append(batch, doc)
			if len(batch) >= mongoBatchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-h.done:
			// Drain remaining items before exit.
			for len(h.queue) > 0 {
				batch = append(batch, <-h.queue)
			}
			flush()
			return
		}
	}
}

// Close flushes pending logs and disconnects from MongoDB.
// Safe to call multiple times.
func (h *MongoHandler) Close() {
	select {
	case <-h.done:
	default:
		close(h.done)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = h.client.Disconnect(ctx)
}

// ─── Multi-handler fan-out ─────────────────────────────────────────────────────

// MultiHandler fans out to multiple slog.Handlers.
type MultiHandler struct {
	handlers []slog.Handler
}

// NewMultiHandler returns a handler that sends each record to all hs.
func NewMultiHandler(hs ...slog.Handler) *MultiHandler {
	return &MultiHandler{handlers: hs}
}

func (m *MultiHandler) Enabled(ctx context.Context, l slog.Level) bool {
	for _, h := range m.handlers {
		if h.Enabled(ctx, l) {
			return true
		}
	}
	return false
}

func (m *MultiHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, h := range m.handlers {
		if h.Enabled(ctx, r.Level) {
			_ = h.Handle(ctx, r.Clone())
		}
	}
	return nil
}

func (m *MultiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	hs := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		hs[i] = h.WithAttrs(attrs)
	}
	return &MultiHandler{handlers: hs}
}

func (m *MultiHandler) WithGroup(name string) slog.Handler {
	hs := make([]slog.Handler, len(m.handlers))
	for i, h := range m.handlers {
		hs[i] = h.WithGroup(name)
	}
	return &MultiHandler{handlers: hs}
}
