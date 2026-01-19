package utils

import (
	"encoding/json"
	"sync"
)

func UnmarshalEmbeddedJSON(once *sync.Once, data []byte, target *map[string]any) map[string]any {
	once.Do(func() {
		*target = map[string]any{}
		_ = json.Unmarshal(data, target)
	})

	return *target
}
