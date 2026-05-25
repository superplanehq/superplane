package guardrails

import "context"

// ClassificationRequest carries scan output to the LLM classifier.
type ClassificationRequest struct {
	// Findings from the rule engine for this scan.
	Findings []Finding
	// ScanContext for org/workflow/node scoping.
	ScanContext ScanContext
	// ContentHash of the scanned prompt (SHA-256 hex). The raw content is NOT
	// included to avoid storing sensitive material; the classifier operates on
	// the structured findings metadata.
	ContentHash string
}

// ClassificationResult is the output from an LLM classifier.
type ClassificationResult struct {
	Model       string
	RiskScore   int
	Findings    []Finding
	RawResponse string
	TokenCount  int
	LatencyMs   int
}

// Classifier performs deep semantic analysis of scan findings.
// Implementations should be safe to call concurrently.
type Classifier interface {
	// Classify analyses the request and returns a classification result.
	// A nil result with no error means the job was intentionally skipped.
	Classify(ctx context.Context, req ClassificationRequest) (*ClassificationResult, error)
	// Model returns the identifier of the model/implementation.
	Model() string
}
