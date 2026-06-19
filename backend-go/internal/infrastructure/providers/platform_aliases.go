package providers

import (
	"time"

	platformcache "stock-predict-go/internal/platform/cache"
	apperrors "stock-predict-go/internal/platform/errors"
)

// AppError 应用错误的类型别名，引用 apperrors.AppError。
type AppError = apperrors.AppError

// DetailCache 详情缓存的类型别名，引用 platformcache.DetailCache。
type DetailCache = platformcache.DetailCache

const (
	// ErrCodeInvalidFundCode 无效基金代码错误码
	ErrCodeInvalidFundCode = apperrors.ErrCodeInvalidFundCode
	// ErrCodeFundNotFound 基金未找到错误码
	ErrCodeFundNotFound = apperrors.ErrCodeFundNotFound
	// ErrCodeInvalidStockCode 无效股票代码错误码
	ErrCodeInvalidStockCode = apperrors.ErrCodeInvalidStockCode
	// ErrCodeStockNotFound 股票未找到错误码
	ErrCodeStockNotFound = apperrors.ErrCodeStockNotFound
	// ErrCodeInvalidRankingType 无效排名类型错误码
	ErrCodeInvalidRankingType = apperrors.ErrCodeInvalidRankingType
	// ErrCodeSyncSourceRequired 同步源必填错误码
	ErrCodeSyncSourceRequired = apperrors.ErrCodeSyncSourceRequired
	// ErrCodeSyncUnsupported 不支持的同步操作错误码
	ErrCodeSyncUnsupported = apperrors.ErrCodeSyncUnsupported
	// ErrCodeMarketUnavailable 市场不可用错误码
	ErrCodeMarketUnavailable = apperrors.ErrCodeMarketUnavailable
)

var (
	// ErrInvalidFundCode 无效基金代码错误实例
	ErrInvalidFundCode = apperrors.ErrInvalidFundCode
	// ErrFundNotFound 基金未找到错误实例
	ErrFundNotFound = apperrors.ErrFundNotFound
	// ErrInvalidStockCode 无效股票代码错误实例
	ErrInvalidStockCode = apperrors.ErrInvalidStockCode
	// ErrStockNotFound 股票未找到错误实例
	ErrStockNotFound = apperrors.ErrStockNotFound
	// ErrInvalidRankingType 无效排名类型错误实例
	ErrInvalidRankingType = apperrors.ErrInvalidRankingType
	// ErrSyncSourceRequired 同步源必填错误实例
	ErrSyncSourceRequired = apperrors.ErrSyncSourceRequired
	// ErrSyncUnsupported 不支持的同步操作错误实例
	ErrSyncUnsupported = apperrors.ErrSyncUnsupported
	// ErrMarketUnavailable 市场不可用错误实例
	ErrMarketUnavailable = apperrors.ErrMarketUnavailable
)

// NewAppError 创建新的应用错误实例。
func NewAppError(code int, message string, httpStatus int) *AppError {
	return apperrors.NewAppError(code, message, httpStatus)
}

// NewDetailCache 创建新的详情缓存实例。
func NewDetailCache(maxEntries int, ttl time.Duration) *DetailCache {
	return platformcache.NewDetailCache(maxEntries, ttl)
}
