package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/superplanehq/superplane/pkg/lint/modelstxdebt"
)

const defaultBaselinePath = ".models-tx-debt-baseline.json"

type baseline struct {
	MaxAllowedInTransactionDefinitions int      `json:"maxAllowedInTransactionDefinitions"`
	MaxAllowedDatabaseConnCalls        int      `json:"maxAllowedDatabaseConnCalls"`
	InTransactionDefinitionKeys        []string `json:"inTransactionDefinitionKeys"`
	DatabaseConnCallKeys               []string `json:"databaseConnCallKeys"`
	UpdatedAt                          string   `json:"updatedAt"`
}

func main() {
	updateBaseline := flag.Bool("update-baseline", false, "write the current counts as the new baseline")
	rootDir := flag.String("root", modelstxdebt.DefaultRootDir, "directory to scan")
	baselinePath := flag.String("baseline", defaultBaselinePath, "baseline file path")
	flag.Parse()

	result, err := modelstxdebt.Scan(*rootDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "models tx debt scan failed: %v\n", err)
		os.Exit(1)
	}

	currentInTransaction := result.InTransactionDefinitionCount()
	currentDatabaseConn := result.DatabaseConnCallCount()

	if *updateBaseline {
		newBaseline := baseline{
			MaxAllowedInTransactionDefinitions: currentInTransaction,
			MaxAllowedDatabaseConnCalls:        currentDatabaseConn,
			InTransactionDefinitionKeys:        locationKeys(result.InTransactionDefinitions),
			DatabaseConnCallKeys:               locationKeys(result.DatabaseConnCalls),
			UpdatedAt:                          time.Now().UTC().Format(time.RFC3339),
		}

		if err := writeBaseline(*baselinePath, newBaseline); err != nil {
			fmt.Fprintf(os.Stderr, "failed to write models tx debt baseline: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf(
			"Updated models tx debt baseline to %d InTransaction definition(s) and %d database.Conn() call(s).\n",
			currentInTransaction,
			currentDatabaseConn,
		)
		printCountsVsBaseline(os.Stdout, currentInTransaction, currentDatabaseConn, newBaseline)
		fmt.Printf("WITHIN DEBT CAP %d/%d InTransaction, %d/%d database.Conn()\n",
			currentInTransaction, currentInTransaction,
			currentDatabaseConn, currentDatabaseConn,
		)
		return
	}

	existingBaseline, err := readBaseline(*baselinePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			fmt.Fprintf(os.Stderr, "models tx debt baseline file %s does not exist. Run with --update-baseline first.\n", *baselinePath)
			os.Exit(1)
		}

		fmt.Fprintf(os.Stderr, "failed to read models tx debt baseline: %v\n", err)
		os.Exit(1)
	}

	inTransactionRegression := currentInTransaction - existingBaseline.MaxAllowedInTransactionDefinitions
	databaseConnRegression := currentDatabaseConn - existingBaseline.MaxAllowedDatabaseConnCalls
	newInTransactionKeys := findNewKeys(locationKeys(result.InTransactionDefinitions), existingBaseline.InTransactionDefinitionKeys)
	newDatabaseConnKeys := findNewKeys(locationKeys(result.DatabaseConnCalls), existingBaseline.DatabaseConnCallKeys)
	resolvedInTransactionKeys := findResolvedKeys(locationKeys(result.InTransactionDefinitions), existingBaseline.InTransactionDefinitionKeys)
	resolvedDatabaseConnKeys := findResolvedKeys(locationKeys(result.DatabaseConnCalls), existingBaseline.DatabaseConnCallKeys)
	inTransactionImproved := inTransactionRegression < 0 || len(resolvedInTransactionKeys) > 0
	databaseConnImproved := databaseConnRegression < 0 || len(resolvedDatabaseConnKeys) > 0

	if inTransactionRegression > 0 || databaseConnRegression > 0 || len(newInTransactionKeys) > 0 || len(newDatabaseConnKeys) > 0 {
		fmt.Fprintln(os.Stderr, "Models tx debt exceeded in", *rootDir+".")
		printCountsVsBaseline(os.Stderr, currentInTransaction, currentDatabaseConn, existingBaseline)

		if len(newInTransactionKeys) > 0 {
			fmt.Fprintf(os.Stderr, "\nNew InTransaction definition(s):\n")
			printKeys(os.Stderr, newInTransactionKeys)
		}

		if len(newDatabaseConnKeys) > 0 {
			fmt.Fprintf(os.Stderr, "\nNew database.Conn() call(s):\n")
			printKeys(os.Stderr, newDatabaseConnKeys)
		}

		fmt.Fprintf(os.Stderr, "\n%s\n", modelstxdebt.Guidance)
		fmt.Fprintf(os.Stderr, "\nFAILED %d/%d InTransaction, %d/%d database.Conn()\n",
			currentInTransaction, existingBaseline.MaxAllowedInTransactionDefinitions,
			currentDatabaseConn, existingBaseline.MaxAllowedDatabaseConnCalls,
		)
		os.Exit(1)
	}

	if inTransactionImproved || databaseConnImproved {
		fmt.Fprintf(os.Stderr, "Models tx debt improved; update the baseline.\n")
		printCountsVsBaseline(os.Stderr, currentInTransaction, currentDatabaseConn, existingBaseline)
		if len(resolvedInTransactionKeys) > 0 {
			fmt.Fprintf(os.Stderr, "\nResolved InTransaction definition(s):\n")
			printKeys(os.Stderr, resolvedInTransactionKeys)
		}
		if len(resolvedDatabaseConnKeys) > 0 {
			fmt.Fprintf(os.Stderr, "\nResolved database.Conn() call(s):\n")
			printKeys(os.Stderr, resolvedDatabaseConnKeys)
		}
		fmt.Fprintf(os.Stderr, "Run: make check.models.tx.debt.baseline.update\n")
		os.Exit(1)
	}

	printCountsVsBaseline(os.Stdout, currentInTransaction, currentDatabaseConn, existingBaseline)
	fmt.Printf("WITHIN DEBT CAP %d/%d InTransaction, %d/%d database.Conn()\n",
		currentInTransaction, existingBaseline.MaxAllowedInTransactionDefinitions,
		currentDatabaseConn, existingBaseline.MaxAllowedDatabaseConnCalls,
	)
}

func readBaseline(path string) (baseline, error) {
	var stored baseline

	file, err := os.Open(path)
	if err != nil {
		return stored, err
	}
	defer file.Close()

	if err := json.NewDecoder(file).Decode(&stored); err != nil {
		return stored, fmt.Errorf("decode baseline: %w", err)
	}

	return stored, nil
}

func writeBaseline(path string, stored baseline) error {
	encoded, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return fmt.Errorf("encode baseline: %w", err)
	}

	encoded = append(encoded, '\n')
	return os.WriteFile(path, encoded, 0o644)
}

func printCountsVsBaseline(out io.Writer, currentInTransaction, currentDatabaseConn int, allowed baseline) {
	fmt.Fprintf(out, "- InTransaction definitions: %d (allowed %d)\n",
		currentInTransaction, allowed.MaxAllowedInTransactionDefinitions)
	fmt.Fprintf(out, "- database.Conn() calls: %d (allowed %d)\n",
		currentDatabaseConn, allowed.MaxAllowedDatabaseConnCalls)
}

func locationKeys(locations []modelstxdebt.Location) []string {
	keys := make([]string, len(locations))
	for i, location := range locations {
		keys[i] = location.Key()
	}
	return keys
}

func findNewKeys(currentKeys, baselineKeys []string) []string {
	baselineSet := make(map[string]struct{}, len(baselineKeys))
	for _, key := range baselineKeys {
		baselineSet[key] = struct{}{}
	}

	var newKeys []string
	for _, key := range currentKeys {
		if _, ok := baselineSet[key]; ok {
			continue
		}
		newKeys = append(newKeys, key)
	}

	return newKeys
}

func findResolvedKeys(currentKeys, baselineKeys []string) []string {
	currentSet := make(map[string]struct{}, len(currentKeys))
	for _, key := range currentKeys {
		currentSet[key] = struct{}{}
	}

	var resolvedKeys []string
	for _, key := range baselineKeys {
		if _, ok := currentSet[key]; ok {
			continue
		}
		resolvedKeys = append(resolvedKeys, key)
	}

	return resolvedKeys
}

func printKeys(out *os.File, keys []string) {
	for _, key := range keys {
		fmt.Fprintf(out, "  - %s\n", key)
	}
}
