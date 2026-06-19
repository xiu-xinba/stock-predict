package providers

import (
	"io"
	"net/http"
	"strings"
)

type resilientRoundTripFunc func(*http.Request) (*http.Response, error)

func (f resilientRoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func testResponse(status int, body string, headers map[string]string) *http.Response {
	resp := &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
	for key, value := range headers {
		resp.Header.Set(key, value)
	}
	return resp
}
