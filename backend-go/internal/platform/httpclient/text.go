// Package httpclient 提供了带重试、熔断和限流策略的 HTTP 客户端封装。
package httpclient

import (
	"bytes"
	"io"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// Clamp 将浮点值限制在 [min, max] 范围内，NaN 和 Inf 返回 0。
func Clamp(v, min, max float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return math.Min(math.Max(v, min), max)
}

// RoundVal 将浮点值四舍五入到指定小数位数。
func RoundVal(v float64, places int) float64 {
	pow := math.Pow10(places)
	return math.Round(v*pow) / pow
}

// IsAllDigits 判断字符串是否全部由数字字符组成。
func IsAllDigits(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}

// ParseQuoteFloat 解析行情数据中的浮点数字符串，自动去除逗号、百分号和空白，
// 无效值（空串、"--"等）返回 0。
func ParseQuoteFloat(raw string) float64 {
	raw = strings.TrimSpace(strings.TrimSuffix(strings.ReplaceAll(raw, ",", ""), "%"))
	if raw == "" || raw == "--" || raw == "---" {
		return 0
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0
	}
	return value
}

// EnsureUTF8 检测 GBK 编码内容并转码为 UTF-8。
// 中文金融 API（如东方财富、新浪）常返回 GBK 编码的 JSON，
// 但 Content-Type 头声称 charset=UTF-8。
// 检测策略：
//  1. 若数据不是合法 UTF-8，则按 GBK 解码（确定性判断）。
//  2. 若数据是合法 UTF-8，则使用统计检测：在 UTF-8 中，
//     中文字符占 3 字节（0xE4-0xE9 + 2 个续字节）；
//     在 GBK 中占 2 字节（0x81-0xFE + 0x40-0xFE）。
//     若数据有大量高位字节对但缺少 3 字节 UTF-8 CJK 序列，
//     则很可能是被误读为 UTF-8 的 GBK 编码。
func EnsureUTF8(data []byte) []byte {
	if len(data) == 0 {
		return data
	}
	if !utf8.Valid(data) {
		if decoded, err := decodeGBK(data); err == nil {
			return decoded
		}
		return data
	}
	// Data is valid UTF-8, but might be GBK misread as UTF-8.
	if looksLikeGBK(data) {
		if decoded, err := decodeGBK(data); err == nil {
			return decoded
		}
	}
	return data
}

func decodeGBK(data []byte) ([]byte, error) {
	return io.ReadAll(transform.NewReader(
		bytes.NewReader(data), simplifiedchinese.GBK.NewDecoder(),
	))
}

// DecodeGBK 强制将 GBK 编码数据转码为 UTF-8，适用于已知始终返回 GBK 编码的 API（如东方财富）。
func DecodeGBK(data []byte) []byte {
	decoded, err := decodeGBK(data)
	if err != nil {
		return data
	}
	return decoded
}

// looksLikeGBK 使用统计分析检测恰好构成合法（但错误的）UTF-8 序列的 GBK 编码内容。
// 在 UTF-8 中，每个 CJK 字符是 3 字节序列（起始字节 0xE4-0xE9）；
// 在 GBK 中，每个 CJK 字符是 2 字节序列。若数据包含大量高位字节对
// 但缺少 3 字节 UTF-8 CJK 序列，则数据很可能是 GBK 而非 UTF-8。
func looksLikeGBK(data []byte) bool {
	var utf8CJK3 int  // count of 3-byte UTF-8 CJK sequences
	var highPairs int // count of 2-byte high-bit pairs (potential GBK)
	i := 0
	for i < len(data) {
		b := data[i]
		if b < 0x80 {
			i++
			continue
		}
		// Check for 3-byte UTF-8 CJK: 0xE4-0xE9 followed by two 0x80-0xBF
		if b >= 0xE4 && b <= 0xE9 && i+2 < len(data) &&
			data[i+1] >= 0x80 && data[i+1] <= 0xBF &&
			data[i+2] >= 0x80 && data[i+2] <= 0xBF {
			utf8CJK3++
			i += 3
			continue
		}
		// Check for 2-byte high-bit pair (GBK pattern: 0x81-0xFE + 0x40-0xFE)
		if b >= 0x81 && b <= 0xFE && i+1 < len(data) &&
			((data[i+1] >= 0x40 && data[i+1] <= 0x7E) ||
				(data[i+1] >= 0x80 && data[i+1] <= 0xFE)) {
			highPairs++
			i += 2
			continue
		}
		i++
	}
	// If there are significant high-bit pairs but few 3-byte UTF-8 CJK
	// sequences, the data is likely GBK. We require at least 4 high pairs
	// and a ratio where GBK pairs outnumber UTF-8 CJK by 2:1 or more.
	return highPairs >= 4 && highPairs > utf8CJK3*2
}
