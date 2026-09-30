package badges

import (
	"bytes"
	"fmt"
	"html"
	"math"
	"strings"
	"text/template"
	"time"

	"github.com/google/uuid"
)

// Series colors match the Velocity "who created" chart. The card chrome
// follows a dark README stat card: muted labels, one large number, quiet chart.
const (
	colorSuperplane = "#10b981"
	colorManual     = "#64748b"
	colorBorder     = "#30363d"
	colorText       = "#f0f6fc"
	colorSubtle     = "#8b949e"
	cardBackground  = "#0d1117"

	fontFamily = "-apple-system, Segoe UI, Helvetica, Arial, sans-serif"
)

// Size is the badge layout. Small is a shields-style strip. Large is a card.
// Wide is a full-width card with a daily chart.
type Size string

const (
	SizeSmall Size = "small"
	SizeLarge Size = "large"
	SizeWide  Size = "wide"
)

// DayPoint is one day of merged pull requests. Manual work is PeopleMerged.
type DayPoint struct {
	Date             time.Time
	SuperplaneMerged int
	PeopleMerged     int
}

// Input is everything the SVG needs. CostCents is nil when the cost switch
// is off. The renderer never reads a URL flag.
//
// Layout follows dark README stat cards (Repobeats-style): rounded card,
// large share, metric row, and a quiet daily chart. Series colors stay
// the Velocity chart colors.
type Input struct {
	PeriodDays               int
	SuperplaneMerged         int
	PeopleMerged             int
	SuperplaneWaste          int
	PreviousSuperplaneMerged int
	PreviousPeopleMerged     int
	HasPrevious              bool
	ShareKnown               bool
	CostCents                *int64
	Days                     []DayPoint
	Updated                  time.Time
	Size                     Size
}

type metrics struct {
	share       int
	manualShare int
	showShare   bool
	merged      int
	trend       string
	showTrend   bool
	mergeRate   int
	showMerge   bool
	cost        string
	showCost    bool
}

// placedText is one text node. Font size, weight, and color come from the
// class the template gives it, so layout only decides position and content.
type placedText struct {
	X     int
	Y     int
	Text  string
	Width int
	Show  bool
}

type dayBar struct {
	Rects []barRect
	Label placedText
}

type barRect struct {
	X      int
	Y      int
	Width  int
	Height int
	Radius int
	Fill   string
}

// Render returns an SVG document for the requested size.
func Render(in Input) (string, error) {
	if in.PeriodDays <= 0 {
		in.PeriodDays = 30
	}
	if in.Updated.IsZero() {
		in.Updated = time.Now()
	}
	if !in.ShareKnown {
		in.PeopleMerged = 0
		in.PreviousPeopleMerged = 0
		for i := range in.Days {
			in.Days[i].PeopleMerged = 0
		}
	}
	switch in.Size {
	case SizeLarge:
		return execute(largeTemplate, layoutLarge(in))
	case SizeWide:
		return execute(wideTemplate, layoutWide(in))
	default:
		return execute(smallTemplate, layoutSmall(in))
	}
}

func execute(tmpl *template.Template, data any) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func derive(in Input) metrics {
	merged := in.SuperplaneMerged + in.PeopleMerged
	out := metrics{
		showShare: in.ShareKnown,
		merged:    merged,
	}
	if in.ShareKnown && merged > 0 {
		out.share = in.SuperplaneMerged * 100 / merged
		out.manualShare = 100 - out.share
	}
	if !in.ShareKnown {
		out.merged = 0
	}

	if out.showShare && in.HasPrevious && merged > 0 {
		previousShare := 0
		previousMerged := in.PreviousSuperplaneMerged + in.PreviousPeopleMerged
		if previousMerged > 0 {
			previousShare = in.PreviousSuperplaneMerged * 100 / previousMerged
		}
		delta := out.share - previousShare
		switch {
		case delta > 0:
			out.trend = fmt.Sprintf("▲ %d pts", delta)
		case delta < 0:
			out.trend = fmt.Sprintf("▼ %d pts", -delta)
		default:
			out.trend = "No change"
		}
		out.showTrend = true
	}

	closures := in.SuperplaneMerged + in.SuperplaneWaste
	if closures > 0 {
		out.mergeRate = in.SuperplaneMerged * 100 / closures
		out.showMerge = true
	}
	if in.CostCents != nil && in.SuperplaneMerged > 0 && *in.CostCents > 0 {
		out.cost = formatCostPerMerged(*in.CostCents, in.SuperplaneMerged)
		out.showCost = true
	}
	return out
}

func formatCostPerMerged(costCents int64, superplaneMerged int) string {
	per := (costCents + int64(superplaneMerged)/2) / int64(superplaneMerged)
	return "$" + formatDollars(per) + " per merged PR"
}

func formatDollars(cents int64) string {
	sign := ""
	if cents < 0 {
		sign = "-"
		cents = -cents
	}
	dollars := cents / 100
	remainder := cents % 100
	body := fmt.Sprintf("%d", dollars)
	if dollars >= 1000 {
		body = withThousands(body)
	}
	return fmt.Sprintf("%s%s.%02d", sign, body, remainder)
}

func withThousands(digits string) string {
	n := len(digits)
	if n <= 3 {
		return digits
	}
	var b strings.Builder
	lead := n % 3
	if lead == 0 {
		lead = 3
	}
	b.WriteString(digits[:lead])
	for i := lead; i < n; i += 3 {
		b.WriteByte(',')
		b.WriteString(digits[i : i+3])
	}
	return b.String()
}

func countLine(superplane, period int, compact bool) string {
	if superplane == 0 && !compact {
		return fmt.Sprintf("0 PRs · last %d days", period)
	}
	if compact {
		return fmt.Sprintf("%d PRs merged · last %d days", superplane, period)
	}
	return fmt.Sprintf("%d PRs merged via SuperPlane · last %d days", superplane, period)
}

func dayLabel(start time.Time) string {
	if start.Month() != start.AddDate(0, 0, -1).Month() {
		return start.Format("Mon Jan 2")
	}
	return start.Format("Mon 2")
}

func escapeXML(value string) string {
	return html.EscapeString(value)
}

func text(x, y int, value string) placedText {
	return placedText{X: x, Y: y, Text: escapeXML(value), Show: true}
}

func swatch(x, y int, fill string) barRect {
	return barRect{X: x, Y: y, Width: 8, Height: 8, Radius: 2, Fill: fill}
}

func textWidth(text string, fontSize float64, bold bool) int {
	scale := 1.0
	if bold {
		scale = 1.06
	}
	width := 0.0
	for _, r := range text {
		width += glyphWidth(r) * fontSize * scale
	}
	return int(math.Ceil(width))
}

func glyphWidth(r rune) float64 {
	switch r {
	case ' ', '·':
		return 0.34
	case 'i', 'l', 'j', 'I', '1', '.', ',':
		return 0.34
	case 'm', 'w', 'M', 'W':
		return 0.92
	case '%':
		return 0.82
	case '▲', '▼':
		return 0.85
	default:
		if r > 127 {
			return 0.7
		}
		return 0.58
	}
}

func clipID() string {
	return "spb-" + strings.ReplaceAll(uuid.NewString(), "-", "")
}

type smallModel struct {
	Width      int
	LeftWidth  int
	RightWidth int
	LeftTextX  int
	RightTextX int
	LeftText   string
	RightText  string
	Title      string
	ClipID     string
}

func layoutSmall(in Input) smallModel {
	m := derive(in)
	left := "SuperPlane"
	right := fmt.Sprintf("%d PRs · %dd", in.SuperplaneMerged, in.PeriodDays)
	title := fmt.Sprintf("SuperPlane · %d PRs · %dd", in.SuperplaneMerged, in.PeriodDays)
	if m.showShare {
		right = fmt.Sprintf("%d%% PRs · %dd", m.share, in.PeriodDays)
		title = fmt.Sprintf("SuperPlane %d%% · %d PRs · %dd", m.share, in.SuperplaneMerged, in.PeriodDays)
	}
	pad := 10
	leftWidth := textWidth(left, 12, true) + pad*2
	rightWidth := textWidth(right, 12, true) + pad*2
	return smallModel{
		Width:      leftWidth + rightWidth,
		LeftWidth:  leftWidth,
		RightWidth: rightWidth,
		LeftTextX:  pad,
		RightTextX: leftWidth + pad,
		LeftText:   escapeXML(left),
		RightText:  escapeXML(right),
		Title:      escapeXML(title),
		ClipID:     clipID(),
	}
}

type largeModel struct {
	Height    int
	InnerH    int
	Title     string
	ClipID    string
	Wordmark  placedText
	Share     placedText
	Trend     placedText
	Caption   placedText
	Count     placedText
	MergeRate placedText
	Cost      placedText
	BarX      int
	BarY      int
	BarW      int
	BarH      int
	GreenW    int
	SlateX    int
	SlateW    int
	ShowTrack bool
}

func layoutLarge(in Input) largeModel {
	m := derive(in)
	const (
		width = 360
		pad   = 20
		step  = 20
	)
	out := largeModel{
		ClipID:    clipID(),
		Title:     escapeXML(largeTitle(in, m)),
		Wordmark:  text(pad, 28, "SuperPlane"),
		BarX:      pad,
		BarW:      width - pad*2,
		BarH:      6,
		ShowTrack: m.showShare,
	}

	y := 52
	if m.showShare {
		share := fmt.Sprintf("%d%%", m.share)
		out.Share = text(pad, 72, share)
		if m.showTrend {
			out.Trend = text(pad+textWidth(share, 36, true)+12, 64, m.trend)
		}
		y = 96
		out.Caption = text(pad, y, "of merged PRs via SuperPlane")
		y += step
	}
	out.Count = text(pad, y, countLine(in.SuperplaneMerged, in.PeriodDays, false))
	if m.showMerge {
		y += step
		out.MergeRate = text(pad, y, fmt.Sprintf("%d%% of SuperPlane PRs merged", m.mergeRate))
	}
	if m.showCost {
		y += step
		out.Cost = text(pad, y, m.cost)
	}

	out.BarY = y + 14
	out.Height = y + 16
	if m.showShare {
		out.Height = out.BarY + out.BarH + 20
		out.GreenW, out.SlateW = splitBar(out.BarW, m)
		out.SlateX = pad + out.GreenW
	}
	out.InnerH = out.Height - 1
	return out
}

// splitBar divides the share bar. Each visible side keeps at least two
// pixels, so a small share is still a visible sliver.
func splitBar(barW int, m metrics) (green, slate int) {
	if m.merged == 0 {
		return 0, 0
	}
	green = barW * m.share / 100
	slate = barW - green
	if m.share > 0 && green < 2 {
		green = 2
	}
	if m.manualShare > 0 && slate < 2 && barW > green+2 {
		slate = 2
		green = barW - slate
	}
	return green, slate
}

func largeTitle(in Input, m metrics) string {
	if m.showShare {
		return fmt.Sprintf("%d%% of merged PRs via SuperPlane vs previous %d days", m.share, in.PeriodDays)
	}
	return fmt.Sprintf("%d PRs merged via SuperPlane · last %d days", in.SuperplaneMerged, in.PeriodDays)
}

type wideModel struct {
	Height    int
	InnerH    int
	Title     string
	Wordmark  placedText
	Share     placedText
	Trend     placedText
	Count     placedText
	MergeRate placedText
	Cost      placedText
	Days      []dayBar
	Legend    legendModel
	Baseline  barRect
}

func layoutWide(in Input) wideModel {
	m := derive(in)
	const (
		pad         = 28
		right       = 772
		row1Y       = 44
		row2Y       = 76
		chartX      = 28
		chartW      = 744
		chartH      = 120
		baseChartY  = 96
		baseLegendY = 252
		baseHeight  = 280
	)
	out := wideModel{
		Title:    escapeXML(wideTitle(in, m)),
		Wordmark: text(pad, row1Y, "SuperPlane"),
	}
	if m.showShare {
		shareX := pad + textWidth("SuperPlane", 16, true) + 16
		out.Share = text(shareX, row1Y, fmt.Sprintf("%d%%", m.share))
	}

	var row metricRow
	if m.showTrend {
		row.add(&out.Trend, fmt.Sprintf("%s vs previous %d days", m.trend, in.PeriodDays), 13, true, false)
	}
	row.add(&out.Count, countLine(in.SuperplaneMerged, in.PeriodDays, true), 13, false, false)
	if m.showMerge {
		row.add(&out.MergeRate, fmt.Sprintf("%d%% merge rate", m.mergeRate), 13, false, false)
	}
	if m.showCost {
		row.add(&out.Cost, m.cost, 13, true, true)
	}
	extra := row.place(pad, row2Y, right)

	chartY := baseChartY + extra
	out.Days = layoutChart(in.Days, chartX, chartY, chartW, chartH)
	out.Legend = layoutLegend(m, in.Updated, pad, baseLegendY+extra, right)
	out.Baseline = barRect{X: chartX, Y: chartY + chartH, Width: chartW, Height: 1, Fill: colorBorder}
	out.Height = baseHeight + extra
	out.InnerH = out.Height - 1
	return out
}

func wideTitle(in Input, m metrics) string {
	if m.showShare {
		return fmt.Sprintf("SuperPlane %d%% of merged PRs vs previous %d days", m.share, in.PeriodDays)
	}
	return fmt.Sprintf("SuperPlane · %d PRs merged · last %d days", in.SuperplaneMerged, in.PeriodDays)
}

const (
	metricRowGap  = 18
	metricRowStep = 22
)

// metricRow places the wide header metrics from left to right. Each entry
// writes its position into the model field that the template renders, so
// the template can name every metric instead of ranging over a slice.
type metricRow struct {
	entries []metricEntry
}

type metricEntry struct {
	field *placedText
	text  string
	size  float64
	bold  bool
	wrap  bool
}

func (r *metricRow) add(field *placedText, value string, size float64, bold, wrap bool) {
	r.entries = append(r.entries, metricEntry{field: field, text: value, size: size, bold: bold, wrap: wrap})
}

// place returns the extra card height the row needs. Only the last entry
// can wrap, and only when one line runs past right.
func (r *metricRow) place(startX, y, right int) int {
	if len(r.entries) == 0 {
		return 0
	}
	r.placeLine(r.entries, startX, y)
	last := r.entries[len(r.entries)-1]
	if !r.overflows(right) || !last.wrap || len(r.entries) < 2 {
		return 0
	}
	r.placeLine(r.entries[:len(r.entries)-1], startX, y)
	r.placeLine(r.entries[len(r.entries)-1:], startX, y+metricRowStep)
	return metricRowStep
}

func (r *metricRow) placeLine(entries []metricEntry, startX, y int) {
	x := startX
	for _, entry := range entries {
		width := textWidth(entry.text, entry.size, entry.bold)
		*entry.field = placedText{X: x, Y: y, Text: escapeXML(entry.text), Width: width, Show: true}
		x += width + metricRowGap
	}
}

func (r *metricRow) overflows(right int) bool {
	last := r.entries[len(r.entries)-1].field
	return last.X+last.Width > right
}

func layoutChart(days []DayPoint, chartX, chartY, chartW, chartH int) []dayBar {
	out := make([]dayBar, len(days))
	maxCount := 0
	for _, day := range days {
		total := day.SuperplaneMerged + day.PeopleMerged
		if total > maxCount {
			maxCount = total
		}
	}
	labels := labelIndexes(len(days))
	labelSet := map[int]bool{}
	for _, idx := range labels {
		labelSet[idx] = true
	}
	if len(days) == 0 {
		return out
	}
	slot := float64(chartW) / float64(len(days))
	barW := int(math.Round(slot * 0.42))
	if barW < 3 {
		barW = 3
	}
	bottom := chartY + chartH
	for i, day := range days {
		x := chartX + int(math.Round(float64(i)*slot+(slot-float64(barW))/2))
		bar := dayBar{}
		total := day.SuperplaneMerged + day.PeopleMerged
		if total > 0 && maxCount > 0 {
			totalH := total * chartH / maxCount
			if totalH < 1 {
				totalH = 1
			}
			slateH := 0
			greenH := 0
			if day.PeopleMerged > 0 && day.SuperplaneMerged > 0 {
				slateH = day.PeopleMerged * totalH / total
				greenH = totalH - slateH
				if slateH < 1 {
					slateH = 1
				}
				if greenH < 1 {
					greenH = 1
				}
				if slateH+greenH > chartH {
					greenH = chartH - slateH
					if greenH < 1 {
						greenH = 1
						slateH = chartH - 1
					}
				}
			} else if day.SuperplaneMerged > 0 {
				greenH = totalH
			} else {
				slateH = totalH
			}
			if greenH > 0 {
				radius := 3
				if radius > barW/2 {
					radius = barW / 2
				}
				if radius > greenH {
					radius = greenH
				}
				extend := 0
				if slateH > 0 && greenH > radius {
					extend = radius
				}
				bar.Rects = append(bar.Rects, barRect{
					X: x, Y: bottom - slateH - greenH, Width: barW, Height: greenH + extend,
					Radius: radius, Fill: colorSuperplane,
				})
			}
			if slateH > 0 {
				radius := 0
				if greenH == 0 {
					radius = 3
					if radius > barW/2 {
						radius = barW / 2
					}
					if radius > slateH {
						radius = slateH
					}
				}
				bar.Rects = append(bar.Rects, barRect{
					X: x, Y: bottom - slateH, Width: barW, Height: slateH, Radius: radius, Fill: colorManual,
				})
			}
		}
		if labelSet[i] {
			bar.Label = text(x+barW/2, bottom+16, dayLabel(day.Date))
		}
		out[i] = bar
	}
	return out
}

func labelIndexes(n int) []int {
	if n <= 0 {
		return nil
	}
	if n == 1 {
		return []int{0}
	}
	mid := (n - 1) / 2
	if mid == 0 || mid == n-1 {
		return []int{0, n - 1}
	}
	return []int{0, mid, n - 1}
}

// legendModel is the footer row: one swatch and label per series on the
// left, and the render date on the right. Manual work is hidden when the
// workspace has no people merges to compare against.
type legendModel struct {
	Auto         placedText
	Manual       placedText
	Updated      placedText
	AutoSwatch   barRect
	ManualSwatch barRect
}

func layoutLegend(m metrics, updated time.Time, x, y, right int) legendModel {
	out := legendModel{
		Updated:    text(right, y, "Updated "+updated.Format("Jan 2")),
		AutoSwatch: swatch(x, y-8, colorSuperplane),
	}
	if !m.showShare {
		out.Auto = text(x+14, y, "Automated via SuperPlane")
		return out
	}

	auto := fmt.Sprintf("Automated via SuperPlane (%d%%)", m.share)
	out.Auto = text(x+14, y, auto)
	manualX := x + 14 + textWidth(auto, 11, false) + 28
	out.Manual = text(manualX, y, fmt.Sprintf("Manual work (%d%%)", m.manualShare))
	out.ManualSwatch = swatch(manualX-14, y-8, colorManual)
	return out
}

var (
	smallTemplate = template.Must(template.New("small").Parse(smallSVG))
	largeTemplate = template.Must(template.New("large").Parse(largeSVG))
	wideTemplate  = template.Must(template.New("wide").Parse(wideSVG))
)

// Each card keeps type styling in one <style> block, so the markup below
// stays readable SVG and every <text> carries only a class and a position.
// GitHub serves the badge as an image, so an internal stylesheet applies.

const smallSVG = `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" width="{{.Width}}" height="28" viewBox="0 0 {{.Width}} 28" role="img">
  <title>{{.Title}}</title>
  <style>
    text { font-family: ` + fontFamily + `; font-size: 12px; font-weight: 600; fill: #ffffff; }
  </style>
  <clipPath id="{{.ClipID}}"><rect width="{{.Width}}" height="28" rx="4"/></clipPath>
  <g clip-path="url(#{{.ClipID}})">
    <rect width="{{.LeftWidth}}" height="28" fill="` + cardBackground + `"/>
    <rect x="{{.LeftWidth}}" width="{{.RightWidth}}" height="28" fill="` + colorSuperplane + `"/>
    <text x="{{.LeftTextX}}" y="19">{{.LeftText}}</text>
    <text x="{{.RightTextX}}" y="19">{{.RightText}}</text>
  </g>
</svg>
`

const largeSVG = `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" width="360" height="{{.Height}}" viewBox="0 0 360 {{.Height}}" role="img">
  <title>{{.Title}}</title>
  <style>
    text { font-family: ` + fontFamily + `; }
    .wordmark { font-size: 13px; font-weight: 700; fill: ` + colorText + `; }
    .share    { font-size: 36px; font-weight: 700; fill: ` + colorSuperplane + `; }
    .trend    { font-size: 13px; font-weight: 600; fill: ` + colorSubtle + `; }
    .caption  { font-size: 13px; font-weight: 400; fill: ` + colorSubtle + `; }
    .count    { font-size: 13px; font-weight: 500; fill: ` + colorText + `; }
    .rate     { font-size: 13px; font-weight: 400; fill: ` + colorSubtle + `; }
    .cost     { font-size: 13px; font-weight: 600; fill: ` + colorText + `; }
  </style>
  <rect x="0.5" y="0.5" width="359" height="{{.InnerH}}" rx="12" fill="` + cardBackground + `" stroke="` + colorBorder + `"/>
  <text class="wordmark" x="{{.Wordmark.X}}" y="{{.Wordmark.Y}}">{{.Wordmark.Text}}</text>
  {{if .Share.Show}}<text class="share" x="{{.Share.X}}" y="{{.Share.Y}}">{{.Share.Text}}</text>{{end}}
  {{if .Trend.Show}}<text class="trend" x="{{.Trend.X}}" y="{{.Trend.Y}}">{{.Trend.Text}}</text>{{end}}
  {{if .Caption.Show}}<text class="caption" x="{{.Caption.X}}" y="{{.Caption.Y}}">{{.Caption.Text}}</text>{{end}}
  <text class="count" x="{{.Count.X}}" y="{{.Count.Y}}">{{.Count.Text}}</text>
  {{if .MergeRate.Show}}<text class="rate" x="{{.MergeRate.X}}" y="{{.MergeRate.Y}}">{{.MergeRate.Text}}</text>{{end}}
  {{if .Cost.Show}}<text class="cost" x="{{.Cost.X}}" y="{{.Cost.Y}}">{{.Cost.Text}}</text>{{end}}
  {{if .ShowTrack}}
  <clipPath id="{{.ClipID}}"><rect x="{{.BarX}}" y="{{.BarY}}" width="{{.BarW}}" height="{{.BarH}}" rx="3"/></clipPath>
  <g clip-path="url(#{{.ClipID}})">
    <rect x="{{.BarX}}" y="{{.BarY}}" width="{{.BarW}}" height="{{.BarH}}" fill="#21262d"/>
    {{if gt .GreenW 0}}<rect x="{{.BarX}}" y="{{.BarY}}" width="{{.GreenW}}" height="{{.BarH}}" fill="` + colorSuperplane + `"/>{{end}}
    {{if gt .SlateW 0}}<rect x="{{.SlateX}}" y="{{.BarY}}" width="{{.SlateW}}" height="{{.BarH}}" fill="` + colorManual + `"/>{{end}}
  </g>
  {{end}}
</svg>
`

const wideSVG = `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" width="100%" height="{{.Height}}" viewBox="0 0 800 {{.Height}}" role="img">
  <title>{{.Title}}</title>
  <style>
    text { font-family: ` + fontFamily + `; }
    .wordmark  { font-size: 16px; font-weight: 700; fill: ` + colorText + `; }
    .share     { font-size: 28px; font-weight: 700; fill: ` + colorSuperplane + `; }
    .trend     { font-size: 13px; font-weight: 600; fill: ` + colorSubtle + `; }
    .count     { font-size: 13px; font-weight: 500; fill: ` + colorText + `; }
    .rate      { font-size: 13px; font-weight: 400; fill: ` + colorSubtle + `; }
    .cost      { font-size: 13px; font-weight: 600; fill: ` + colorText + `; }
    .day-label { font-size: 10px; font-weight: 400; fill: ` + colorSubtle + `; text-anchor: middle; }
    .legend    { font-size: 11px; font-weight: 500; fill: ` + colorText + `; }
    .updated   { font-size: 11px; font-weight: 400; fill: ` + colorSubtle + `; text-anchor: end; }
  </style>
  <rect x="0.5" y="0.5" width="799" height="{{.InnerH}}" rx="12" fill="` + cardBackground + `" stroke="` + colorBorder + `"/>

  <text class="wordmark" x="{{.Wordmark.X}}" y="{{.Wordmark.Y}}">{{.Wordmark.Text}}</text>
  {{if .Share.Show}}<text class="share" x="{{.Share.X}}" y="{{.Share.Y}}">{{.Share.Text}}</text>{{end}}
  {{if .Trend.Show}}<text class="trend" x="{{.Trend.X}}" y="{{.Trend.Y}}">{{.Trend.Text}}</text>{{end}}
  <text class="count" x="{{.Count.X}}" y="{{.Count.Y}}">{{.Count.Text}}</text>
  {{if .MergeRate.Show}}<text class="rate" x="{{.MergeRate.X}}" y="{{.MergeRate.Y}}">{{.MergeRate.Text}}</text>{{end}}
  {{if .Cost.Show}}<text class="cost" x="{{.Cost.X}}" y="{{.Cost.Y}}">{{.Cost.Text}}</text>{{end}}

  <rect x="{{.Baseline.X}}" y="{{.Baseline.Y}}" width="{{.Baseline.Width}}" height="{{.Baseline.Height}}" fill="{{.Baseline.Fill}}"/>
  {{range .Days}}<g class="day">
    {{range .Rects}}<rect x="{{.X}}" y="{{.Y}}" width="{{.Width}}" height="{{.Height}}"{{if gt .Radius 0}} rx="{{.Radius}}"{{end}} fill="{{.Fill}}"/>{{end}}
    {{if .Label.Show}}<text class="day-label" x="{{.Label.X}}" y="{{.Label.Y}}">{{.Label.Text}}</text>{{end}}
  </g>
  {{end}}
  {{with .Legend}}<rect x="{{.AutoSwatch.X}}" y="{{.AutoSwatch.Y}}" width="{{.AutoSwatch.Width}}" height="{{.AutoSwatch.Height}}" rx="{{.AutoSwatch.Radius}}" fill="{{.AutoSwatch.Fill}}"/>
  <text class="legend" x="{{.Auto.X}}" y="{{.Auto.Y}}">{{.Auto.Text}}</text>
  {{if .Manual.Show}}<rect x="{{.ManualSwatch.X}}" y="{{.ManualSwatch.Y}}" width="{{.ManualSwatch.Width}}" height="{{.ManualSwatch.Height}}" rx="{{.ManualSwatch.Radius}}" fill="{{.ManualSwatch.Fill}}"/>
  <text class="legend" x="{{.Manual.X}}" y="{{.Manual.Y}}">{{.Manual.Text}}</text>{{end}}
  <text class="updated" x="{{.Updated.X}}" y="{{.Updated.Y}}">{{.Updated.Text}}</text>{{end}}
</svg>
`
