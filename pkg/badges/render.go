package badges

import (
	"bytes"
	"fmt"
	"html"
	"math"
	"slices"
	"strings"
	"text/template"
	"time"

	"github.com/google/uuid"
)

// Colors match the Velocity "who created" chart.
const (
	colorSuperplane = "#10b981"
	colorManual     = "#64748b"
	colorSlateDark  = "#1e293b"
	colorText       = "#0f172a"
	colorSubtle     = "#475569"
	colorBorder     = "#e2e8f0"
	colorTrack      = "#e2e8f0"
	colorWhite      = "#ffffff"
	colorLabel      = "#ffffff"

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
// Mobbin and ReUI were not available in this run. Layout follows the badge
// spec, shields.io proportions, and the Velocity chart colors.
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

type placedText struct {
	X      int
	Y      int
	Size   int
	Weight string
	Fill   string
	Anchor string
	Text   string
	Width  int
}

type dayBar struct {
	Rects []barRect
	Label placedText
	Show  bool
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
	Font       string
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
	pad := 8
	leftWidth := textWidth(left, 11, true) + pad*2
	rightWidth := textWidth(right, 11, true) + pad*2
	return smallModel{
		Font:       fontFamily,
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
	Font      string
	Height    int
	InnerH    int
	Title     string
	ClipID    string
	Texts     []placedText
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
		width = 320
		pad   = 16
	)
	texts := []placedText{
		{X: pad, Y: 26, Size: 13, Weight: "700", Fill: colorText, Text: escapeXML("SuperPlane")},
	}
	y := 48
	if m.showShare {
		shareText := fmt.Sprintf("%d%%", m.share)
		texts = append(texts, placedText{X: pad, Y: 62, Size: 32, Weight: "700", Fill: colorSuperplane, Text: escapeXML(shareText)})
		if m.showTrend {
			texts = append(texts, placedText{
				X:      pad + textWidth(shareText, 32, true) + 10,
				Y:      54,
				Size:   12,
				Weight: "600",
				Fill:   colorManual,
				Text:   escapeXML(m.trend),
			})
		}
		y = 84
		texts = append(texts, placedText{X: pad, Y: y, Size: 12, Weight: "400", Fill: colorSubtle, Text: escapeXML("of merged PRs via SuperPlane")})
		y += 18
	}
	texts = append(texts, placedText{X: pad, Y: y, Size: 12, Weight: "500", Fill: colorText, Text: escapeXML(countLine(in.SuperplaneMerged, in.PeriodDays, false))})
	if m.showMerge {
		y += 18
		texts = append(texts, placedText{
			X: pad, Y: y, Size: 12, Weight: "400", Fill: colorManual,
			Text: escapeXML(fmt.Sprintf("%d%% of SuperPlane PRs merged", m.mergeRate)),
		})
	}
	if m.showCost {
		y += 18
		texts = append(texts, placedText{X: pad, Y: y, Size: 12, Weight: "600", Fill: colorText, Text: escapeXML(m.cost)})
	}
	barY := y + 12
	barH := 8
	height := y + 16
	barW := width - pad*2
	greenW := 0
	slateW := 0
	if m.showShare {
		height = barY + barH + 16
		if m.merged > 0 {
			greenW = barW * m.share / 100
			slateW = barW - greenW
			if m.share > 0 && greenW < 2 {
				greenW = 2
			}
			if m.manualShare > 0 && slateW < 2 && barW > greenW+2 {
				slateW = 2
				greenW = barW - slateW
			}
		}
	}
	title := fmt.Sprintf("%d PRs merged via SuperPlane · last %d days", in.SuperplaneMerged, in.PeriodDays)
	if m.showShare {
		title = fmt.Sprintf("%d%% of merged PRs via SuperPlane vs previous %d days", m.share, in.PeriodDays)
	}
	return largeModel{
		Font:      fontFamily,
		Height:    height,
		InnerH:    height - 1,
		Title:     escapeXML(title),
		ClipID:    clipID(),
		Texts:     texts,
		BarX:      pad,
		BarY:      barY,
		BarW:      barW,
		BarH:      barH,
		GreenW:    greenW,
		SlateX:    pad + greenW,
		SlateW:    slateW,
		ShowTrack: m.showShare,
	}
}

type wideModel struct {
	Font    string
	Height  int
	Title   string
	Texts   []placedText
	Days    []dayBar
	Legend  []placedText
	Squares []barRect
}

func layoutWide(in Input) wideModel {
	m := derive(in)
	const (
		width  = 800
		height = 240
		maxX   = 780
	)
	items := []headerItem{
		{text: "SuperPlane", size: 14, weight: "700", fill: colorText},
	}
	if m.showShare {
		items = append(items, headerItem{
			text: fmt.Sprintf("%d%%", m.share), size: 18, weight: "700", fill: colorSuperplane, dropRank: dropShare,
		})
		if m.showTrend {
			items = append(items, headerItem{
				text:     m.trend + fmt.Sprintf(" vs previous %d days", in.PeriodDays),
				size:     12,
				weight:   "600",
				fill:     colorManual,
				dropRank: dropTrend,
			})
		}
		items = append(items, headerItem{
			text:     "of merged PRs via SuperPlane",
			size:     12,
			weight:   "400",
			fill:     colorSubtle,
			dropRank: dropPhrase,
		})
	}
	items = append(items, headerItem{
		text:     countLine(in.SuperplaneMerged, in.PeriodDays, true),
		size:     12,
		weight:   "500",
		fill:     colorText,
		dropRank: dropCount,
	})
	if m.showMerge {
		items = append(items, headerItem{
			text:     fmt.Sprintf("%d%% merge rate", m.mergeRate),
			size:     12,
			weight:   "400",
			fill:     colorManual,
			dropRank: dropRate,
		})
	}
	if m.showCost {
		items = append(items, headerItem{text: m.cost, size: 12, weight: "600", fill: colorText})
	}
	texts := placeHeader(items, 20, 30, maxX)

	chart := layoutChart(in.Days, 24, 56, 752, 118)
	legend, squares := layoutLegend(m, in.Updated, 24, 214, 776)
	title := fmt.Sprintf("SuperPlane · %d PRs merged · last %d days", in.SuperplaneMerged, in.PeriodDays)
	if m.showShare {
		title = fmt.Sprintf("SuperPlane %d%% of merged PRs vs previous %d days", m.share, in.PeriodDays)
	}
	return wideModel{
		Font:    fontFamily,
		Height:  height,
		Title:   escapeXML(title),
		Texts:   texts,
		Days:    chart,
		Legend:  legend,
		Squares: squares,
	}
}

const (
	dropShare  = 1
	dropCount  = 2
	dropRate   = 3
	dropTrend  = 4
	dropPhrase = 5
)

type headerItem struct {
	text     string
	size     float64
	weight   string
	fill     string
	dropRank int
}

func placeHeader(items []headerItem, startX, y, maxX int) []placedText {
	kept := append([]headerItem(nil), items...)
	for {
		placed := placeHeaderPass(kept, startX, y)
		if !headerOverflows(placed, maxX) {
			return placed
		}
		index := highestDropRank(kept)
		if index < 0 {
			return placed
		}
		kept = slices.Delete(kept, index, index+1)
	}
}

func highestDropRank(items []headerItem) int {
	best := -1
	bestRank := 0
	for i, item := range items {
		if item.dropRank > bestRank {
			best = i
			bestRank = item.dropRank
		}
	}
	return best
}

func placeHeaderPass(items []headerItem, startX, y int) []placedText {
	x := startX
	out := make([]placedText, 0, len(items))
	for _, item := range items {
		width := textWidth(item.text, item.size, item.weight == "700" || item.weight == "600")
		out = append(out, placedText{
			X:      x,
			Y:      y,
			Size:   int(item.size),
			Weight: item.weight,
			Fill:   item.fill,
			Text:   escapeXML(item.text),
			Width:  width,
		})
		x += width + 14
	}
	return out
}

func headerOverflows(items []placedText, maxX int) bool {
	if len(items) == 0 {
		return false
	}
	last := items[len(items)-1]
	return last.X+last.Width > maxX
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
	barW := int(math.Round(slot * 0.62))
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
				bar.Rects = append(bar.Rects, barRect{
					X: x, Y: bottom - slateH, Width: barW, Height: slateH, Fill: colorManual,
				})
			}
		}
		if labelSet[i] {
			label := dayLabel(day.Date)
			bar.Label = placedText{
				X:      x + barW/2,
				Y:      bottom + 16,
				Size:   10,
				Weight: "400",
				Fill:   colorManual,
				Anchor: "middle",
				Text:   escapeXML(label),
			}
			bar.Show = true
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

func layoutLegend(m metrics, updated time.Time, x, y, right int) ([]placedText, []barRect) {
	updatedLabel := "Updated " + updated.Format("Jan 2")
	if !m.showShare {
		texts := []placedText{
			{X: x + 14, Y: y, Size: 11, Weight: "500", Fill: colorText, Text: escapeXML("Automated via SuperPlane")},
			{X: right, Y: y, Size: 11, Weight: "400", Fill: colorManual, Anchor: "end", Text: escapeXML(updatedLabel)},
		}
		squares := []barRect{
			{X: x, Y: y - 8, Width: 8, Height: 8, Radius: 2, Fill: colorSuperplane},
		}
		return texts, squares
	}
	auto := fmt.Sprintf("Automated via SuperPlane (%d%%)", m.share)
	manual := fmt.Sprintf("Manual work (%d%%)", m.manualShare)
	texts := []placedText{
		{X: x + 14, Y: y, Size: 11, Weight: "500", Fill: colorText, Text: escapeXML(auto)},
		{X: x + 14 + textWidth(auto, 11, false) + 28, Y: y, Size: 11, Weight: "500", Fill: colorText, Text: escapeXML(manual)},
		{X: right, Y: y, Size: 11, Weight: "400", Fill: colorManual, Anchor: "end", Text: escapeXML(updatedLabel)},
	}
	squares := []barRect{
		{X: x, Y: y - 8, Width: 8, Height: 8, Radius: 2, Fill: colorSuperplane},
		{X: texts[1].X - 14, Y: y - 8, Width: 8, Height: 8, Radius: 2, Fill: colorManual},
	}
	return texts, squares
}

var (
	smallTemplate = template.Must(template.New("small").Parse(smallSVG))
	largeTemplate = template.Must(template.New("large").Parse(largeSVG))
	wideTemplate  = template.Must(template.New("wide").Parse(wideSVG))
)

const smallSVG = `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" width="{{.Width}}" height="20" viewBox="0 0 {{.Width}} 20" role="img">
  <title>{{.Title}}</title>
  <clipPath id="{{.ClipID}}"><rect width="{{.Width}}" height="20" rx="3"/></clipPath>
  <g clip-path="url(#{{.ClipID}})">
    <rect width="{{.LeftWidth}}" height="20" fill="#1e293b"/>
    <rect x="{{.LeftWidth}}" width="{{.RightWidth}}" height="20" fill="#10b981"/>
    <text x="{{.LeftTextX}}" y="14" fill="#ffffff" font-size="11" font-family="{{.Font}}" font-weight="600">{{.LeftText}}</text>
    <text x="{{.RightTextX}}" y="14" fill="#ffffff" font-size="11" font-family="{{.Font}}" font-weight="600">{{.RightText}}</text>
  </g>
</svg>
`

const largeSVG = `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" width="320" height="{{.Height}}" viewBox="0 0 320 {{.Height}}" role="img">
  <title>{{.Title}}</title>
  <rect x="0.5" y="0.5" width="319" height="{{.InnerH}}" rx="10" fill="#ffffff" stroke="#e2e8f0"/>
  {{range .Texts}}<text x="{{.X}}" y="{{.Y}}" fill="{{.Fill}}" font-size="{{.Size}}" font-family="{{$.Font}}" font-weight="{{.Weight}}">{{.Text}}</text>
  {{end}}{{if .ShowTrack}}<clipPath id="{{.ClipID}}"><rect x="{{.BarX}}" y="{{.BarY}}" width="{{.BarW}}" height="{{.BarH}}" rx="4"/></clipPath>
  <g clip-path="url(#{{.ClipID}})">
    <rect x="{{.BarX}}" y="{{.BarY}}" width="{{.BarW}}" height="{{.BarH}}" fill="#e2e8f0"/>
    {{if gt .GreenW 0}}<rect x="{{.BarX}}" y="{{.BarY}}" width="{{.GreenW}}" height="{{.BarH}}" fill="#10b981"/>{{end}}
    {{if gt .SlateW 0}}<rect x="{{.SlateX}}" y="{{.BarY}}" width="{{.SlateW}}" height="{{.BarH}}" fill="#64748b"/>{{end}}
  </g>{{end}}
</svg>
`

const wideSVG = `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" width="100%" height="{{.Height}}" viewBox="0 0 800 {{.Height}}" role="img">
  <title>{{.Title}}</title>
  <rect x="0.5" y="0.5" width="799" height="239" rx="12" fill="#ffffff" stroke="#e2e8f0"/>
  {{range .Texts}}<text x="{{.X}}" y="{{.Y}}" fill="{{.Fill}}" font-size="{{.Size}}" font-family="{{$.Font}}" font-weight="{{.Weight}}"{{if .Anchor}} text-anchor="{{.Anchor}}"{{end}}>{{.Text}}</text>
  {{end}}{{range .Days}}<g class="day">
    {{range .Rects}}<rect x="{{.X}}" y="{{.Y}}" width="{{.Width}}" height="{{.Height}}"{{if gt .Radius 0}} rx="{{.Radius}}"{{end}} fill="{{.Fill}}"/>{{end}}
    {{if .Show}}<text x="{{.Label.X}}" y="{{.Label.Y}}" fill="{{.Label.Fill}}" font-size="{{.Label.Size}}" font-family="{{$.Font}}" text-anchor="middle">{{.Label.Text}}</text>{{end}}
  </g>
  {{end}}{{range .Squares}}<rect x="{{.X}}" y="{{.Y}}" width="{{.Width}}" height="{{.Height}}" rx="{{.Radius}}" fill="{{.Fill}}"/>{{end}}
  {{range .Legend}}<text x="{{.X}}" y="{{.Y}}" fill="{{.Fill}}" font-size="{{.Size}}" font-family="{{$.Font}}" font-weight="{{.Weight}}"{{if .Anchor}} text-anchor="{{.Anchor}}"{{end}}>{{.Text}}</text>
  {{end}}</svg>
`
