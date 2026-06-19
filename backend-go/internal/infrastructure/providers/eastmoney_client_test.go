package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestEastmoneyClientWaitsBetweenRequests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("User-Agent"); got == "" {
			t.Fatalf("expected browser user agent header")
		}
		if got := r.Header.Get("Referer"); got != "https://quote.eastmoney.com/" {
			t.Fatalf("unexpected referer header: %q", got)
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	current := time.Date(2026, 6, 2, 9, 30, 0, 0, time.Local)
	var sleeps []time.Duration
	client := newEastmoneyClient(&http.Client{Timeout: time.Second})
	client.minInterval = time.Second
	client.now = func() time.Time { return current }
	client.sleep = func(d time.Duration) {
		sleeps = append(sleeps, d)
		current = current.Add(d)
	}
	client.jitter = func() time.Duration { return 100 * time.Millisecond }

	if _, err := client.Get(context.Background(), server.URL, "https://quote.eastmoney.com/"); err != nil {
		t.Fatalf("first request failed: %v", err)
	}
	if _, err := client.Get(context.Background(), server.URL, "https://quote.eastmoney.com/"); err != nil {
		t.Fatalf("second request failed: %v", err)
	}

	if len(sleeps) != 1 {
		t.Fatalf("expected one throttle sleep, got %d: %+v", len(sleeps), sleeps)
	}
	if sleeps[0] != 1100*time.Millisecond {
		t.Fatalf("expected 1.1s throttle sleep, got %s", sleeps[0])
	}
}

func TestEastmoneyClientSerializesConcurrentRequests(t *testing.T) {
	var active int32
	var maxActive int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nowActive := atomic.AddInt32(&active, 1)
		for {
			previous := atomic.LoadInt32(&maxActive)
			if nowActive <= previous || atomic.CompareAndSwapInt32(&maxActive, previous, nowActive) {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
		atomic.AddInt32(&active, -1)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client := newEastmoneyClient(server.Client())
	client.minInterval = 0
	client.jitter = func() time.Duration { return 0 }

	var wg sync.WaitGroup
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := client.Get(context.Background(), server.URL, "https://quote.eastmoney.com/"); err != nil {
				t.Errorf("request failed: %v", err)
			}
		}()
	}
	wg.Wait()

	if maxActive != 1 {
		t.Fatalf("expected serialized upstream requests, saw %d active requests", maxActive)
	}
}

func TestEastmoneyClientReturnsStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "blocked", http.StatusTooManyRequests)
	}))
	defer server.Close()

	client := newEastmoneyClient(server.Client())
	client.minInterval = 0

	if _, err := client.Get(context.Background(), server.URL, "https://quote.eastmoney.com/"); err == nil {
		t.Fatalf("expected non-2xx response to return an error")
	}
}
