package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"path"
	"sort"
	"strings"
	"time"
)

const (
	defaultProfilePath  = "coverage-go.out"
	defaultBaselinePath = ".go-coverage-baseline.json"
	modulePrefix        = "github.com/superplanehq/superplane/"
)

var defaultIgnoredPackagePrefixes = []string{
	"pkg/components",
	"pkg/integrations",
	"pkg/openapi_client",
	"pkg/protos/",
	"pkg/triggers",
	"pkg/web/assets",
}

type baseline struct {
	MinTotalCoverage       float64            `json:"minTotalCoverage"`
	MinCoverageByPackage   map[string]float64 `json:"minCoverageByPackage"`
	IgnoredPackagePrefixes []string           `json:"ignoredPackagePrefixes,omitempty"`
	UpdatedAt              string             `json:"updatedAt"`
}

type coverageStats struct {
	TotalCoverage     float64
	CoverageByPackage map[string]float64
}

type packageTotals struct {
	totalStatements   int
	coveredStatements int
}

type packageRegression struct {
	Package         string
	CurrentCoverage float64
	MinCoverage     float64
}

func main() {
	updateBaseline := flag.Bool("update-baseline", false, "write the current coverage as the new baseline")
	profilePath := flag.String("profile", defaultProfilePath, "coverage profile path")
	baselinePath := flag.String("baseline", defaultBaselinePath, "coverage baseline file path")
	flag.Parse()

	existingBaseline, err := readBaseline(*baselinePath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		fmt.Fprintf(os.Stderr, "failed to read Go coverage baseline: %v\n", err)
		os.Exit(1)
	}

	ignoredPackagePrefixes := defaultIgnoredPackagePrefixes
	if existingBaseline != nil && len(existingBaseline.IgnoredPackagePrefixes) > 0 {
		ignoredPackagePrefixes = existingBaseline.IgnoredPackagePrefixes
	}

	stats, err := readCoverageProfile(*profilePath, ignoredPackagePrefixes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to read Go coverage profile: %v\n", err)
		os.Exit(1)
	}

	if *updateBaseline {
		newBaseline := baseline{
			MinTotalCoverage:       stats.TotalCoverage,
			MinCoverageByPackage:   stats.CoverageByPackage,
			IgnoredPackagePrefixes: ignoredPackagePrefixes,
			UpdatedAt:              time.Now().UTC().Format(time.RFC3339),
		}

		if err := writeBaseline(*baselinePath, newBaseline); err != nil {
			fmt.Fprintf(os.Stderr, "failed to write Go coverage baseline: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Updated Go coverage baseline to total coverage %.1f%% across %d package(s).\n", stats.TotalCoverage, len(stats.CoverageByPackage))
		printCoverageVsBudget(stats, newBaseline)
		fmt.Printf("WITHIN BUDGET %.1f/%.1f\n", stats.TotalCoverage, stats.TotalCoverage)
		return
	}

	if existingBaseline == nil {
		fmt.Fprintf(os.Stderr, "Go coverage baseline file %s does not exist. Run with --update-baseline first.\n", *baselinePath)
		os.Exit(1)
	}

	totalRegression := roundCoverage(existingBaseline.MinTotalCoverage - stats.TotalCoverage)
	packageRegressions := findPackageRegressions(stats.CoverageByPackage, existingBaseline.MinCoverageByPackage)
	missingPackages := findMissingPackages(stats.CoverageByPackage, existingBaseline.MinCoverageByPackage)

	if totalRegression > 0 || len(packageRegressions) > 0 || len(missingPackages) > 0 {
		fmt.Fprintln(os.Stderr, "Go coverage budget exceeded.")
		fmt.Fprintf(os.Stderr, "- Total coverage: %.1f%% (allowed %.1f%%)\n", stats.TotalCoverage, existingBaseline.MinTotalCoverage)

		if len(packageRegressions) > 0 {
			fmt.Fprintln(os.Stderr, "- Package regressions:")
			for _, regression := range packageRegressions {
				fmt.Fprintf(os.Stderr, "  - %s: %.1f%% (allowed %.1f%%)\n", regression.Package, regression.CurrentCoverage, regression.MinCoverage)
			}
		}

		if len(missingPackages) > 0 {
			fmt.Fprintln(os.Stderr, "- Missing packages:")
			for _, pkg := range missingPackages {
				fmt.Fprintf(os.Stderr, "  - %s\n", pkg)
			}
		}

		printCoverageVsBudget(stats, *existingBaseline)
		fmt.Fprintf(os.Stderr, "FAILED %.1f/%.1f\n", stats.TotalCoverage, existingBaseline.MinTotalCoverage)
		os.Exit(1)
	}

	printCoverageVsBudget(stats, *existingBaseline)
	fmt.Printf("WITHIN BUDGET %.1f/%.1f\n", stats.TotalCoverage, existingBaseline.MinTotalCoverage)
}

func readCoverageProfile(profilePath string, ignoredPackagePrefixes []string) (coverageStats, error) {
	data, err := os.ReadFile(profilePath)
	if err != nil {
		return coverageStats{}, err
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) == 0 {
		return coverageStats{}, fmt.Errorf("coverage profile %s is empty", profilePath)
	}

	totalStatements := 0
	totalCoveredStatements := 0
	packageStats := map[string]packageTotals{}

	for index, line := range lines {
		if index == 0 {
			if !strings.HasPrefix(line, "mode:") {
				return coverageStats{}, fmt.Errorf("coverage profile %s is missing mode header", profilePath)
			}
			continue
		}

		fields := strings.Fields(line)
		if len(fields) != 3 {
			return coverageStats{}, fmt.Errorf("unexpected coverage line: %s", line)
		}

		fileWithRange := fields[0]
		statementCount, err := parseNonNegativeInt(fields[1])
		if err != nil {
			return coverageStats{}, fmt.Errorf("invalid statement count in line %q: %w", line, err)
		}

		executionCount, err := parseNonNegativeInt(fields[2])
		if err != nil {
			return coverageStats{}, fmt.Errorf("invalid execution count in line %q: %w", line, err)
		}

		filePath := strings.SplitN(fileWithRange, ":", 2)[0]
		packagePath := toPackagePath(filePath)

		totalStatements += statementCount
		if executionCount > 0 {
			totalCoveredStatements += statementCount
		}

		if shouldIgnorePackage(packagePath, ignoredPackagePrefixes) {
			continue
		}

		t := packageStats[packagePath]
		t.totalStatements += statementCount
		if executionCount > 0 {
			t.coveredStatements += statementCount
		}
		packageStats[packagePath] = t
	}

	coverageByPackage := map[string]float64{}
	for packagePath, t := range packageStats {
		if t.totalStatements == 0 {
			continue
		}

		coverageByPackage[packagePath] = roundCoverage(100 * float64(t.coveredStatements) / float64(t.totalStatements))
	}

	totalCoverage := 0.0
	if totalStatements > 0 {
		totalCoverage = roundCoverage(100 * float64(totalCoveredStatements) / float64(totalStatements))
	}

	return coverageStats{
		TotalCoverage:     totalCoverage,
		CoverageByPackage: coverageByPackage,
	}, nil
}

func parseNonNegativeInt(value string) (int, error) {
	parsed := 0
	for _, char := range value {
		if char < '0' || char > '9' {
			return 0, fmt.Errorf("non-numeric value %q", value)
		}
		parsed = parsed*10 + int(char-'0')
	}

	return parsed, nil
}

func toPackagePath(filePath string) string {
	trimmedPath := strings.TrimPrefix(filePath, modulePrefix)
	return path.Dir(trimmedPath)
}

func shouldIgnorePackage(packagePath string, ignoredPackagePrefixes []string) bool {
	for _, prefix := range ignoredPackagePrefixes {
		if strings.HasPrefix(packagePath, prefix) {
			return true
		}
	}

	return false
}

func readBaseline(baselinePath string) (*baseline, error) {
	raw, err := os.ReadFile(baselinePath)
	if err != nil {
		return nil, err
	}

	var parsed baseline
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}

	if parsed.MinCoverageByPackage == nil {
		parsed.MinCoverageByPackage = map[string]float64{}
	}

	return &parsed, nil
}

func writeBaseline(baselinePath string, data baseline) error {
	encoded, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(baselinePath, append(encoded, '\n'), 0o644)
}

func findPackageRegressions(currentByPackage map[string]float64, minByPackage map[string]float64) []packageRegression {
	regressions := make([]packageRegression, 0)

	for pkg, minCoverage := range minByPackage {
		currentCoverage, exists := currentByPackage[pkg]
		if !exists {
			continue
		}

		if roundCoverage(minCoverage-currentCoverage) > 0 {
			regressions = append(regressions, packageRegression{
				Package:         pkg,
				CurrentCoverage: currentCoverage,
				MinCoverage:     minCoverage,
			})
		}
	}

	sort.Slice(regressions, func(i, j int) bool {
		currentDrop := regressions[i].MinCoverage - regressions[i].CurrentCoverage
		otherDrop := regressions[j].MinCoverage - regressions[j].CurrentCoverage
		if currentDrop != otherDrop {
			return currentDrop > otherDrop
		}

		return regressions[i].Package < regressions[j].Package
	})

	return regressions
}

func findMissingPackages(currentByPackage map[string]float64, minByPackage map[string]float64) []string {
	missing := make([]string, 0)

	for pkg := range minByPackage {
		if _, exists := currentByPackage[pkg]; exists {
			continue
		}
		missing = append(missing, pkg)
	}

	sort.Strings(missing)
	return missing
}

func printCoverageVsBudget(stats coverageStats, budget baseline) {
	fmt.Println("Coverage vs budget:")
	fmt.Printf("- total: %.1f/%.1f\n", stats.TotalCoverage, budget.MinTotalCoverage)

	if len(budget.MinCoverageByPackage) == 0 {
		fmt.Println("- No package coverage found.")
		return
	}

	sortedPackages := mapsKeys(budget.MinCoverageByPackage)
	for _, pkg := range sortedPackages {
		currentCoverage := stats.CoverageByPackage[pkg]
		minCoverage := budget.MinCoverageByPackage[pkg]
		status := ""
		if roundCoverage(minCoverage-currentCoverage) > 0 {
			status = " !!! OVER BUDGET"
		}
		fmt.Printf("- %s: %.1f/%.1f%s\n", pkg, currentCoverage, minCoverage, status)
	}
}

func mapsKeys(values map[string]float64) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func roundCoverage(value float64) float64 {
	return math.Round(value*10) / 10
}
