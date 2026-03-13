package logger

import (
	"context"
	"fmt"
	"time"

	"github.com/shashiranjanraj/kashvi/config"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap/zapcore"
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

// MongoZapCore is a zapcore.Core that writes to MongoDB asynchronously.
type MongoZapCore struct {
	zapcore.LevelEnabler
	col    *mongo.Collection
	client *mongo.Client
	queue  chan LogDocument
	done   chan struct{}
	fields []zapcore.Field
}

var globalZapMongoCore *MongoZapCore

// NewMongoZapCore creates a generic zap core connecting to MongoDB.
func NewMongoZapCore(levelEnabler zapcore.LevelEnabler) (zapcore.Core, error) {
	uri := config.MongoURI()
	if uri == "" {
		return nil, fmt.Errorf("mongo logging disabled")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOpts := options.Client().ApplyURI(uri).
		SetConnectTimeout(5 * time.Second).
		SetServerSelectionTimeout(5 * time.Second).
		SetMaxPoolSize(10)

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("mongo_zap: connect: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("mongo_zap: ping: %w", err)
	}

	col := client.Database(config.MongoLogDB()).Collection(config.MongoLogCollection())

	// Create time-based index
	_, _ = col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "time", Value: -1}},
		Options: options.Index().SetBackground(true),
	})

	core := &MongoZapCore{
		LevelEnabler: levelEnabler,
		col:          col,
		client:       client,
		queue:        make(chan LogDocument, mongoQueueSize),
		done:         make(chan struct{}),
	}
	globalZapMongoCore = core

	go core.drainLoop()
	return core, nil
}

func (c *MongoZapCore) With(fields []zapcore.Field) zapcore.Core {
	clone := &MongoZapCore{
		LevelEnabler: c.LevelEnabler,
		col:          c.col,
		client:       c.client,
		queue:        c.queue,
		done:         c.done,
		fields:       make([]zapcore.Field, len(c.fields)+len(fields)),
	}
	copy(clone.fields, c.fields)
	copy(clone.fields[len(c.fields):], fields)
	return clone
}

func (c *MongoZapCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

func (c *MongoZapCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	doc := LogDocument{
		Time:  ent.Time,
		Level: ent.Level.String(),
		Msg:   ent.Message,
		Attrs: bson.M{},
	}

	if ent.Caller.Defined {
		doc.Source = ent.Caller.TrimmedPath()
	}

	// Helper to extract fields
	allFields := make([]zapcore.Field, 0, len(c.fields)+len(fields))
	allFields = append(allFields, c.fields...)
	allFields = append(allFields, fields...)

	enc := zapcore.NewMapObjectEncoder()
	for _, f := range allFields {
		f.AddTo(enc)
	}

	for k, v := range enc.Fields {
		if k == "request_id" {
			if s, ok := v.(string); ok {
				doc.RequestID = s
				continue
			}
		}
		doc.Attrs[k] = v
	}

	select {
	case c.queue <- doc:
	default:
	}
	return nil
}

func (c *MongoZapCore) Sync() error {
	return nil
}

func (c *MongoZapCore) drainLoop() {
	ticker := time.NewTicker(mongoDrainTick)
	defer ticker.Stop()

	batch := make([]interface{}, 0, mongoBatchSize)

	flush := func() {
		if len(batch) == 0 {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = c.col.InsertMany(ctx, batch)
		batch = batch[:0]
	}

	for {
		select {
		case doc := <-c.queue:
			batch = append(batch, doc)
			if len(batch) >= mongoBatchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-c.done:
			for len(c.queue) > 0 {
				batch = append(batch, <-c.queue)
			}
			flush()
			return
		}
	}
}

// CloseMongoHandler flushes pending logs and disconnects from MongoDB gracefully.
func CloseMongoHandler() {
	if globalZapMongoCore != nil {
		select {
		case <-globalZapMongoCore.done:
		default:
			close(globalZapMongoCore.done)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = globalZapMongoCore.client.Disconnect(ctx)
	}
}
