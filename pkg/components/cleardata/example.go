package cleardata

func (c *ClearData) ExampleOutput() map[string]any {
	return map[string]any{
		"key":          "pr_sandboxes",
		"exists":       true,
		"removed":      true,
		"removedCount": 1,
	}
}
