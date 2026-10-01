package gocoverage

import "strings"

// Block is one region from a Go coverage profile.
type Block struct {
	File       string
	StartLine  int
	EndLine    int
	Statements int
	Covered    bool
}

// NormalizeBlocks corrects a Go 1.27 cover profile bug.
//
// A branchless function that contains a multi-line composite literal is split
// into several blocks, and each block is given the original block's statement
// count. Summing those blocks multiplies uncovered statements and drops package
// coverage even when the tests did not change. Consecutive uncovered blocks
// with that repeated count are one block. Covered runs stay unchanged so a
// real executed path is not reduced.
func NormalizeBlocks(blocks []Block) []Block {
	if len(blocks) == 0 {
		return nil
	}

	normalized := make([]Block, 0, len(blocks))
	for index := 0; index < len(blocks); {
		end := index + 1
		for end < len(blocks) && sameStatementRun(blocks[index], blocks[end]) {
			end++
		}
		group := blocks[index:end]
		if duplicatedStatementRun(group) && !anyCovered(group) {
			normalized = append(normalized, group[0])
			index = end
			continue
		}

		normalized = append(normalized, group...)
		index = end
	}

	return normalized
}

func sameStatementRun(left, right Block) bool {
	return left.File == right.File && left.Statements == right.Statements && right.StartLine >= left.StartLine
}

func duplicatedStatementRun(group []Block) bool {
	if len(group) < 2 || group[0].Statements == 0 {
		return false
	}

	for _, block := range group {
		if block.Statements > lineSpan(block) {
			return true
		}
	}
	return false
}

func anyCovered(group []Block) bool {
	for _, block := range group {
		if block.Covered {
			return true
		}
	}
	return false
}
func lineSpan(block Block) int {
	span := block.EndLine - block.StartLine + 1
	if span < 1 {
		return 1
	}
	return span
}

// ParseBlockLocation reads the file and line range from a coverage profile location.
// The location form is path:startLine.startCol,endLine.endCol.
func ParseBlockLocation(location string) (file string, startLine int, endLine int, ok bool) {
	file, positions, found := strings.Cut(location, ":")
	if !found || file == "" {
		return "", 0, 0, false
	}

	start, end, found := strings.Cut(positions, ",")
	if !found {
		return "", 0, 0, false
	}

	startLine, ok = leadingInt(start)
	if !ok {
		return "", 0, 0, false
	}
	endLine, ok = leadingInt(end)
	if !ok {
		return "", 0, 0, false
	}
	return file, startLine, endLine, true
}

func leadingInt(value string) (int, bool) {
	parsed := 0
	digits := 0
	for _, char := range value {
		if char < '0' || char > '9' {
			break
		}
		parsed = parsed*10 + int(char-'0')
		digits++
	}
	if digits == 0 {
		return 0, false
	}
	return parsed, true
}
