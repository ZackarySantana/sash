package sashsse

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestBrokerPublishNoSubscribers(t *testing.T) {
	b := NewBroker()
	b.Publish("e", "hello")
	if err := b.PublishJSON("j", map[string]int{"x": 1}); err != nil {
		t.Fatal(err)
	}
}

func TestBrokerSSEHeadersAndTick(t *testing.T) {
	b := NewBroker()
	srv := httptest.NewServer(WithCORS(b))
	defer srv.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(20 * time.Millisecond)
		_ = b.PublishJSON("tick", map[string]int{"sec": 1})
	}()

	res, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if ct := res.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("Content-Type = %q", ct)
	}

	buf := make([]byte, 256)
	n, _ := res.Body.Read(buf)
	body := string(buf[:n])
	if !strings.Contains(body, "event: tick") || !strings.Contains(body, `"sec":1`) {
		t.Fatalf("unexpected chunk: %q", body)
	}
	wg.Wait()
}
