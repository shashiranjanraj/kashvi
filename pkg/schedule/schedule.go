// Package schedule provides a cron-style task scheduler for Kashvi.
//
// Usage:
//
//	schedule.EveryMinute().Run(func() { log.Println("tick") })
//	schedule.Every(5).Minutes().Run(syncData)
//	schedule.Daily().At("03:00").Run(backupDB)
//	schedule.Cron("0 * * * *").Run(myTask)
//
//	// Start the scheduler in the background (call once at boot):
//	schedule.Start(ctx)
package schedule

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/shashiranjanraj/kashvi/pkg/logger"
)

// Task is the function signature for a scheduled task.
type Task func()

// entry represents a single scheduled job.
type entry struct {
	id         string
	interval   time.Duration
	cronExpr   string // "" unless using Cron()
	task       Task
	lastRun    time.Time
	running    bool // overlap guard
	noOverlap  bool
	beforeHook Task
	afterHook  Task
	mu         sync.Mutex
}

// Schedule is a fluent builder for a single entry before it is registered.
type Schedule struct {
	e *entry
}

// ------------------- Registry -------------------

var (
	regMu   sync.Mutex
	entries []*entry
)

// EveryMinute schedules the task to run every 60 seconds.
func EveryMinute() *Schedule { return Every(1).Minutes() }

// Every starts a fluent builder with n units.
func Every(n int) *freqBuilder { return &freqBuilder{n: n} }

// Hourly schedules the task to run every hour.
func Hourly() *Schedule { return Every(1).Hours() }

// Daily schedules the task to run every 24 hours.
func Daily() *Schedule { return Every(24).Hours() }

// Weekly schedules the task to run every 7 days.
func Weekly() *Schedule { return Every(7).Days() }

// Cron schedules using a 5-field cron expression (min hour dom mon dow).
// Full cron parsing is done inline to keep dependencies at zero.
func Cron(expr string) *Schedule {
	e := &entry{cronExpr: expr, noOverlap: false}
	return &Schedule{e: e}
}

// ------------------- Fluent frequency builder -------------------

type freqBuilder struct{ n int }

func (f *freqBuilder) Seconds() *Schedule {
	return &Schedule{e: &entry{interval: time.Duration(f.n) * time.Second}}
}
func (f *freqBuilder) Minutes() *Schedule {
	return &Schedule{e: &entry{interval: time.Duration(f.n) * time.Minute}}
}
func (f *freqBuilder) Hours() *Schedule {
	return &Schedule{e: &entry{interval: time.Duration(f.n) * time.Hour}}
}
func (f *freqBuilder) Days() *Schedule {
	return &Schedule{e: &entry{interval: time.Duration(f.n) * 24 * time.Hour}}
}

// ------------------- Schedule chainable options -------------------

// WithoutOverlapping prevents a new run if the previous one is still executing.
func (s *Schedule) WithoutOverlapping() *Schedule {
	s.e.noOverlap = true
	return s
}

// Before registers a hook that fires before the task.
func (s *Schedule) Before(fn Task) *Schedule {
	s.e.beforeHook = fn
	return s
}

// After registers a hook that fires after the task (always, even on panic).
func (s *Schedule) After(fn Task) *Schedule {
	s.e.afterHook = fn
	return s
}

// Name gives the entry a human-readable identifier for logging.
func (s *Schedule) Name(id string) *Schedule {
	s.e.id = id
	return s
}

// Run registers the task and adds it to the global scheduler registry.
// Call Start() to begin dispatching.
func (s *Schedule) Run(fn Task) {
	s.e.task = fn
	if s.e.id == "" {
		s.e.id = fmt.Sprintf("task-%d", len(entries)+1)
	}
	regMu.Lock()
	entries = append(entries, s.e)
	regMu.Unlock()
}

// ------------------- Scheduler loop -------------------

// Start begins the scheduler loop in the background.
// It ticks every second and dispatches due tasks.
// Call before any tasks are registered to ensure none are missed.
func Start(ctx context.Context) {
	go run(ctx)
	logger.Info("schedule: scheduler started")
}

func run(ctx context.Context) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("schedule: scheduler stopped")
			return
		case now := <-ticker.C:
			regMu.Lock()
			current := make([]*entry, len(entries))
			copy(current, entries)
			regMu.Unlock()

			for _, e := range current {
				if isDue(e, now) {
					dispatch(e)
				}
			}
		}
	}
}

func isDue(e *entry, now time.Time) bool {
	if e.cronExpr != "" {
		return matchCron(e.cronExpr, now)
	}
	if e.lastRun.IsZero() {
		return true // first run
	}
	return now.Sub(e.lastRun) >= e.interval
}

func dispatch(e *entry) {
	e.mu.Lock()
	if e.noOverlap && e.running {
		e.mu.Unlock()
		logger.Warn("schedule: skipping overlapping task", "id", e.id)
		return
	}
	e.running = true
	e.lastRun = time.Now()
	e.mu.Unlock()

	go func() {
		defer func() {
			e.mu.Lock()
			e.running = false
			e.mu.Unlock()
			if r := recover(); r != nil {
				logger.Error("schedule: task panicked", "id", e.id, "panic", r)
			}
			if e.afterHook != nil {
				e.afterHook()
			}
		}()

		if e.beforeHook != nil {
			e.beforeHook()
		}
		logger.Info("schedule: running task", "id", e.id)
		e.task()
	}()
}

// ------------------- Minimal cron parser -------------------
// Supports 5-field cron: minute hour dom month dow
// Each field: * | number | */step | number-number

func matchCron(expr string, t time.Time) bool {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return false
	}
	checks := []struct {
		field string
		val   int
	}{
		{fields[0], t.Minute()},
		{fields[1], t.Hour()},
		{fields[2], t.Day()},
		{fields[3], int(t.Month())},
		{fields[4], int(t.Weekday())},
	}
	for _, c := range checks {
		if !matchField(c.field, c.val) {
			return false
		}
	}
	return true
}

func matchField(field string, val int) bool {
	if field == "*" {
		return true
	}
	// */step
	if strings.HasPrefix(field, "*/") {
		var step int
		fmt.Sscanf(field[2:], "%d", &step)
		return step > 0 && val%step == 0
	}
	// range a-b
	if strings.Contains(field, "-") {
		var lo, hi int
		fmt.Sscanf(field, "%d-%d", &lo, &hi)
		return val >= lo && val <= hi
	}
	// exact
	var n int
	fmt.Sscanf(field, "%d", &n)
	return n == val
}

// List returns all currently registered scheduled entries (for CLI display).
func List() []string {
	regMu.Lock()
	defer regMu.Unlock()
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		freq := e.cronExpr
		if freq == "" {
			freq = e.interval.String()
		}
		out = append(out, fmt.Sprintf("%s  [%s]", e.id, freq))
	}
	return out
}
