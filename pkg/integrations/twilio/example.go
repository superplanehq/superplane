package twilio

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

func getExampleOutput(name string) map[string]any {
	return loadJSON(fmt.Sprintf("example_output_%s.json", name))
}

func getExampleData(name string) map[string]any {
	return loadJSON(fmt.Sprintf("example_data_%s.json", name))
}

func loadJSON(filename string) map[string]any {
	_, currentFile, _, _ := runtime.Caller(0)
	dir := filepath.Dir(currentFile)
	data, err := os.ReadFile(filepath.Join(dir, filename))
	if err != nil {
		return map[string]any{}
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return map[string]any{}
	}
	return result
}
