package badges

import (
	"encoding/xml"
	"fmt"
	"io"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRender_EachSizeIsValidXMLWithShareAndTotal(t *testing.T) {
	in := exampleInput()
	for _, size := range []Size{SizeSmall, SizeLarge, SizeWide} {
		in.Size = size
		svg := mustRender(t, in)
		parseSVG(t, svg)
		assert.Contains(t, svg, "46%")
		assert.Contains(t, svg, "35")
		assert.NotContains(t, svg, "<script")
	}
}

func TestRender_EscapesText(t *testing.T) {
	escaped := escapeXML(`a<b>&"`)
	assert.NotContains(t, escaped, "<")
	assert.Contains(t, escaped, "lt;")
	assert.Contains(t, escaped, "amp;")
	svg := mustRender(t, exampleInput())
	assert.NotContains(t, svg, "<script")
	assert.NotContains(t, svg, "<>")
}

func TestRender_FullWidthBarCounts(t *testing.T) {
	for _, period := range []int{7, 14, 30} {
		in := exampleInput()
		in.Size = SizeWide
		in.PeriodDays = period
		in.Days = daysFor(period, 1, 0)
		svg := mustRender(t, in)
		assert.Equal(t, period, strings.Count(svg, `class="day"`), "period %d", period)
	}
}

func TestRender_StackedDayHasTwoRects(t *testing.T) {
	in := exampleInput()
	in.Size = SizeWide
	in.PeriodDays = 7
	in.Days = daysFor(7, 0, 0)
	in.Days[3] = DayPoint{Date: in.Days[3].Date, SuperplaneMerged: 4, PeopleMerged: 2}
	svg := mustRender(t, in)
	groups := dayGroupBodies(svg)
	require.Len(t, groups, 7)
	rects := strings.Count(groups[3], "<rect")
	assert.Equal(t, 2, rects)
	assert.Contains(t, groups[3], colorSuperplane)
	assert.Contains(t, groups[3], colorManual)
}

func TestRender_ZeroMergesHasNoBarsTrendMergeRateOrCost(t *testing.T) {
	cost := int64(14700)
	in := Input{
		PeriodDays:               14,
		CostCents:                &cost,
		HasPrevious:              true,
		ShareKnown:               true,
		PreviousSuperplaneMerged: 10,
		PreviousPeopleMerged:     10,
		Updated:                  time.Now(),
		Days:                     daysFor(14, 0, 0),
	}
	for _, size := range []Size{SizeLarge, SizeWide} {
		in.Size = size
		svg := mustRender(t, in)
		parseSVG(t, svg)
		assert.Contains(t, svg, "0 PRs")
		assert.NotContains(t, svg, "▲")
		assert.NotContains(t, svg, "▼")
		assert.NotContains(t, svg, "No change")
		assert.NotContains(t, svg, "merge rate")
		assert.NotContains(t, svg, "of SuperPlane PRs merged")
		assert.NotContains(t, svg, "$")
		assert.NotContains(t, svg, "per merged PR")
		for _, group := range dayGroupBodies(svg) {
			assert.NotContains(t, group, "<rect")
		}
	}
}

func TestRender_TrendAndMergeRateAndCost(t *testing.T) {
	in := exampleInput()
	in.Size = SizeLarge
	svg := mustRender(t, in)
	assert.Contains(t, svg, "46%")
	assert.Contains(t, svg, "▲ 8 pts")
	assert.Contains(t, svg, "92%")
	assert.Contains(t, svg, "$4.20 per merged PR")
	assert.Contains(t, svg, "vs previous 14 days")

	in.PreviousSuperplaneMerged = 50
	in.PreviousPeopleMerged = 50
	svg = mustRender(t, in)
	assert.Contains(t, svg, "▼")

	in.PreviousSuperplaneMerged = 23
	in.PreviousPeopleMerged = 27
	svg = mustRender(t, in)
	assert.Contains(t, svg, "No change")

	in.HasPrevious = false
	svg = mustRender(t, in)
	assert.NotContains(t, svg, "▲")
	assert.NotContains(t, svg, "▼")
	assert.NotContains(t, svg, "No change")

	in = exampleInput()
	in.Size = SizeLarge
	in.SuperplaneMerged = 0
	in.PeopleMerged = 4
	in.SuperplaneWaste = 0
	svg = mustRender(t, in)
	assert.NotContains(t, svg, "of SuperPlane PRs merged")
	assert.NotContains(t, svg, "merge rate")
}

func TestRender_HidesCostWhenNilOrZeroAndSmallNeverShowsExtras(t *testing.T) {
	in := exampleInput()
	in.Size = SizeLarge
	in.CostCents = nil
	svg := mustRender(t, in)
	assert.NotContains(t, svg, "$")
	assert.NotContains(t, svg, "per merged PR")

	zero := int64(0)
	in.CostCents = &zero
	svg = mustRender(t, in)
	assert.NotContains(t, svg, "$")
	assert.NotContains(t, svg, "per merged PR")

	in = exampleInput()
	in.Size = SizeSmall
	svg = mustRender(t, in)
	assert.Contains(t, svg, "46% PRs · 14d")
	assert.NotContains(t, svg, "▲")
	assert.NotContains(t, svg, "▼")
	assert.NotContains(t, svg, "No change")
	assert.NotContains(t, svg, "merge rate")
	assert.NotContains(t, svg, "of SuperPlane PRs merged")
	assert.NotContains(t, svg, "$")
	assert.NotContains(t, svg, "per merged PR")
}

func TestRender_OmitsShareWhenPeopleMergesAreUnknown(t *testing.T) {
	in := exampleInput()
	in.ShareKnown = false
	for _, size := range []Size{SizeSmall, SizeLarge, SizeWide} {
		in.Size = size
		svg := mustRender(t, in)
		parseSVG(t, svg)
		assert.NotContains(t, svg, "46%")
		assert.NotContains(t, svg, ">100%<")
		assert.NotContains(t, svg, "100% of merged PRs")
		assert.NotContains(t, svg, "100% PRs")
		assert.NotContains(t, svg, "▲")
		assert.NotContains(t, svg, "of merged PRs via SuperPlane")
		assert.NotContains(t, svg, "Manual work")
	}

	in.Size = SizeSmall
	assert.Contains(t, mustRender(t, in), "35 PRs · 14d")
	in.Size = SizeLarge
	assert.Contains(t, mustRender(t, in), "35 PRs merged via SuperPlane · last 14 days")
	in.Size = SizeWide
	assert.Contains(t, mustRender(t, in), "Automated via SuperPlane")
}

func TestRender_WideKeepsLongCostInsideViewBox(t *testing.T) {
	in := exampleInput()
	in.Size = SizeWide
	cost := int64(350_000_000_000)
	in.CostCents = &cost
	label := formatCostPerMerged(cost, in.SuperplaneMerged)
	svg := mustRender(t, in)
	parseSVG(t, svg)
	assertWideKeepsCostAndMetrics(t, svg, label)
}

func TestRender_WideKeepsMetricsWhenCostIsShown(t *testing.T) {
	in := exampleInput()
	in.Size = SizeWide
	svg := mustRender(t, in)
	assertWideKeepsCostAndMetrics(t, svg, "$4.20 per merged PR")
}

func TestRender_WideKeepsFittingCostOnTheHeaderLine(t *testing.T) {
	cost := int64(100)
	in := Input{
		PeriodDays:       7,
		SuperplaneMerged: 1,
		ShareKnown:       true,
		CostCents:        &cost,
		Updated:          time.Now(),
		Days:             daysFor(7, 0, 0),
		Size:             SizeWide,
	}
	svg := mustRender(t, in)
	_, costY := textPosition(t, svg, "$1.00 per merged PR")
	_, rateY := textPosition(t, svg, "100% merge rate")
	assert.Equal(t, rateY, costY)
	assert.Equal(t, 240, viewBoxHeight(t, svg))
}

func assertWideKeepsCostAndMetrics(t *testing.T, svg, costLabel string) {
	t.Helper()
	parseSVG(t, svg)
	assert.Contains(t, svg, costLabel)
	assert.Contains(t, svg, "▲ 8 pts vs previous 14 days")
	assert.Contains(t, svg, "92% merge rate")
	assert.Contains(t, svg, "35 PRs merged · last 14 days")
	_, costY := textPosition(t, svg, costLabel)
	_, rateY := textPosition(t, svg, "92% merge rate")
	_, countY := textPosition(t, svg, "35 PRs merged · last 14 days")
	assert.Equal(t, rateY, countY)
	assert.Greater(t, costY, rateY)
	assert.LessOrEqual(t, textRightEdge(t, svg, costLabel), 780)
	assert.LessOrEqual(t, textRightEdge(t, svg, "92% merge rate"), 780)
	assert.LessOrEqual(t, textRightEdge(t, svg, "35 PRs merged · last 14 days"), 780)
	height := viewBoxHeight(t, svg)
	assert.Greater(t, height, costY)
	assert.Contains(t, svg, fmt.Sprintf(`width="799" height="%d"`, height-1))
}

func textRightEdge(t *testing.T, svg, label string) int {
	t.Helper()
	origin, _ := textPosition(t, svg, label)
	tag := textTag(t, svg, label)
	size := regexp.MustCompile(`font-size="(\d+)"`).FindStringSubmatch(tag)
	require.Len(t, size, 2)
	var fontSize int
	_, err := fmt.Sscanf(size[1], "%d", &fontSize)
	require.NoError(t, err)
	bold := strings.Contains(tag, `font-weight="700"`) || strings.Contains(tag, `font-weight="600"`)
	return origin + textWidth(label, float64(fontSize), bold)
}

func textPosition(t *testing.T, svg, label string) (int, int) {
	t.Helper()
	tag := textTag(t, svg, label)
	x := regexp.MustCompile(`x="(\d+)"`).FindStringSubmatch(tag)
	y := regexp.MustCompile(`y="(\d+)"`).FindStringSubmatch(tag)
	require.Len(t, x, 2)
	require.Len(t, y, 2)
	var originX, originY int
	_, err := fmt.Sscanf(x[1], "%d", &originX)
	require.NoError(t, err)
	_, err = fmt.Sscanf(y[1], "%d", &originY)
	require.NoError(t, err)
	return originX, originY
}

func textTag(t *testing.T, svg, label string) string {
	t.Helper()
	needle := ">" + label + "</text>"
	index := strings.Index(svg, needle)
	require.NotEqual(t, -1, index)
	start := strings.LastIndex(svg[:index], "<text ")
	require.NotEqual(t, -1, start)
	return svg[start:index]
}

func viewBoxHeight(t *testing.T, svg string) int {
	t.Helper()
	match := regexp.MustCompile(`viewBox="0 0 \d+ (\d+)"`).FindStringSubmatch(svg)
	require.Len(t, match, 2)
	var height int
	_, err := fmt.Sscanf(match[1], "%d", &height)
	require.NoError(t, err)
	return height
}

func TestRender_ThousandsSeparator(t *testing.T) {
	cost := int64(3500000)
	in := exampleInput()
	in.Size = SizeLarge
	in.CostCents = &cost
	svg := mustRender(t, in)
	assert.Contains(t, svg, "$1,000.00 per merged PR")
}

func exampleInput() Input {
	cost := int64(14700)
	return Input{
		PeriodDays:               14,
		SuperplaneMerged:         35,
		PeopleMerged:             41,
		SuperplaneWaste:          3,
		PreviousSuperplaneMerged: 19,
		PreviousPeopleMerged:     31,
		HasPrevious:              true,
		ShareKnown:               true,
		CostCents:                &cost,
		Updated:                  time.Now(),
		Days:                     daysFor(14, 2, 1),
		Size:                     SizeLarge,
	}
}

func daysFor(n, superplane, people int) []DayPoint {
	start := time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, 1-n)
	days := make([]DayPoint, n)
	for i := range days {
		days[i] = DayPoint{
			Date:             start.AddDate(0, 0, i),
			SuperplaneMerged: superplane,
			PeopleMerged:     people,
		}
	}
	return days
}

func mustRender(t *testing.T, in Input) string {
	t.Helper()
	svg, err := Render(in)
	require.NoError(t, err)
	require.NotEmpty(t, svg)
	return svg
}

func parseSVG(t *testing.T, svg string) {
	t.Helper()
	decoder := xml.NewDecoder(strings.NewReader(svg))
	for {
		_, err := decoder.Token()
		if err == io.EOF {
			return
		}
		require.NoError(t, err)
	}
}

func dayGroupBodies(svg string) []string {
	re := regexp.MustCompile(`(?s)<g class="day">(.*?)</g>`)
	matches := re.FindAllStringSubmatch(svg, -1)
	out := make([]string, 0, len(matches))
	for _, match := range matches {
		out = append(out, match[1])
	}
	return out
}
