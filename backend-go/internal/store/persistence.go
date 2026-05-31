package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"stock-predict-go/internal/dto"
)

var ErrNoSyncSource = errors.New("fund sync source is required")

var fileMu sync.Mutex

type fundRecord struct {
	FundCode      string  `json:"fund_code"`
	FundName      string  `json:"fund_name"`
	FundType      string  `json:"fund_type"`
	PinyinAbbr    string  `json:"pinyin_abbr,omitempty"`
	PinyinFull    string  `json:"pinyin_full,omitempty"`
	Company       string  `json:"company,omitempty"`
	Manager       string  `json:"manager,omitempty"`
	LatestNAV     float64 `json:"latest_nav"`
	CumulativeNAV float64 `json:"cumulative_nav"`
	Return1M      float64 `json:"return_1m"`
	Return3M      float64 `json:"return_3m"`
	Return6M      float64 `json:"return_6m"`
	Return1Y      float64 `json:"return_1y"`
	Return3Y      float64 `json:"return_3y"`
	RiskLevel     string  `json:"risk_level,omitempty"`
	InceptionDate string  `json:"inception_date,omitempty"`
	EstimatedNAV  float64 `json:"estimated_nav"`
	ChangePct     float64 `json:"change_pct"`
	QuoteDate     string  `json:"quote_date,omitempty"`
	QuoteSource   string  `json:"quote_source,omitempty"`
}

func NewPersistentStore(path string) (*MemoryStore, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return NewMemoryStore(), nil
	}
	items, err := readFundsJSON(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
		store := NewMemoryStore()
		store.path = path
		if err := store.saveLocked(); err != nil {
			return nil, err
		}
		return store, nil
	}
	store := NewMemoryStoreWithFunds(mergeSeedFunds(items))
	store.path = path
	return store, nil
}

func (s *MemoryStore) ReplaceFunds(funds []dto.FundItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	next := make(map[string]dto.FundItem, len(funds))
	for _, fund := range funds {
		fund.FundCode = normalizeFundCode(fund.FundCode)
		if fund.FundCode == "" {
			continue
		}
		next[fund.FundCode] = fund
	}
	if len(next) == 0 {
		return fmt.Errorf("no valid funds to store")
	}
	s.funds = next
	return s.saveLocked()
}

func (s *MemoryStore) MergeFunds(funds []dto.FundItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	valid := 0
	for _, fund := range funds {
		fund.FundCode = normalizeFundCode(fund.FundCode)
		if fund.FundCode == "" {
			continue
		}
		if existing, ok := s.funds[fund.FundCode]; ok {
			fund = mergeFund(existing, fund)
		}
		s.funds[fund.FundCode] = fund
		valid++
	}
	if valid == 0 {
		return fmt.Errorf("no valid funds to store")
	}
	return s.saveLocked()
}

func (s *MemoryStore) DataPath() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.path
}

func (s *MemoryStore) saveLocked() error {
	if strings.TrimSpace(s.path) == "" {
		return nil
	}
	items := make([]dto.FundItem, 0, len(s.funds))
	for _, fund := range s.funds {
		items = append(items, fund)
	}
	return writeFundsJSON(s.path, items)
}

func readFundsJSON(path string) ([]dto.FundItem, error) {
	fileMu.Lock()
	defer fileMu.Unlock()
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var records []fundRecord
	if err := json.NewDecoder(file).Decode(&records); err != nil {
		return nil, err
	}
	out := make([]dto.FundItem, 0, len(records))
	for _, record := range records {
		out = append(out, record.toFundItem())
	}
	return out, nil
}

func writeFundsJSON(path string, funds []dto.FundItem) error {
	fileMu.Lock()
	defer fileMu.Unlock()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	records := make([]fundRecord, 0, len(funds))
	for _, fund := range funds {
		records = append(records, newFundRecord(fund))
	}
	payload, err := json.MarshalIndent(records, "", "  ")
	if err != nil {
		return err
	}
	payload = append(payload, '\n')
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, payload, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		os.Remove(tmpPath)
		return err
	}
	return nil
}

func newFundRecord(f dto.FundItem) fundRecord {
	return fundRecord{
		FundCode:      f.FundCode,
		FundName:      f.FundName,
		FundType:      f.FundType,
		PinyinAbbr:    f.PinyinAbbr,
		PinyinFull:    f.PinyinFull,
		Company:       f.Company,
		Manager:       f.Manager,
		LatestNAV:     f.LatestNAV,
		CumulativeNAV: f.CumulativeNAV,
		Return1M:      f.Return1M,
		Return3M:      f.Return3M,
		Return6M:      f.Return6M,
		Return1Y:      f.Return1Y,
		Return3Y:      f.Return3Y,
		RiskLevel:     f.RiskLevel,
		InceptionDate: f.InceptionDate,
		EstimatedNAV:  f.EstimatedNAV,
		ChangePct:     f.ChangePct,
		QuoteDate:     f.QuoteDate,
		QuoteSource:   f.QuoteSource,
	}
}

func (r fundRecord) toFundItem() dto.FundItem {
	return dto.FundItem{
		FundCode:      r.FundCode,
		FundName:      r.FundName,
		FundType:      r.FundType,
		PinyinAbbr:    r.PinyinAbbr,
		PinyinFull:    r.PinyinFull,
		Company:       r.Company,
		Manager:       r.Manager,
		LatestNAV:     r.LatestNAV,
		CumulativeNAV: r.CumulativeNAV,
		Return1M:      r.Return1M,
		Return3M:      r.Return3M,
		Return6M:      r.Return6M,
		Return1Y:      r.Return1Y,
		Return3Y:      r.Return3Y,
		RiskLevel:     r.RiskLevel,
		InceptionDate: r.InceptionDate,
		EstimatedNAV:  r.EstimatedNAV,
		ChangePct:     r.ChangePct,
		QuoteDate:     r.QuoteDate,
		QuoteSource:   r.QuoteSource,
	}
}
