package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"stock-predict-go/internal/dto"
)

var ErrModelUnsupportedFund = errors.New("model service does not cover fund")

type circuitState int

const (
	circuitClosed   circuitState = iota
	circuitOpen
	circuitHalfOpen
)

type ModelClient struct {
	baseURL      *url.URL
	httpClient   *http.Client
	logger       *slog.Logger
	circuitMu    sync.Mutex
	circuitState circuitState
	failCount    int
	openedAt     time.Time
}

type modelPredictionResponse struct {
	FundCode    string                 `json:"fund_code"`
	FundName    string                 `json:"fund_name"`
	AsOfTime    string                 `json:"asof_time"`
	Model       modelMetadata          `json:"model"`
	Prediction  modelPredictionPayload `json:"prediction"`
	DataQuality modelDataQuality       `json:"data_quality"`
	CreatedAt   string                 `json:"created_at"`
}

type modelMetadata struct {
	Candidate  string `json:"candidate"`
	FeatureSet string `json:"feature_set"`
	ModelPath  string `json:"model_path"`
}

type modelPredictionPayload struct {
	Horizon             string                   `json:"horizon"`
	TargetWindow        string                   `json:"target_window"`
	Direction           dto.Direction            `json:"direction"`
	DirectionConfidence float64                  `json:"direction_confidence"`
	PredictedChangePct  float64                  `json:"predicted_change_pct"`
	ChangeRange         dto.ChangeRange          `json:"change_range"`
	PredictionInterval  *dto.PredictionInterval  `json:"prediction_interval"`
	ReturnDecomposition *dto.ReturnDecomposition `json:"return_decomposition"`
	ActionabilityGate   *dto.ActionabilityGate   `json:"actionability_gate"`
	ClassProbabilities  map[string]float64       `json:"class_probabilities"`
	TopFactors          []modelFactor            `json:"top_factors"`
	SignalStatus        dto.SignalStatus         `json:"signal_status"`
	IsActionable        bool                     `json:"is_actionable"`
	Reliability         string                   `json:"reliability"`
	ReliabilityNote     string                   `json:"reliability_note"`
}

type modelFactor struct {
	Name        string   `json:"name"`
	Importance  float64  `json:"importance"`
	Value       *float64 `json:"value"`
	Description string   `json:"description"`
}

type modelDataQuality struct {
	FeatureCount       int    `json:"feature_count"`
	HasPanicFactor     bool   `json:"has_panic_factor"`
	HasFuturesFeatures bool   `json:"has_futures_features"`
	Note               string `json:"note"`
}

func NewModelClient(rawURL string, timeout time.Duration, logger *slog.Logger) (*ModelClient, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return nil, nil
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("invalid model service url %q", rawURL)
	}
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	return &ModelClient{
		baseURL:    parsed,
		httpClient: &http.Client{Timeout: timeout},
		logger:     logger,
	}, nil
}

func (c *ModelClient) Predict(ctx context.Context, fundCode string) (modelPredictionResponse, error) {
	var out modelPredictionResponse
	if c == nil {
		return out, fmt.Errorf("model client is not configured")
	}

	if !c.allowRequest() {
		return out, fmt.Errorf("model service circuit breaker is open")
	}

	endpoint := *c.baseURL
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/predict/" + url.PathEscape(fundCode)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		c.recordFailure()
		return out, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.recordFailure()
		return out, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		var payload struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&payload)
		if resp.StatusCode == http.StatusBadRequest && strings.Contains(payload.Error, "No sample row found") {
			return out, fmt.Errorf("%w: %s", ErrModelUnsupportedFund, payload.Error)
		}
		c.recordFailure()
		if payload.Error != "" {
			return out, fmt.Errorf("model service returned status %d: %s", resp.StatusCode, payload.Error)
		}
		return out, fmt.Errorf("model service returned status %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		c.recordFailure()
		return out, err
	}
	c.recordSuccess()
	if out.FundCode == "" {
		out.FundCode = fundCode
	}
	return out, nil
}

func (c *ModelClient) allowRequest() bool {
	c.circuitMu.Lock()
	defer c.circuitMu.Unlock()
	switch c.circuitState {
	case circuitClosed:
		return true
	case circuitOpen:
		if time.Since(c.openedAt) >= 30*time.Second {
			c.circuitState = circuitHalfOpen
			return true
		}
		return false
	case circuitHalfOpen:
		return true
	default:
		return true
	}
}

func (c *ModelClient) recordFailure() {
	c.circuitMu.Lock()
	defer c.circuitMu.Unlock()
	c.failCount++
	if c.failCount >= 3 {
		c.circuitState = circuitOpen
		c.openedAt = time.Now()
	}
}

func (c *ModelClient) recordSuccess() {
	c.circuitMu.Lock()
	defer c.circuitMu.Unlock()
	c.failCount = 0
	c.circuitState = circuitClosed
}
