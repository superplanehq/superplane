package loop

const exampleTimestamp = "2026-01-16T17:56:16.680755501Z"

func exampleOutput() map[string]any {
	return map[string]any{
		"type":      PayloadTypeDone,
		"data":      donePayload(3, StopReasonConditionMet, 4521),
		"timestamp": exampleTimestamp,
	}
}
