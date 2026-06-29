package console

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// resolveYAMLSource picks the YAML source for `console set`, applying the
// precedence rules:
//
//  1. The `-f/--file` flag (`"-"` means stdin).
//  2. A positional file argument after the canvas name.
//  3. Stdin when piped without an explicit `-`.
//
// It returns the YAML bytes and a human-friendly label for the source
// (used in error messages) so callers can attribute failures clearly.
func resolveYAMLSource(stdin io.Reader, flagValue string, positional string) ([]byte, string, error) {
	flagValue = strings.TrimSpace(flagValue)
	positional = strings.TrimSpace(positional)

	if flagValue != "" && positional != "" && flagValue != positional {
		return nil, "", fmt.Errorf("provide YAML either via --file or as a positional argument, not both")
	}

	switch {
	case flagValue == "-" || positional == "-":
		data, err := readStdin(stdin)
		if err != nil {
			return nil, "", err
		}
		return data, "stdin", nil

	case flagValue != "":
		data, err := readFile(flagValue)
		if err != nil {
			return nil, "", err
		}
		return data, flagValue, nil

	case positional != "":
		data, err := readFile(positional)
		if err != nil {
			return nil, "", err
		}
		return data, positional, nil
	}

	return nil, "", fmt.Errorf("no YAML source provided (use --file <path>, --file -, a positional file path, or pipe via stdin with -)")
}

func readFile(path string) ([]byte, error) {
	// #nosec G304 -- the path is provided by the operator on the command line.
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read %q: %w", path, err)
	}
	return data, nil
}

func readStdin(stdin io.Reader) ([]byte, error) {
	if stdin == nil {
		stdin = os.Stdin
	}
	data, err := io.ReadAll(stdin)
	if err != nil {
		return nil, fmt.Errorf("failed to read stdin: %w", err)
	}
	return data, nil
}
