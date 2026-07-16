package utils

import (
	"encoding/json"
	"sync"
)

type EmbeddedJSON struct {
	once  sync.Once
	data  []byte
	value map[string]any
}

func NewEmbeddedJSON(data []byte) *EmbeddedJSON {
	return &EmbeddedJSON{data: data}
}

func (j *EmbeddedJSON) Value() map[string]any {
	j.once.Do(func() {
		j.value = map[string]any{}
		_ = json.Unmarshal(j.data, &j.value)
	})

	return j.value
}
