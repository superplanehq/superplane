package gocoverage_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/lint/gocoverage"
)

func TestNormalizeBlocksCollapsesGo127DuplicatedStatementCounts(t *testing.T) {
	blocks := []gocoverage.Block{
		{File: "root.go", StartLine: 9, EndLine: 13, Statements: 37},
		{File: "root.go", StartLine: 15, EndLine: 19, Statements: 37},
		{File: "root.go", StartLine: 20, EndLine: 21, Statements: 37},
		{File: "list.go", StartLine: 11, EndLine: 20, Statements: 8, Covered: true},
	}

	normalized := gocoverage.NormalizeBlocks(blocks)

	require.Equal(t, []gocoverage.Block{
		{File: "root.go", StartLine: 9, EndLine: 13, Statements: 37},
		{File: "list.go", StartLine: 11, EndLine: 20, Statements: 8, Covered: true},
	}, normalized)
}

func TestNormalizeBlocksKeepsSeparateRegionWithTheSameStatementCount(t *testing.T) {
	blocks := []gocoverage.Block{
		{File: "root.go", StartLine: 9, EndLine: 11, Statements: 5},
		{File: "root.go", StartLine: 12, EndLine: 20, Statements: 5},
		{File: "root.go", StartLine: 40, EndLine: 42, Statements: 5},
		{File: "root.go", StartLine: 43, EndLine: 44, Statements: 5},
		{File: "root.go", StartLine: 50, EndLine: 50, Statements: 5},
		{File: "list.go", StartLine: 11, EndLine: 20, Statements: 8, Covered: true},
	}

	normalized := gocoverage.NormalizeBlocks(blocks)

	require.Equal(t, []gocoverage.Block{
		{File: "root.go", StartLine: 9, EndLine: 11, Statements: 5},
		{File: "root.go", StartLine: 12, EndLine: 20, Statements: 5},
		{File: "root.go", StartLine: 40, EndLine: 42, Statements: 5},
		{File: "root.go", StartLine: 50, EndLine: 50, Statements: 5},
		{File: "list.go", StartLine: 11, EndLine: 20, Statements: 8, Covered: true},
	}, normalized)
}

func TestNormalizeBlocksKeepsWideRegionThatRepeatsTheStatementCount(t *testing.T) {
	blocks := []gocoverage.Block{
		{File: "root.go", StartLine: 9, EndLine: 16, Statements: 13},
		{File: "root.go", StartLine: 18, EndLine: 27, Statements: 13},
		{File: "root.go", StartLine: 30, EndLine: 48, Statements: 13},
		{File: "root.go", StartLine: 49, EndLine: 53, Statements: 13},
		{File: "set.go", StartLine: 18, EndLine: 40, Statements: 12, Covered: true},
	}

	normalized := gocoverage.NormalizeBlocks(blocks)

	require.Equal(t, []gocoverage.Block{
		{File: "root.go", StartLine: 9, EndLine: 16, Statements: 13},
		{File: "root.go", StartLine: 30, EndLine: 48, Statements: 13},
		{File: "root.go", StartLine: 49, EndLine: 53, Statements: 13},
		{File: "set.go", StartLine: 18, EndLine: 40, Statements: 12, Covered: true},
	}, normalized)
}

func TestNormalizeBlocksLeavesCoveredRunsIntact(t *testing.T) {
	blocks := []gocoverage.Block{
		{File: "cmd.go", StartLine: 8, EndLine: 12, Statements: 10, Covered: true},
		{File: "cmd.go", StartLine: 14, EndLine: 18, Statements: 10},
		{File: "cmd.go", StartLine: 19, EndLine: 20, Statements: 10, Covered: true},
	}

	require.Equal(t, blocks, gocoverage.NormalizeBlocks(blocks))
}

func TestNormalizeBlocksKeepsFittingRegionAfterADuplicatedFragment(t *testing.T) {
	blocks := []gocoverage.Block{
		{File: "staging.go", StartLine: 121, EndLine: 123, Statements: 4},
		{File: "staging.go", StartLine: 124, EndLine: 128, Statements: 4},
		{File: "staging.go", StartLine: 132, EndLine: 136, Statements: 2, Covered: true},
	}

	normalized := gocoverage.NormalizeBlocks(blocks)

	require.Equal(t, []gocoverage.Block{
		{File: "staging.go", StartLine: 121, EndLine: 123, Statements: 4},
		{File: "staging.go", StartLine: 124, EndLine: 128, Statements: 4},
		{File: "staging.go", StartLine: 132, EndLine: 136, Statements: 2, Covered: true},
	}, normalized)
}

func TestNormalizeBlocksLeavesAccurateProfilesAlone(t *testing.T) {
	blocks := []gocoverage.Block{
		{File: "root.go", StartLine: 8, EndLine: 93, Statements: 37},
		{File: "plain.go", StartLine: 10, EndLine: 12, Statements: 2, Covered: true},
		{File: "plain.go", StartLine: 14, EndLine: 16, Statements: 2},
	}

	require.Equal(t, blocks, gocoverage.NormalizeBlocks(blocks))
}

func TestParseBlockLocation(t *testing.T) {
	file, start, end, ok := gocoverage.ParseBlockLocation("pkg/cli/commands/members/root.go:9.2,13.1")
	require.True(t, ok)
	require.Equal(t, "pkg/cli/commands/members/root.go", file)
	require.Equal(t, 9, start)
	require.Equal(t, 13, end)
}
