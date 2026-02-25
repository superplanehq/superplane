package getdata

func (c *GetData) ExampleOutput() map[string]any {
	return map[string]any{
		"key":    "incident_id",
		"value":  "INC-1234",
		"exists": true,
	}
}
