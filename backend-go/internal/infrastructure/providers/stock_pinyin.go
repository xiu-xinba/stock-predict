package providers

import (
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/text/encoding/simplifiedchinese"
)

// toNum 将任意类型转换为 float64，支持 float64、string、json.Number。
func toNum(v any) float64 {
	switch n := v.(type) {
	case float64:
		if n < -1e15 || n > 1e15 {
			return 0
		}
		return n
	case string:
		if n == "" || n == "-" {
			return 0
		}
		f, err := strconv.ParseFloat(n, 64)
		if err != nil {
			return 0
		}
		if f < -1e15 || f > 1e15 {
			return 0
		}
		return f
	case json.Number:
		s := n.String()
		if s == "" || s == "-" {
			return 0
		}
		f, err := n.Float64()
		if err != nil {
			return 0
		}
		if f < -1e15 || f > 1e15 {
			return 0
		}
		return f
	default:
		return 0
	}
}

// polyphoneOverrides 多音字拼音首字母覆盖表。
var polyphoneOverrides = map[rune][]string{
	0x884C: {"H", "X"},
	0x91CD: {"Z", "C"},
	0x957F: {"C", "Z"},
	0x4E50: {"L", "Y"},
	0x53C2: {"C", "S"},
	0x5355: {"D", "S"},
}

// pinyinAbbr 生成名称的拼音首字母缩写，多音字取第一个读音。
func pinyinAbbr(name string) string {
	var abbr strings.Builder
	for _, r := range name {
		if r >= 0x4e00 && r <= 0x9fff {
			if overrides, ok := polyphoneOverrides[r]; ok {
				abbr.WriteString(overrides[0])
				continue
			}
			initial := pinyinInitial(r)
			if initial != "" {
				abbr.WriteString(initial)
			}
		} else if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			abbr.WriteRune(r)
		}
	}
	return strings.ToLower(abbr.String())
}

// pinyinAbbrAll 生成名称的所有拼音首字母组合，包含多音字的所有读音变体。
func pinyinAbbrAll(name string) []string {
	type polyPos struct {
		idx  int
		alts []string
	}
	var polys []polyPos

	runes := []rune(name)
	for i, r := range runes {
		if r >= 0x4e00 && r <= 0x9fff {
			if overrides, ok := polyphoneOverrides[r]; ok {
				polys = append(polys, polyPos{idx: i, alts: overrides})
			}
		}
	}

	if len(polys) == 0 {
		return []string{pinyinAbbr(name)}
	}

	var baseRunes []rune
	for _, r := range runes {
		if r >= 0x4e00 && r <= 0x9fff {
			if overrides, ok := polyphoneOverrides[r]; ok {
				baseRunes = append(baseRunes, []rune(strings.ToLower(overrides[0]))...)
			} else {
				initial := pinyinInitial(r)
				if initial != "" {
					baseRunes = append(baseRunes, []rune(strings.ToLower(initial))...)
				}
			}
		} else if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			baseRunes = append(baseRunes, r)
		}
	}

	base := string(baseRunes)
	results := []string{base}

	for _, p := range polys {
		charIdx := 0
		for i := 0; i < p.idx; i++ {
			r := runes[i]
			if r >= 0x4e00 && r <= 0x9fff {
				if ov, ok := polyphoneOverrides[r]; ok {
					charIdx += len(ov[0])
				} else {
					init := pinyinInitial(r)
					if init != "" {
						charIdx++
					}
				}
			} else if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
				charIdx++
			}
		}

		for _, alt := range p.alts[1:] {
			newRunes := make([]rune, len(baseRunes))
			copy(newRunes, baseRunes)
			altLower := []rune(strings.ToLower(alt))
			if charIdx+len(altLower) <= len(newRunes) {
				for j, ar := range altLower {
					newRunes[charIdx+j] = ar
				}
				results = append(results, string(newRunes))
			}
		}
	}

	return results
}

// pinyinInitialTable GBK 编码区间与拼音首字母的映射表。
var pinyinInitialTable = []struct {
	code    int
	initial string
}{
	{45217, "A"}, {45253, "B"}, {45761, "C"}, {46318, "D"},
	{46826, "E"}, {47010, "F"}, {47297, "G"}, {47614, "H"},
	{48119, "J"}, {49062, "K"}, {49324, "L"}, {49896, "M"},
	{50371, "N"}, {50614, "O"}, {50622, "P"}, {50906, "Q"},
	{51387, "R"}, {51446, "S"}, {52218, "T"}, {52698, "W"},
	{52980, "X"}, {53689, "Y"}, {54481, "Z"},
}

// pinyinInitial 根据汉字的 GBK 编码查表获取拼音首字母。
func pinyinInitial(r rune) string {
	if r < 0x4E00 || r > 0x9FFF {
		return ""
	}
	gbBytes, err := simplifiedchinese.GBK.NewEncoder().Bytes([]byte(string(r)))
	if err != nil || len(gbBytes) != 2 {
		return ""
	}
	gbCode := int(gbBytes[0])<<8 | int(gbBytes[1])
	if gbCode < pinyinInitialTable[0].code || gbCode > 55289 {
		return ""
	}
	i := sort.Search(len(pinyinInitialTable), func(i int) bool {
		return pinyinInitialTable[i].code > gbCode
	})
	if i == 0 {
		return ""
	}
	return pinyinInitialTable[i-1].initial
}
