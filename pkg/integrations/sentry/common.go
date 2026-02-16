package sentry

type NodeMetadata struct{}

type SentryAppMetadata struct {
	Slug         string `json:"slug"`
	ClientSecret string `json:"clientSecret"`
}
