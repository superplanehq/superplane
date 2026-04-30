package workers

func processedCount(values ...bool) int {
	count := 0
	for _, value := range values {
		if value {
			count++
		}
	}

	return count
}
