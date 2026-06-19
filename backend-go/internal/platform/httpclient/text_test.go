package httpclient

import (
	"math"
	"strings"
	"testing"
	"unicode/utf8"

	"golang.org/x/text/encoding/simplifiedchinese"
)

func TestClampNormal(t *testing.T) {
	if v := Clamp(5, 0, 10); v != 5 {
		t.Fatalf("expected 5, got %f", v)
	}
	if v := Clamp(-1, 0, 10); v != 0 {
		t.Fatalf("expected 0, got %f", v)
	}
	if v := Clamp(15, 0, 10); v != 10 {
		t.Fatalf("expected 10, got %f", v)
	}
}

func TestClampNaN(t *testing.T) {
	if v := Clamp(math.NaN(), 0, 10); v != 0 {
		t.Fatalf("expected 0 for NaN, got %f", v)
	}
}

func TestClampInf(t *testing.T) {
	if v := Clamp(math.Inf(1), 0, 10); v != 0 {
		t.Fatalf("expected 0 for +Inf, got %f", v)
	}
	if v := Clamp(math.Inf(-1), 0, 10); v != 0 {
		t.Fatalf("expected 0 for -Inf, got %f", v)
	}
}

func TestClampBoundary(t *testing.T) {
	if v := Clamp(0, 0, 10); v != 0 {
		t.Fatalf("expected 0 at lower bound, got %f", v)
	}
	if v := Clamp(10, 0, 10); v != 10 {
		t.Fatalf("expected 10 at upper bound, got %f", v)
	}
}

func TestRoundValZeroPlaces(t *testing.T) {
	if v := RoundVal(3.14159, 0); v != 3 {
		t.Fatalf("expected 3, got %f", v)
	}
}

func TestRoundValTwoPlaces(t *testing.T) {
	if v := RoundVal(3.14159, 2); v != 3.14 {
		t.Fatalf("expected 3.14, got %f", v)
	}
}

func TestRoundValFourPlaces(t *testing.T) {
	if v := RoundVal(3.14159265, 4); v != 3.1416 {
		t.Fatalf("expected 3.1416, got %f", v)
	}
}

func TestIsAllDigitsEmpty(t *testing.T) {
	if IsAllDigits("") {
		t.Fatalf("expected false for empty string")
	}
}

func TestIsAllDigitsPureDigits(t *testing.T) {
	if !IsAllDigits("123456") {
		t.Fatalf("expected true for pure digits")
	}
}

func TestIsAllDigitsWithLetter(t *testing.T) {
	if IsAllDigits("123a56") {
		t.Fatalf("expected false for string with letter")
	}
}

func TestIsAllDigitsWithSpace(t *testing.T) {
	if IsAllDigits("123 56") {
		t.Fatalf("expected false for string with space")
	}
}

func TestParseQuoteFloatNormal(t *testing.T) {
	if v := ParseQuoteFloat("3.14"); v != 3.14 {
		t.Fatalf("expected 3.14, got %f", v)
	}
}

func TestParseQuoteFloatCommaSeparated(t *testing.T) {
	if v := ParseQuoteFloat("1,234.56"); v != 1234.56 {
		t.Fatalf("expected 1234.56, got %f", v)
	}
}

func TestParseQuoteFloatPercent(t *testing.T) {
	if v := ParseQuoteFloat("2.5%"); v != 2.5 {
		t.Fatalf("expected 2.5, got %f", v)
	}
}

func TestParseQuoteFloatEmpty(t *testing.T) {
	if v := ParseQuoteFloat(""); v != 0 {
		t.Fatalf("expected 0 for empty string, got %f", v)
	}
}

func TestParseQuoteFloatDoubleDash(t *testing.T) {
	if v := ParseQuoteFloat("--"); v != 0 {
		t.Fatalf("expected 0 for --, got %f", v)
	}
}

func TestParseQuoteFloatTripleDash(t *testing.T) {
	if v := ParseQuoteFloat("---"); v != 0 {
		t.Fatalf("expected 0 for ---, got %f", v)
	}
}

func TestEnsureUTF8AlreadyUTF8(t *testing.T) {
	input := []byte("hello 世界")
	result := EnsureUTF8(input)
	if string(result) != "hello 世界" {
		t.Fatalf("expected 'hello 世界', got '%s'", string(result))
	}
}

func TestEnsureUTF8InvalidUTF8(t *testing.T) {
	// Generate correct GBK bytes for "航空航天"
	encoder := simplifiedchinese.GBK.NewEncoder()
	gbk, err := encoder.Bytes([]byte("航空航天"))
	if err != nil {
		t.Fatalf("failed to encode test string as GBK: %v", err)
	}
	if utf8.Valid(gbk) {
		t.Log("Note: GBK bytes happen to be valid UTF-8, using different test path")
	}
	result := EnsureUTF8(gbk)
	if !utf8.Valid(result) {
		t.Fatalf("expected valid UTF-8, got invalid")
	}
	if string(result) != "航空航天" {
		t.Fatalf("expected '航空航天', got '%s'", string(result))
	}
}

func TestEnsureUTF8GBKInJSON(t *testing.T) {
	// Simulate eastmoney API response with GBK-encoded Chinese in JSON
	// GBK for {"name":"电池"} where 电池 = B5 E7 B3 D8
	gbkJSON := []byte{
		0x7B, 0x22, 0x6E, 0x61, 0x6D, 0x65, 0x22, 0x3A, 0x22, // {"name":"
		0xB5, 0xE7, 0xB3, 0xD8, // 电池 in GBK
		0x22, 0x7D, // "}
	}
	result := EnsureUTF8(gbkJSON)
	if !utf8.Valid(result) {
		t.Fatalf("expected valid UTF-8, got invalid")
	}
	expected := `{"name":"电池"}`
	if string(result) != expected {
		t.Fatalf("expected '%s', got '%s'", expected, string(result))
	}
}

func TestEnsureUTF8GBKValidUTF8Sequence(t *testing.T) {
	// GBK data that happens to form valid UTF-8 sequences (the hard case).
	// Generate a realistic JSON-like payload with multiple GBK Chinese chars.
	encoder := simplifiedchinese.GBK.NewEncoder()
	chineseGBK, err := encoder.Bytes([]byte("行业板块"))
	if err != nil {
		t.Fatalf("failed to encode: %v", err)
	}
	gbkData := make([]byte, 0, 64)
	gbkData = append(gbkData, `{"f14":"`...)
	gbkData = append(gbkData, chineseGBK...)
	gbkData = append(gbkData, `","f3":1}`...)

	result := EnsureUTF8(gbkData)
	if !utf8.Valid(result) {
		t.Fatalf("expected valid UTF-8, got invalid")
	}
	resultStr := string(result)
	if !strings.Contains(resultStr, "行业板块") {
		t.Fatalf("expected result to contain '行业板块', got '%s'", resultStr)
	}
}

func TestEnsureUTF8PureUTF8JSON(t *testing.T) {
	// Real UTF-8 JSON with Chinese - must NOT be falsely detected as GBK
	input := []byte(`{"f14":"航空航天","f3":3.08}`)
	result := EnsureUTF8(input)
	if string(result) != string(input) {
		t.Fatalf("UTF-8 JSON should not be modified, got '%s'", string(result))
	}
}

func TestLooksLikeGBK(t *testing.T) {
	encoder := simplifiedchinese.GBK.NewEncoder()
	// GBK data with many high-bit pairs but no 3-byte UTF-8 CJK sequences
	gbkChinese, err := encoder.Bytes([]byte("行业板块航空航天"))
	if err != nil {
		t.Fatalf("failed to encode: %v", err)
	}
	if !looksLikeGBK(gbkChinese) {
		t.Fatal("expected GBK data to be detected by looksLikeGBK")
	}
	// Pure UTF-8 Chinese - should NOT be detected as GBK
	utf8Data := []byte("航空航天行业板块触摸屏电池")
	if looksLikeGBK(utf8Data) {
		t.Fatal("UTF-8 Chinese should NOT be detected as GBK")
	}
	// Pure ASCII - should NOT be detected as GBK
	if looksLikeGBK([]byte("hello world")) {
		t.Fatal("ASCII should NOT be detected as GBK")
	}
}
