package gitlab

import "net/http"

func gitlabHeaders(event, token string) http.Header {
	headers := http.Header{}
	if event != "" {
		headers.Set("X-Gitlab-Event", event)
	}

	if token != "" {
		headers.Set("X-Gitlab-Token", token)
	}

	return headers
}
