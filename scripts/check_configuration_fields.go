package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/superplanehq/superplane/pkg/lint/configurationfields"
)

const defaultBaselinePath = ".configuration-fields-baseline.txt"

func main() {
	updateBaseline := flag.Bool("update-baseline", false, "write the current violations as the new baseline")
	baselinePath := flag.String("baseline", defaultBaselinePath, "baseline file path")
	flag.Parse()

	issues, err := configurationfields.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "configuration field lint failed: %v\n", err)
		os.Exit(1)
	}

	currentKeys := make([]string, 0, len(issues))
	currentSet := make(map[string]configurationfields.Issue, len(issues))
	for _, issue := range issues {
		key := issue.Key()
		currentKeys = append(currentKeys, key)
		currentSet[key] = issue
	}
	sort.Strings(currentKeys)

	if *updateBaseline {
		if err := writeBaseline(*baselinePath, currentKeys); err != nil {
			fmt.Fprintf(os.Stderr, "failed to write configuration field baseline: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Updated configuration field baseline with %d violation(s).\n", len(currentKeys))
		return
	}

	baselineKeys, err := readBaseline(*baselinePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(os.Stderr, "configuration field baseline file %s does not exist. Run with --update-baseline first.\n", *baselinePath)
			os.Exit(1)
		}

		fmt.Fprintf(os.Stderr, "failed to read configuration field baseline: %v\n", err)
		os.Exit(1)
	}

	baselineSet := make(map[string]struct{}, len(baselineKeys))
	for _, key := range baselineKeys {
		baselineSet[key] = struct{}{}
	}

	var failed bool
	for _, key := range currentKeys {
		if _, ok := baselineSet[key]; ok {
			continue
		}

		fmt.Fprintln(os.Stderr, currentSet[key].String())
		failed = true
	}

	var resolved []string
	for key := range baselineSet {
		if _, ok := currentSet[key]; ok {
			continue
		}
		resolved = append(resolved, key)
	}
	sort.Strings(resolved)

	if len(resolved) > 0 {
		fmt.Fprintf(os.Stderr, "configuration field baseline contains %d resolved violation(s). Run with --update-baseline to refresh.\n", len(resolved))
		for _, key := range resolved {
			fmt.Fprintf(os.Stderr, "- %s\n", key)
		}
		failed = true
	}

	if failed {
		fmt.Fprintf(os.Stderr, "\nComponent and integration configuration fields must use camelCase. Existing violations are tracked in %s.\n", *baselinePath)
		os.Exit(1)
	}
}

func readBaseline(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var keys []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		keys = append(keys, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return keys, nil
}

func writeBaseline(path string, keys []string) error {
	var builder strings.Builder
	builder.WriteString("# Configuration fields on components, triggers, widgets, and integrations must use camelCase.\n")
	for _, key := range keys {
		builder.WriteString(key)
		builder.WriteByte('\n')
	}

	return os.WriteFile(path, []byte(builder.String()), 0o644)
}
