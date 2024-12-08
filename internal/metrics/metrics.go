package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

// Metrics tracks application performance and error statistics
type Metrics struct {
	processedMessages   int64         // Total number of processed messages
	droppedMessages     int64         // Number of messages dropped due to queue overflow
	processingErrors    int64         // Number of processing errors
	avgProcessingTime   time.Duration // Average message processing time
	totalProcessingTime time.Duration // Total processing time for all messages
	mu                  sync.RWMutex  // Mutex for thread-safe metrics updates
}

// NewMetrics creates a new Metrics instance
func NewMetrics() *Metrics {
	return &Metrics{}
}

// RecordProcessingTime updates message processing time metrics
func (m *Metrics) RecordProcessingTime(d time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.totalProcessingTime += d
	m.processedMessages++
	m.avgProcessingTime = time.Duration(int64(m.totalProcessingTime) / m.processedMessages)
}

// IncrementDropped atomically increments the dropped messages counter
func (m *Metrics) IncrementDropped() {
	atomic.AddInt64(&m.droppedMessages, 1)
}

// IncrementErrors atomically increments the processing errors counter
func (m *Metrics) IncrementErrors() {
	atomic.AddInt64(&m.processingErrors, 1)
}

// GetStats returns current metrics statistics
func (m *Metrics) GetStats() (processed, dropped, errors int64, avgTime time.Duration) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.processedMessages, m.droppedMessages, m.processingErrors, m.avgProcessingTime
}
