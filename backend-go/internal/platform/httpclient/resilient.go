// Package httpclient 提供了带重试、熔断和限流策略的 HTTP 客户端封装。
package httpclient

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

type sourceState struct {
	lastCall      time.Time
	cooldownUntil time.Time
}

type bufferedHTTPResponse struct {
	statusCode    int
	header        http.Header
	body          []byte
	contentLength int64
}

// ResilientHTTPClient 是具备限速、熔断和请求合并能力的 HTTP 客户端。
// 每个数据源维护独立的调用状态（最近调用时间、冷却截止时间），
// GET 请求通过 singleflight 自动合并并发调用。
type ResilientHTTPClient struct {
	client               *http.Client                               // 底层 HTTP 客户端
	policies             map[string]SourcePolicy                    // 数据源标识到策略的映射
	states               map[string]*sourceState                    // 数据源标识到运行时状态的映射
	mu                   sync.Mutex                                 // 保护 states 的互斥锁
	group                singleflight.Group                         // 合并并发 GET 请求
	now                  func() time.Time                           // 获取当前时间的函数（便于测试注入）
	sleep                func(context.Context, time.Duration) error // 可取消的等待函数
	sharedRequestTimeout time.Duration                              // 合并请求的超时时间
}

// NewResilientHTTPClient 创建一个弹性 HTTP 客户端，合并默认策略和自定义策略。
// 若 client 为 nil 则使用默认配置创建底层客户端。
func NewResilientHTTPClient(client *http.Client, policies []SourcePolicy) *ResilientHTTPClient {
	if client == nil {
		client = New(Config{})
	}
	merged := make(map[string]SourcePolicy)
	for _, policy := range DefaultSourcePolicies() {
		merged[policy.Source] = normalizeSourcePolicy(policy)
	}
	for _, policy := range policies {
		merged[policy.Source] = normalizeSourcePolicy(policy)
	}
	return &ResilientHTTPClient{
		client:               client,
		policies:             merged,
		states:               make(map[string]*sourceState),
		now:                  time.Now,
		sleep:                sleepContext,
		sharedRequestTimeout: 15 * time.Second,
	}
}

// Do 发送 HTTP 请求，自动应用数据源策略（限速、请求头、冷却）。
// GET 请求通过 singleflight 合并并发调用以减少重复请求。
func (c *ResilientHTTPClient) Do(ctx context.Context, source string, req *http.Request) (*http.Response, error) {
	if req == nil {
		return nil, fmt.Errorf("nil request")
	}
	if ctx == nil {
		ctx = req.Context()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	policy := c.policy(source)

	out := req.Clone(ctx)
	applySourceHeaders(out, policy)
	if out.Method != http.MethodGet {
		if err := c.wait(ctx, policy); err != nil {
			return nil, err
		}
		return c.doOnce(ctx, policy, out)
	}

	key := policy.Source + ":" + out.Method + ":" + out.URL.String()
	resultCh := c.group.DoChan(key, func() (any, error) {
		timeout := c.sharedRequestTimeout
		if timeout <= 0 {
			timeout = 15 * time.Second
		}
		sharedCtx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		if err := c.wait(sharedCtx, policy); err != nil {
			return nil, err
		}
		sharedRequest := out.Clone(sharedCtx)
		resp, err := c.doOnce(sharedCtx, policy, sharedRequest)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(io.LimitReader(resp.Body, int64(MaxProviderPayload)))
		if err != nil {
			return nil, err
		}
		return &bufferedHTTPResponse{
			statusCode:    resp.StatusCode,
			header:        resp.Header.Clone(),
			body:          body,
			contentLength: int64(len(body)),
		}, nil
	})

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-resultCh:
		if result.Err != nil {
			return nil, result.Err
		}
		buffered, ok := result.Val.(*bufferedHTTPResponse)
		if !ok {
			return nil, fmt.Errorf("unexpected response type %T", result.Val)
		}
		return buffered.toResponse(out), nil
	}
}

func (c *ResilientHTTPClient) doOnce(_ context.Context, policy SourcePolicy, req *http.Request) (*http.Response, error) {
	resp, err := c.client.Do(req)
	if err != nil {
		c.applyErrorCooldown(policy)
		return nil, err
	}
	c.applyResponseCooldown(policy, resp)
	return resp, nil
}

func (c *ResilientHTTPClient) policy(source string) SourcePolicy {
	if policy, ok := c.policies[source]; ok {
		return policy
	}
	return normalizeSourcePolicy(SourcePolicy{Source: source})
}

// wait 根据数据源策略等待限速间隔或冷却时间，确保请求频率不超过策略限制。
// 先计算冷却剩余时间，再计算最小请求间隔的等待时间，取较大值。
func (c *ResilientHTTPClient) wait(ctx context.Context, policy SourcePolicy) error {
	c.mu.Lock()
	state := c.states[policy.Source]
	if state == nil {
		state = &sourceState{}
		c.states[policy.Source] = state
	}
	now := c.now()
	wait := time.Duration(0)
	if !state.cooldownUntil.IsZero() && now.Before(state.cooldownUntil) {
		wait = state.cooldownUntil.Sub(now)
	}
	if policy.MinInterval > 0 && !state.lastCall.IsZero() {
		rateWait := policy.MinInterval - now.Sub(state.lastCall)
		if rateWait > wait {
			wait = rateWait
		}
	}
	state.lastCall = now.Add(wait)
	c.mu.Unlock()

	if wait <= 0 {
		return nil
	}
	return c.sleep(ctx, wait)
}

func (c *ResilientHTTPClient) applyResponseCooldown(policy SourcePolicy, resp *http.Response) {
	if resp == nil {
		return
	}
	switch resp.StatusCode {
	case http.StatusForbidden, http.StatusTooManyRequests, http.StatusServiceUnavailable:
		cooldown := retryAfterDuration(resp.Header.Get("Retry-After"), c.now())
		if cooldown <= 0 {
			cooldown = policy.CooldownOnLimit
		}
		c.setCooldown(policy.Source, cooldown)
	}
}

func (c *ResilientHTTPClient) applyErrorCooldown(policy SourcePolicy) {
	if policy.CooldownOnError <= 0 {
		return
	}
	c.setCooldown(policy.Source, policy.CooldownOnError)
}

func (c *ResilientHTTPClient) setCooldown(source string, duration time.Duration) {
	if duration <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	state := c.states[source]
	if state == nil {
		state = &sourceState{}
		c.states[source] = state
	}
	until := c.now().Add(duration)
	if until.After(state.cooldownUntil) {
		state.cooldownUntil = until
	}
}

func (r *bufferedHTTPResponse) toResponse(req *http.Request) *http.Response {
	return &http.Response{
		StatusCode:    r.statusCode,
		Header:        r.header.Clone(),
		Body:          io.NopCloser(bytes.NewReader(r.body)),
		ContentLength: r.contentLength,
		Request:       req,
	}
}

func normalizeSourcePolicy(policy SourcePolicy) SourcePolicy {
	if policy.Source == "" {
		policy.Source = "unknown"
	}
	if policy.UserAgent == "" {
		policy.UserAgent = CompliantUserAgent
	}
	if policy.Accept == "" {
		policy.Accept = "*/*"
	}
	if policy.CooldownOnLimit <= 0 {
		policy.CooldownOnLimit = 30 * time.Second
	}
	if policy.CooldownOnError < 0 {
		policy.CooldownOnError = 0
	}
	return policy
}

func applySourceHeaders(req *http.Request, policy SourcePolicy) {
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", policy.UserAgent)
	}
	if policy.Referer != "" && req.Header.Get("Referer") == "" {
		req.Header.Set("Referer", policy.Referer)
	}
	if policy.Accept != "" && req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", policy.Accept)
	}
}

func retryAfterDuration(value string, now time.Time) time.Duration {
	if value == "" {
		return 0
	}
	if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second
	}
	if retryAt, err := http.ParseTime(value); err == nil && retryAt.After(now) {
		return retryAt.Sub(now)
	}
	return 0
}

func sleepContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
