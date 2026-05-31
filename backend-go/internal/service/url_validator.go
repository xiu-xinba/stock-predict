package service

import (
	"net/url"
	"strings"
)

func isAllowedURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	for _, suffix := range []string{".eastmoney.com", ".qq.com"} {
		if strings.HasSuffix(host, suffix) || host == strings.TrimPrefix(suffix, ".") {
			return true
		}
	}
	for _, exact := range []string{"push2.eastmoney.com", "push2his.eastmoney.com", "qt.gtimg.cn"} {
		if host == exact {
			return true
		}
	}
	return false
}
