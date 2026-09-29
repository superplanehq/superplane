package badges

import (
	"encoding/xml"
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
		PreviousSuperplaneMerged: 10,
		PreviousPeopleMerged:     10,
		Updated:                  time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC),
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
		CostCents:                &cost,
		Updated:                  time.Date(2026, 9, 29, 15, 0, 0, 0, time.UTC),
		Days:                     daysFor(14, 2, 1),
		Size:                     SizeLarge,
	}
}

func daysFor(n, superplane, people int) []DayPoint {
	start := time.Date(2026, 9, 16, 0, 0, 0, 0, time.UTC)
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
