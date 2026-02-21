// Package sse provides Server-Sent Events (SSE) support for Kashvi.
//
// Usage:
//
//	router.Get("/events", "sse.stream", ctx.Wrap(func(c *ctx.Context) {
//	    stream := sse.New(c.W, c.R)
//	    for i := 0; i < 10; i++ {
//	        stream.Send("update", map[string]any{"tick": i})
//	        time.Sleep(time.Second)
//	        if stream.IsClosed() { break }
//	    }
//	}))
package sse

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Stream represents an active SSE connection to one client.
type Stream struct {
	w       http.ResponseWriter
	r       *http.Request
	flusher http.Flusher
	closed  bool
}

// New creates an SSE stream and sets the required headers.
// Returns nil if the ResponseWriter does not support flushing.
func New(w http.ResponseWriter, r *http.Request) *Stream {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "SSE not supported", http.StatusInternalServerError)
		return nil
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // disable nginx buffering

	return &Stream{w: w, r: r, flusher: flusher}
}

// Send writes a named SSE event with a JSON-encoded data payload.
func (s *Stream) Send(event string, data any) error {
	if s == nil || s.closed {
		return nil
	}

	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("sse: marshal: %w", err)
	}

	fmt.Fprintf(s.w, "event: %s\ndata: %s\n\n", event, payload)
	s.flusher.Flush()

	// Check if client disconnected.
	select {
	case <-s.r.Context().Done():
		s.closed = true
	default:
	}
	return nil
}

// SendRaw writes a raw SSE data line (no event name).
func (s *Stream) SendRaw(data string) {
	if s == nil || s.closed {
		return
	}
	fmt.Fprintf(s.w, "data: %s\n\n", data)
	s.flusher.Flush()
}

// Comment writes an SSE comment (useful as a keepalive heartbeat).
func (s *Stream) Comment(msg string) {
	if s == nil || s.closed {
		return
	}
	fmt.Fprintf(s.w, ": %s\n\n", msg)
	s.flusher.Flush()
}

// IsClosed reports whether the client has disconnected.
func (s *Stream) IsClosed() bool {
	if s == nil {
		return true
	}
	select {
	case <-s.r.Context().Done():
		s.closed = true
	default:
	}
	return s.closed
}
