package guardrails

import "context"

// NoOpClassifier is the default Classifier used when no LLM API is configured.
// It intentionally skips every job so the worker marks them as "skipped" rather
// than leaving them stuck in "pending". Swap this out in Phase 5 for a real
// implementation that calls the Anthropic API.
type NoOpClassifier struct{}

func NewNoOpClassifier() *NoOpClassifier { return &NoOpClassifier{} }

func (c *NoOpClassifier) Model() string { return "noop" }

func (c *NoOpClassifier) Classify(_ context.Context, _ ClassificationRequest) (*ClassificationResult, error) {
	return nil, nil
}
