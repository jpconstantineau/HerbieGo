package app

import (
	"sync"

	"github.com/jpconstantineau/herbiego/internal/ports"
)

const defaultDebugLogMaxSize = 200

// DebugLog records AI provider request/response exchanges in a thread-safe
// ring buffer for live inspection in the TUI debug workspace.
type DebugLog struct {
	mu      sync.RWMutex
	records []ports.AICallRecord
	maxSize int
}

// NewDebugLog creates an empty debug log that retains up to maxSize records.
func NewDebugLog(maxSize int) *DebugLog {
	if maxSize <= 0 {
		maxSize = defaultDebugLogMaxSize
	}
	return &DebugLog{maxSize: maxSize}
}

// Append adds a call record, evicting the oldest entry when the log is full.
func (d *DebugLog) Append(record ports.AICallRecord) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.records = append(d.records, record)
	if len(d.records) > d.maxSize {
		d.records = d.records[len(d.records)-d.maxSize:]
	}
}

// Records returns a snapshot of all retained call records in append order.
func (d *DebugLog) Records() []ports.AICallRecord {
	d.mu.RLock()
	defer d.mu.RUnlock()
	result := make([]ports.AICallRecord, len(d.records))
	copy(result, d.records)
	return result
}
