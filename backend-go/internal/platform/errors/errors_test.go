package errors

import (
	"errors"
	"net/http"
	"testing"
)

func TestAppErrorErrorMessage(t *testing.T) {
	if ErrInvalidFundCode.Error() != "invalid fund code" {
		t.Fatalf("unexpected error message: %q", ErrInvalidFundCode.Error())
	}
	if ErrFundNotFound.Error() != "fund not found" {
		t.Fatalf("unexpected error message: %q", ErrFundNotFound.Error())
	}
	if ErrInvalidStockCode.Error() != "invalid stock code" {
		t.Fatalf("unexpected error message: %q", ErrInvalidStockCode.Error())
	}
	if ErrStockNotFound.Error() != "stock not found" {
		t.Fatalf("unexpected error message: %q", ErrStockNotFound.Error())
	}
	if ErrInvalidRankingType.Error() != "invalid ranking type" {
		t.Fatalf("unexpected error message: %q", ErrInvalidRankingType.Error())
	}
	if ErrSyncSourceRequired.Error() != "fund sync source is required" {
		t.Fatalf("unexpected error message: %q", ErrSyncSourceRequired.Error())
	}
	if ErrSyncUnsupported.Error() != "fund repository does not support sync" {
		t.Fatalf("unexpected error message: %q", ErrSyncUnsupported.Error())
	}
}

func TestAppErrorIsMatchesByCode(t *testing.T) {
	if !errors.Is(ErrInvalidFundCode, ErrInvalidFundCode) {
		t.Fatalf("expected ErrInvalidFundCode to match itself")
	}
	if !errors.Is(ErrFundNotFound, ErrFundNotFound) {
		t.Fatalf("expected ErrFundNotFound to match itself")
	}
	if !errors.Is(ErrInvalidStockCode, ErrInvalidStockCode) {
		t.Fatalf("expected ErrInvalidStockCode to match itself")
	}
	if !errors.Is(ErrStockNotFound, ErrStockNotFound) {
		t.Fatalf("expected ErrStockNotFound to match itself")
	}
	if !errors.Is(ErrInvalidRankingType, ErrInvalidRankingType) {
		t.Fatalf("expected ErrInvalidRankingType to match itself")
	}
	if !errors.Is(ErrSyncSourceRequired, ErrSyncSourceRequired) {
		t.Fatalf("expected ErrSyncSourceRequired to match itself")
	}
	if !errors.Is(ErrSyncUnsupported, ErrSyncUnsupported) {
		t.Fatalf("expected ErrSyncUnsupported to match itself")
	}
}

func TestAppErrorIsDifferentCode(t *testing.T) {
	if errors.Is(ErrInvalidFundCode, ErrFundNotFound) {
		t.Fatalf("expected different error codes not to match")
	}
}

func TestNewAppError(t *testing.T) {
	err := NewAppError(99999, "test error", http.StatusTeapot)
	if err.Code != 99999 {
		t.Fatalf("expected code 99999, got %d", err.Code)
	}
	if err.Message != "test error" {
		t.Fatalf("expected message 'test error', got %q", err.Message)
	}
	if err.HTTPStatus != http.StatusTeapot {
		t.Fatalf("expected status 418, got %d", err.HTTPStatus)
	}
	if err.Error() != "test error" {
		t.Fatalf("expected Error() to return message, got %q", err.Error())
	}
}
