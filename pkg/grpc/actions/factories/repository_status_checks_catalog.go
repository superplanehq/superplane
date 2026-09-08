package factories

import (
	"net/url"
	"strings"
)

type observedRepositoryStatusCheck struct {
	Name       string
	DetailsURL string
}

type repositoryStatusCheck struct {
	Name                 string
	Required             bool
	DetailsURL           string
	SuggestedIntegration string
}

func suggestIntegrationFromCheckURL(rawURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Host == "" {
		return ""
	}

	host := strings.ToLower(parsed.Host)
	switch {
	case hostHasSuffix(host, "semaphoreci.com"), hostHasSuffix(host, "semaphore.com"):
		return "semaphore"
	case hostHasSuffix(host, "circleci.com"):
		return "circleci"
	case hostHasSuffix(host, "harness.io"):
		return "harness"
	default:
		return ""
	}
}

func hostHasSuffix(host, suffix string) bool {
	return host == suffix || strings.HasSuffix(host, "."+suffix)
}

func mergeRepositoryStatusChecks(required []string, observed []observedRepositoryStatusCheck) []repositoryStatusCheck {
	index := map[string]int{}
	checks := make([]repositoryStatusCheck, 0, len(required)+len(observed))

	for _, name := range required {
		appendRepositoryStatusCheck(&checks, index, name, true, "")
	}
	for _, item := range observed {
		appendRepositoryStatusCheck(&checks, index, item.Name, false, item.DetailsURL)
	}

	return checks
}

func appendRepositoryStatusCheck(
	checks *[]repositoryStatusCheck,
	index map[string]int,
	name string,
	required bool,
	detailsURL string,
) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}

	key := strings.ToLower(name)
	if existing, ok := index[key]; ok {
		row := &(*checks)[existing]
		if required {
			row.Required = true
		}
		if row.DetailsURL == "" {
			row.DetailsURL = strings.TrimSpace(detailsURL)
			row.SuggestedIntegration = suggestIntegrationFromCheckURL(row.DetailsURL)
		}
		return
	}

	trimmedURL := strings.TrimSpace(detailsURL)
	index[key] = len(*checks)
	*checks = append(*checks, repositoryStatusCheck{
		Name:                 name,
		Required:             required,
		DetailsURL:           trimmedURL,
		SuggestedIntegration: suggestIntegrationFromCheckURL(trimmedURL),
	})
}
