// Package errors 定义了应用程序级别的错误类型和错误码。
package errors

import "net/http"

const (
	ErrCodeInvalidFundCode    = 10001 // 无效的基金代码
	ErrCodeFundNotFound       = 10002 // 基金未找到
	ErrCodeInvalidStockCode   = 10003 // 无效的股票代码
	ErrCodeStockNotFound      = 10004 // 股票未找到
	ErrCodeInvalidRankingType = 10005 // 无效的排名类型
	ErrCodeSyncSourceRequired = 10006 // 同步数据源未指定
	ErrCodeSyncUnsupported    = 10007 // 仓库不支持同步操作
	ErrCodeMarketUnavailable  = 10008 // 市场数据不可用
)

// AppError 是应用程序统一的错误类型，包含业务错误码、错误消息和对应的 HTTP 状态码。
type AppError struct {
	Code       int    // 业务错误码，如 ErrCodeInvalidFundCode
	Message    string // 错误描述消息
	HTTPStatus int    // 对应的 HTTP 响应状态码
}

// NewAppError 创建一个新的 AppError 实例。
func NewAppError(code int, message string, httpStatus int) *AppError {
	return &AppError{
		Code:       code,
		Message:    message,
		HTTPStatus: httpStatus,
	}
}

// Error 实现 error 接口，返回错误消息。
func (e *AppError) Error() string {
	return e.Message
}

// Is 实现 errors.Is 接口，通过错误码判断两个 AppError 是否等价。
func (e *AppError) Is(target error) bool {
	t, ok := target.(*AppError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

var (
	ErrInvalidFundCode    = NewAppError(ErrCodeInvalidFundCode, "invalid fund code", http.StatusBadRequest)          // 无效的基金代码
	ErrFundNotFound       = NewAppError(ErrCodeFundNotFound, "fund not found", http.StatusNotFound)                   // 基金未找到
	ErrInvalidStockCode   = NewAppError(ErrCodeInvalidStockCode, "invalid stock code", http.StatusBadRequest)         // 无效的股票代码
	ErrStockNotFound      = NewAppError(ErrCodeStockNotFound, "stock not found", http.StatusNotFound)                  // 股票未找到
	ErrInvalidRankingType = NewAppError(ErrCodeInvalidRankingType, "invalid ranking type", http.StatusBadRequest)     // 无效的排名类型
	ErrSyncSourceRequired = NewAppError(ErrCodeSyncSourceRequired, "fund sync source is required", http.StatusBadRequest) // 同步数据源未指定
	ErrSyncUnsupported    = NewAppError(ErrCodeSyncUnsupported, "fund repository does not support sync", http.StatusInternalServerError) // 仓库不支持同步操作
	ErrMarketUnavailable  = NewAppError(ErrCodeMarketUnavailable, "market data unavailable", http.StatusServiceUnavailable) // 市场数据不可用
)
