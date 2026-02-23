package plugins

import (
	"io"
	"net/http"
	"strings"
)

func newHTTPRequest(method, url, body string, headers map[string]string) (*http.Request, error) {
	var bodyReader io.Reader
	if body != "" {
		bodyReader = strings.NewReader(body)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	return req, nil
}

func readResponseBody(resp *http.Response) (any, error) {
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return string(data), nil
}

func flattenHeaders(h http.Header) map[string]string {
	result := make(map[string]string, len(h))
	for k, v := range h {
		result[strings.ToLower(k)] = strings.Join(v, ", ")
	}
	return result
}
