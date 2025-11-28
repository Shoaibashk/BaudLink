/*
Copyright 2024 BaudLink Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package serial

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

// Reader provides continuous reading from a serial port with streaming support
type Reader struct {
	manager     *Manager
	portName    string
	sessionID   string
	bufferSize  int
	running     atomic.Bool
	stopChan    chan struct{}
	subscribers []chan DataEvent
	subMu       sync.RWMutex
}

// DataEvent represents a data read event
type DataEvent struct {
	Data      []byte
	Timestamp time.Time
	Sequence  uint32
	Error     error
}

// NewReader creates a new continuous reader for a port
func NewReader(manager *Manager, portName, sessionID string, bufferSize int) *Reader {
	if bufferSize <= 0 {
		bufferSize = 1024
	}

	return &Reader{
		manager:     manager,
		portName:    portName,
		sessionID:   sessionID,
		bufferSize:  bufferSize,
		stopChan:    make(chan struct{}),
		subscribers: make([]chan DataEvent, 0),
	}
}

// Start begins continuous reading from the port
func (r *Reader) Start(ctx context.Context) error {
	if r.running.Load() {
		return nil
	}

	// Validate session
	_, err := r.manager.ValidateSession(r.portName, r.sessionID)
	if err != nil {
		return err
	}

	r.running.Store(true)

	go r.readLoop(ctx)

	return nil
}

// Stop stops the continuous reader
func (r *Reader) Stop() {
	if !r.running.Load() {
		return
	}

	r.running.Store(false)
	close(r.stopChan)

	// Close all subscriber channels
	r.subMu.Lock()
	for _, ch := range r.subscribers {
		close(ch)
	}
	r.subscribers = nil
	r.subMu.Unlock()
}

// Subscribe creates a new subscription to read events
func (r *Reader) Subscribe() <-chan DataEvent {
	ch := make(chan DataEvent, 100)

	r.subMu.Lock()
	r.subscribers = append(r.subscribers, ch)
	r.subMu.Unlock()

	return ch
}

// Unsubscribe removes a subscription
func (r *Reader) Unsubscribe(ch <-chan DataEvent) {
	r.subMu.Lock()
	defer r.subMu.Unlock()

	for i, sub := range r.subscribers {
		if sub == ch {
			close(sub)
			r.subscribers = append(r.subscribers[:i], r.subscribers[i+1:]...)
			return
		}
	}
}

// readLoop continuously reads from the port
func (r *Reader) readLoop(ctx context.Context) {
	var sequence uint32

	for r.running.Load() {
		select {
		case <-ctx.Done():
			r.Stop()
			return
		case <-r.stopChan:
			return
		default:
			data, err := r.manager.Read(r.portName, r.sessionID, r.bufferSize)
			
			// Skip if no data (timeout with no data is normal)
			if err == nil && len(data) == 0 {
				continue
			}

			event := DataEvent{
				Data:      data,
				Timestamp: time.Now(),
				Sequence:  atomic.AddUint32(&sequence, 1),
				Error:     err,
			}

			r.broadcast(event)

			if err != nil {
				// Check if it's a fatal error
				if err == ErrPortClosed || err == ErrInvalidSession {
					r.Stop()
					return
				}
				// Non-fatal errors - continue reading
				time.Sleep(10 * time.Millisecond)
			}
		}
	}
}

// broadcast sends an event to all subscribers
func (r *Reader) broadcast(event DataEvent) {
	r.subMu.RLock()
	defer r.subMu.RUnlock()

	for _, ch := range r.subscribers {
		select {
		case ch <- event:
		default:
			// Channel full, drop the event
		}
	}
}

// IsRunning returns whether the reader is currently running
func (r *Reader) IsRunning() bool {
	return r.running.Load()
}

// ReadResult represents the result of a single read operation
type ReadResult struct {
	Data  []byte
	Error error
}

// ReadWithTimeout reads data with a specific timeout
func ReadWithTimeout(manager *Manager, portName, sessionID string, maxBytes int, timeout time.Duration) ReadResult {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	resultChan := make(chan ReadResult, 1)

	go func() {
		data, err := manager.Read(portName, sessionID, maxBytes)
		resultChan <- ReadResult{Data: data, Error: err}
	}()

	select {
	case result := <-resultChan:
		return result
	case <-ctx.Done():
		return ReadResult{Error: ErrReadTimeout}
	}
}

// WriteWithTimeout writes data with a specific timeout
func WriteWithTimeout(manager *Manager, portName, sessionID string, data []byte, timeout time.Duration) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	type writeResult struct {
		n   int
		err error
	}

	resultChan := make(chan writeResult, 1)

	go func() {
		n, err := manager.Write(portName, sessionID, data)
		resultChan <- writeResult{n: n, err: err}
	}()

	select {
	case result := <-resultChan:
		return result.n, result.err
	case <-ctx.Done():
		return 0, ErrWriteTimeout
	}
}

// Ticker is a wrapper around time.Ticker for port scanning
type Ticker struct {
	C    <-chan time.Time
	t    *time.Ticker
}

// NewTicker creates a new ticker with the given interval in seconds
func NewTicker(intervalSeconds int) *Ticker {
	t := time.NewTicker(time.Duration(intervalSeconds) * time.Second)
	return &Ticker{
		C: t.C,
		t: t,
	}
}

// Stop stops the ticker
func (t *Ticker) Stop() {
	t.t.Stop()
}

// LineReader reads complete lines from the port
type LineReader struct {
	reader    *Reader
	delimiter byte
	buffer    []byte
	maxLine   int
}

// NewLineReader creates a new line-based reader
func NewLineReader(reader *Reader, delimiter byte, maxLineSize int) *LineReader {
	if maxLineSize <= 0 {
		maxLineSize = 4096
	}

	return &LineReader{
		reader:    reader,
		delimiter: delimiter,
		buffer:    make([]byte, 0, maxLineSize),
		maxLine:   maxLineSize,
	}
}

// ReadLine reads a complete line from the subscription channel
func (lr *LineReader) ReadLine(dataChan <-chan DataEvent) ([]byte, error) {
	for {
		// Check buffer for existing line
		for i, b := range lr.buffer {
			if b == lr.delimiter {
				line := make([]byte, i)
				copy(line, lr.buffer[:i])
				lr.buffer = lr.buffer[i+1:]
				return line, nil
			}
		}

		// Wait for more data
		event, ok := <-dataChan
		if !ok {
			// Channel closed
			if len(lr.buffer) > 0 {
				line := lr.buffer
				lr.buffer = nil
				return line, nil
			}
			return nil, ErrPortClosed
		}

		if event.Error != nil {
			return nil, event.Error
		}

		// Append to buffer
		lr.buffer = append(lr.buffer, event.Data...)

		// Check for buffer overflow
		if len(lr.buffer) > lr.maxLine {
			// Return partial line and reset
			line := lr.buffer
			lr.buffer = nil
			return line, nil
		}
	}
}
