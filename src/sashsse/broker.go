package sashsse

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

type Broker struct {
	mu       sync.Mutex
	clients  map[chan []byte]struct{}
	ChanSize int
}

func NewBroker() *Broker {
	return &Broker{
		clients:  make(map[chan []byte]struct{}),
		ChanSize: 64,
	}
}

func (b *Broker) subscribe() (ch <-chan []byte, done func()) {
	buf := b.ChanSize
	if buf < 1 {
		buf = 16
	}
	c := make(chan []byte, buf)
	b.mu.Lock()
	b.clients[c] = struct{}{}
	b.mu.Unlock()
	return c, func() {
		b.mu.Lock()
		delete(b.clients, c)
		b.mu.Unlock()
	}
}

func (b *Broker) publishFrame(frame []byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for ch := range b.clients {
		select {
		case ch <- frame:
		default:
			// Avoid blocking publishers when a client stalls.
		}
	}
}

func (b *Broker) Publish(event string, data string) {
	buf := formatSSE(event, data)
	raw := buf.Bytes()
	dup := make([]byte, len(raw))
	copy(dup, raw)
	b.publishFrame(dup)
}

func (b *Broker) PublishJSON(event string, v any) error {
	raw, err := json.Marshal(v)
	if err != nil {
		return err
	}
	b.Publish(event, string(raw))
	return nil
}

func formatSSE(event string, data string) *bytes.Buffer {
	var buf bytes.Buffer
	if event != "" {
		fmt.Fprintf(&buf, "event: %s\n", strings.ReplaceAll(event, "\n", " "))
	}
	for _, line := range strings.Split(data, "\n") {
		fmt.Fprintf(&buf, "data: %s\n", line)
	}
	buf.WriteByte('\n')
	return &buf
}

func (b *Broker) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	h := w.Header()
	h.Set("Content-Type", "text/event-stream")
	h.Set("Cache-Control", "no-cache")
	h.Set("Connection", "keep-alive")
	h.Set("X-Accel-Buffering", "no")

	ch, unsub := b.subscribe()
	defer unsub()

	for {
		select {
		case <-r.Context().Done():
			return
		case chunk, ok := <-ch:
			if !ok {
				return
			}
			if _, err := w.Write(chunk); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
