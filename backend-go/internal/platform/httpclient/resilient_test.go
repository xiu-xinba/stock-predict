package httpclient

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

type resilientRoundTripFunc func(*http.Request) (*http.Response, error)

func (f resilientRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func testResponse(status int, body string, headers map[string]string) *http.Response {
	resp := &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	for k, v := range headers {
		resp.Header.Set(k, v)
	}
	return resp
}

func TestResilientHTTPClientAppliesHeadersAndRateLimit(t *testing.T) {
	var requests []*http.Request
	base := &http.Client{Transport: resilientRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests = append(requests, req.Clone(req.Context()))
		return testResponse(http.StatusOK, `{}`, nil), nil
	})}
	now := time.Date(2026, 6, 4, 10, 0, 0, 0, time.UTC)
	var slept []time.Duration
	client := NewResilientHTTPClient(base, []SourcePolicy{{
		Source:      SourceTencent,
		UserAgent:   "StockPredict-Test/1.0",
		Referer:     "https://gu.qq.com/",
		MinInterval: time.Second,
	}})
	client.now = func() time.Time { return now }
	client.sleep = func(ctx context.Context, d time.Duration) error {
		slept = append(slept, d)
		now = now.Add(d)
		return nil
	}

	req1, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://qt.gtimg.cn/q=sh600519", nil)
	if err != nil {
		t.Fatalf("create request 1: %v", err)
	}
	if _, err := client.Do(context.Background(), SourceTencent, req1); err != nil {
		t.Fatalf("first request failed: %v", err)
	}

	req2, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://qt.gtimg.cn/q=sh000001", nil)
	if err != nil {
		t.Fatalf("create request 2: %v", err)
	}
	if _, err := client.Do(context.Background(), SourceTencent, req2); err != nil {
		t.Fatalf("second request failed: %v", err)
	}

	if len(requests) != 2 {
		t.Fatalf("expected two upstream requests, got %d", len(requests))
	}
	if got := requests[0].Header.Get("User-Agent"); got != "StockPredict-Test/1.0" {
		t.Fatalf("unexpected user agent: %q", got)
	}
	if got := requests[0].Header.Get("Referer"); got != "https://gu.qq.com/" {
		t.Fatalf("unexpected referer: %q", got)
	}
	if len(slept) != 1 || slept[0] < time.Second {
		t.Fatalf("expected one rate-limit sleep >= 1s, got %+v", slept)
	}
}

func TestResilientHTTPClientHonorsRetryAfterCooldown(t *testing.T) {
	callCount := 0
	base := &http.Client{Transport: resilientRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		callCount++
		if callCount == 1 {
			return testResponse(http.StatusTooManyRequests, `{}`, map[string]string{"Retry-After": "2"}), nil
		}
		return testResponse(http.StatusOK, `{}`, nil), nil
	})}
	now := time.Date(2026, 6, 4, 10, 0, 0, 0, time.UTC)
	var slept []time.Duration
	client := NewResilientHTTPClient(base, []SourcePolicy{{
		Source:      SourceEastmoney,
		UserAgent:   "StockPredict-Test/1.0",
		Referer:     "https://quote.eastmoney.com/",
		MinInterval: 0,
	}})
	client.now = func() time.Time { return now }
	client.sleep = func(ctx context.Context, d time.Duration) error {
		slept = append(slept, d)
		now = now.Add(d)
		return nil
	}

	req1, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://push2.eastmoney.com/api/test", nil)
	if err != nil {
		t.Fatalf("create request 1: %v", err)
	}
	resp, err := client.Do(context.Background(), SourceEastmoney, req1)
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}
	_ = resp.Body.Close()

	req2, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://push2.eastmoney.com/api/test", nil)
	if err != nil {
		t.Fatalf("create request 2: %v", err)
	}
	resp, err = client.Do(context.Background(), SourceEastmoney, req2)
	if err != nil {
		t.Fatalf("second request failed: %v", err)
	}
	_ = resp.Body.Close()

	if callCount != 2 {
		t.Fatalf("expected two upstream calls, got %d", callCount)
	}
	if len(slept) != 1 || slept[0] < 2*time.Second {
		t.Fatalf("expected Retry-After cooldown sleep >= 2s, got %+v", slept)
	}
}

func TestResilientHTTPClientCoalescesConcurrentRequests(t *testing.T) {
	var mu sync.Mutex
	callCount := 0
	release := make(chan struct{})
	base := &http.Client{Transport: resilientRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		mu.Lock()
		callCount++
		mu.Unlock()
		<-release
		return testResponse(http.StatusOK, `ok`, nil), nil
	})}
	client := NewResilientHTTPClient(base, []SourcePolicy{{
		Source:    SourceTencent,
		UserAgent: "StockPredict-Test/1.0",
		Referer:   "https://gu.qq.com/",
	}})

	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://qt.gtimg.cn/q=sh600519", nil)
			if err != nil {
				errs <- err
				return
			}
			resp, err := client.Do(context.Background(), SourceTencent, req)
			if resp != nil {
				_, _ = io.ReadAll(resp.Body)
				_ = resp.Body.Close()
			}
			errs <- err
		}()
	}

	time.Sleep(20 * time.Millisecond)
	close(release)
	for i := 0; i < 2; i++ {
		if err := <-errs; err != nil {
			t.Fatalf("request failed: %v", err)
		}
	}
	if callCount != 1 {
		t.Fatalf("expected coalesced requests to hit upstream once, got %d", callCount)
	}
}

func TestResilientHTTPClientCoalescesWhileSharedRequestIsRateLimited(t *testing.T) {
	var mu sync.Mutex
	callCount := 0
	base := &http.Client{Transport: resilientRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		mu.Lock()
		callCount++
		mu.Unlock()
		time.Sleep(20 * time.Millisecond)
		return testResponse(http.StatusOK, `ok`, nil), nil
	})}
	client := NewResilientHTTPClient(base, []SourcePolicy{{
		Source:      SourceTencent,
		MinInterval: 80 * time.Millisecond,
	}})

	prime, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://qt.gtimg.cn/prime", nil)
	if err != nil {
		t.Fatalf("create prime request: %v", err)
	}
	resp, err := client.Do(context.Background(), SourceTencent, prime)
	if err != nil {
		t.Fatalf("prime request failed: %v", err)
	}
	_ = resp.Body.Close()

	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		go func() {
			req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://qt.gtimg.cn/shared", nil)
			if err != nil {
				errs <- err
				return
			}
			resp, err := client.Do(context.Background(), SourceTencent, req)
			if resp != nil {
				_, _ = io.ReadAll(resp.Body)
				_ = resp.Body.Close()
			}
			errs <- err
		}()
	}
	for i := 0; i < 2; i++ {
		if err := <-errs; err != nil {
			t.Fatalf("shared request failed: %v", err)
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if callCount != 2 {
		t.Fatalf("expected prime plus one coalesced upstream request, got %d calls", callCount)
	}
}

func TestResilientHTTPClientCallerCancellationDoesNotCancelSharedRequest(t *testing.T) {
	var mu sync.Mutex
	callCount := 0
	started := make(chan struct{})
	release := make(chan struct{})
	base := &http.Client{Transport: resilientRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		mu.Lock()
		callCount++
		if callCount == 1 {
			close(started)
		}
		mu.Unlock()
		select {
		case <-release:
			return testResponse(http.StatusOK, `shared`, nil), nil
		case <-req.Context().Done():
			return nil, req.Context().Err()
		}
	})}
	client := NewResilientHTTPClient(base, []SourcePolicy{{
		Source:    SourceTencent,
		UserAgent: "StockPredict-Test/1.0",
	}})

	sharedResult := make(chan error, 1)
	go func() {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://qt.gtimg.cn/q=sh600519", nil)
		if err != nil {
			sharedResult <- err
			return
		}
		resp, err := client.Do(context.Background(), SourceTencent, req)
		if resp != nil {
			_, _ = io.ReadAll(resp.Body)
			_ = resp.Body.Close()
		}
		sharedResult <- err
	}()

	<-started
	callerCtx, cancelCaller := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancelCaller()
	req, err := http.NewRequestWithContext(callerCtx, http.MethodGet, "https://qt.gtimg.cn/q=sh600519", nil)
	if err != nil {
		t.Fatalf("create cancelable request: %v", err)
	}
	_, err = client.Do(callerCtx, SourceTencent, req)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("cancelable caller error = %v, want context deadline exceeded", err)
	}

	close(release)
	if err := <-sharedResult; err != nil {
		t.Fatalf("shared request failed after another caller canceled: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("expected one shared upstream request, got %d", callCount)
	}
}

func TestResilientHTTPClientSharedRequestHasBoundedTimeout(t *testing.T) {
	base := &http.Client{Transport: resilientRoundTripFunc(func(req *http.Request) (*http.Response, error) {
		<-req.Context().Done()
		return nil, req.Context().Err()
	})}
	client := NewResilientHTTPClient(base, []SourcePolicy{{
		Source: SourceTencent,
	}})
	client.sharedRequestTimeout = 25 * time.Millisecond

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://qt.gtimg.cn/q=sh600519", nil)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	start := time.Now()
	_, err = client.Do(context.Background(), SourceTencent, req)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("request error = %v, want context deadline exceeded", err)
	}
	if elapsed := time.Since(start); elapsed > 200*time.Millisecond {
		t.Fatalf("shared request timeout was not bounded: %s", elapsed)
	}
}
