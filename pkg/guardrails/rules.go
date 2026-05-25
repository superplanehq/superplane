package guardrails

// DefaultRules returns the full production rule set, sorted by priority.
func DefaultRules() []Rule {
	return []Rule{
		awsAccessKey(),
		gitHubPAT(),
		openAIKey(),
		connectionString(),
		jwtBearer(),
		genericHighEntropy(),
		roleDelimiterInjection(),
		instructionOverride(),
	}
}
