package queue

import (
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// FailedJobRecord is the GORM model persisted to the database.
// Auto-migrated by the HTTP kernel at startup.
type FailedJobRecord struct {
	ID       uint      `gorm:"primaryKey;autoIncrement"`
	JobType  string    `gorm:"size:255;not null;index"`
	Payload  string    `gorm:"type:text;not null"`
	Error    string    `gorm:"type:text"`
	Attempts int       `gorm:"not null;default:0"`
	FailedAt time.Time `gorm:"autoCreateTime"`
}

func (FailedJobRecord) TableName() string { return "kashvi_failed_jobs" }

// failedJobStore is the optional DB backend for persisting failed jobs.
// Set via UseDB() — nil means in-memory only.
var failedJobDB *gorm.DB

// UseDB configures the queue to persist failed jobs to the database.
// Call once at boot (e.g. after database.Connect()):
//
//	queue.UseDB(database.DB)
func UseDB(db *gorm.DB) {
	failedJobDB = db
	// Auto-create the table if it doesn't exist.
	db.AutoMigrate(&FailedJobRecord{})
}

// persistFailed writes a failed job record to the database (if configured)
// and also appends to the in-memory slice as a fallback.
func (m *Manager) persistFailed(job Job, typeName string, lastErr error, attempts int) {
	// Always append to in-memory slice.
	m.mu.Lock()
	m.failed = append(m.failed, FailedJob{
		Job: job, Err: lastErr, FailedAt: time.Now(), Attempts: attempts,
	})
	m.mu.Unlock()

	// Persist to DB if available.
	if failedJobDB == nil {
		return
	}

	payload, err := json.Marshal(job)
	if err != nil {
		payload = []byte(fmt.Sprintf(`{"error": "could not marshal: %v"}`, err))
	}

	record := FailedJobRecord{
		JobType:  typeName,
		Payload:  string(payload),
		Error:    lastErr.Error(),
		Attempts: attempts,
		FailedAt: time.Now(),
	}

	if err := failedJobDB.Create(&record).Error; err != nil {
		// Non-fatal — the in-memory slice still has it.
		// logger is not imported here to avoid import cycle, use fmt.
		fmt.Printf("queue: failed to persist failed job %s: %v\n", typeName, err)
	}
}
