package badges

import (
	"fmt"
	"math"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// palette is every color a badge draws with. A theme sets all of them at
// once, and a URL can replace any one of them. Theme names follow the
// README card projects that people already use, such as
// github-readme-stats, so a familiar name keeps working.
//
// Border doubles as the empty part of the share bar, so the six named
// colors are the whole surface a user can change.
type palette struct {
	// Accent fills the SuperPlane series and the small badge value.
	// ShareColor is the large share number. It matches Accent unless that
	// color disappears on the card.
	Accent     string
	ShareColor string
	// Manual fills the people series.
	Manual     string
	Text       string
	Subtle     string
	Background string
	Border     string
	// OnAccent is the text on top of the accent fill. It is not editable:
	// the theme sets it, and a custom accent derives it.
	OnAccent string
}

// Colors replaces single theme colors. Every empty field keeps the color
// the theme sets.
type Colors struct {
	Accent     string
	Manual     string
	Text       string
	Subtle     string
	Background string
	Border     string
}

// ColorParams maps a URL parameter to the palette color it replaces. The
// settings page and any hand-written README URL use these names.
var ColorParams = []string{"accent", "manual", "text", "muted", "bg", "border"}

// ColorsFromQuery reads the color parameters. It drops anything that is
// not a six-digit hex color, so a typo keeps the theme color.
func ColorsFromQuery(query url.Values) Colors {
	return Colors{
		Accent:     NormalizeColor(query.Get("accent")),
		Manual:     NormalizeColor(query.Get("manual")),
		Text:       NormalizeColor(query.Get("text")),
		Subtle:     NormalizeColor(query.Get("muted")),
		Background: NormalizeColor(query.Get("bg")),
		Border:     NormalizeColor(query.Get("border")),
	}
}

// ThemeDefault renders when the URL asks for no theme, or for a name that
// this package does not know.
const ThemeDefault = "superplane"

var themes = map[string]palette{
	ThemeDefault: {
		Accent:     "#10b981",
		Manual:     "#64748b",
		Text:       "#f0f6fc",
		Subtle:     "#8b949e",
		Background: "#0d1117",
		Border:     "#30363d",
		OnAccent:   "#ffffff",
	},
	"light": {
		Accent:     "#059669",
		Manual:     "#8c959f",
		Text:       "#1f2328",
		Subtle:     "#59636e",
		Background: "#ffffff",
		Border:     "#d1d9e0",
		OnAccent:   "#ffffff",
	},
	"github_dark": {
		Accent:     "#58a6ff",
		Manual:     "#484f58",
		Text:       "#c9d1d9",
		Subtle:     "#8b949e",
		Background: "#0d1117",
		Border:     "#30363d",
		OnAccent:   "#0d1117",
	},
	"github_light": {
		Accent:     "#0969da",
		Manual:     "#8c959f",
		Text:       "#1f2328",
		Subtle:     "#59636e",
		Background: "#ffffff",
		Border:     "#d1d9e0",
		OnAccent:   "#ffffff",
	},
	"dracula": {
		Accent:     "#50fa7b",
		Manual:     "#6272a4",
		Text:       "#f8f8f2",
		Subtle:     "#6272a4",
		Background: "#282a36",
		Border:     "#44475a",
		OnAccent:   "#282a36",
	},
	"tokyonight": {
		Accent:     "#7aa2f7",
		Manual:     "#414868",
		Text:       "#c0caf5",
		Subtle:     "#565f89",
		Background: "#1a1b27",
		Border:     "#292e42",
		OnAccent:   "#1a1b27",
	},
	"nord": {
		Accent:     "#88c0d0",
		Manual:     "#4c566a",
		Text:       "#d8dee9",
		Subtle:     "#7b88a1",
		Background: "#2e3440",
		Border:     "#3b4252",
		OnAccent:   "#2e3440",
	},
	"gruvbox": {
		Accent:     "#b8bb26",
		Manual:     "#665c54",
		Text:       "#ebdbb2",
		Subtle:     "#a89984",
		Background: "#282828",
		Border:     "#3c3836",
		OnAccent:   "#282828",
	},
	"catppuccin_mocha": {
		Accent:     "#a6e3a1",
		Manual:     "#45475a",
		Text:       "#cdd6f4",
		Subtle:     "#6c7086",
		Background: "#1e1e2e",
		Border:     "#313244",
		OnAccent:   "#1e1e2e",
	},
}

// ThemeNames returns the theme names a URL can ask for, default first.
func ThemeNames() []string {
	rest := make([]string, 0, len(themes)-1)
	for name := range themes {
		if name != ThemeDefault {
			rest = append(rest, name)
		}
	}
	sort.Strings(rest)
	return append([]string{ThemeDefault}, rest...)
}

// NormalizeTheme maps a URL value to a known theme name. An unknown name
// falls back to the default, so a typo still renders a badge.
func NormalizeTheme(raw string) string {
	name := strings.ToLower(strings.TrimSpace(raw))
	if _, ok := themes[name]; ok {
		return name
	}
	return ThemeDefault
}

var colorPattern = regexp.MustCompile(`^#?([0-9a-fA-F]{6})$`)

// NormalizeColor returns a "#rrggbb" color, or an empty string when the
// value is not a six-digit hex color. The caller treats empty as "keep the
// color the theme sets".
func NormalizeColor(raw string) string {
	match := colorPattern.FindStringSubmatch(strings.TrimSpace(raw))
	if match == nil {
		return ""
	}
	return "#" + strings.ToLower(match[1])
}

func resolvePalette(theme string, colors Colors) palette {
	out := themes[NormalizeTheme(theme)]
	replace(&out.Manual, colors.Manual)
	replace(&out.Text, colors.Text)
	replace(&out.Subtle, colors.Subtle)
	replace(&out.Background, colors.Background)
	replace(&out.Border, colors.Border)
	if custom := NormalizeColor(colors.Accent); custom != "" {
		out.Accent = custom
		out.OnAccent = readableOn(custom)
	}

	// A custom background can land on the text color. The badge keeps its
	// labels readable, because a figure nobody can see is worse than a color
	// the editor did not ask for.
	out.Text = legible(out.Text, out.Background)
	out.Subtle = legible(out.Subtle, out.Background)
	out.ShareColor = shareColor(out.Accent, out.Background, out.Text)
	return out
}

// legible keeps a text color unless it disappears into the background.
func legible(text, background string) string {
	if contrastRatio(text, background) >= minTextContrast {
		return text
	}
	return readableOn(background)
}

func replace(target *string, color string) {
	if normalized := NormalizeColor(color); normalized != "" {
		*target = normalized
	}
}

const (
	onAccentDark  = "#101418"
	onAccentLight = "#ffffff"
	// minShareContrast is WCAG AA for large text. A share number below this
	// uses the card text color so it stays visible.
	minShareContrast = 3
	// minTextContrast is the lowest ratio a text color can have against the
	// background. It sits below the WCAG text ratios on purpose: it rejects a
	// color that is close to invisible and leaves every preset alone.
	minTextContrast = 2
)

// readableOn picks the text color with the higher contrast on a custom
// accent fill. A luminance cutoff left mid-gray fills, such as #999999,
// with white text that was hard to read.
func readableOn(fill string) string {
	if contrastRatio(fill, onAccentDark) >= contrastRatio(fill, onAccentLight) {
		return onAccentDark
	}
	return onAccentLight
}

// shareColor keeps the large share number readable on the card. A custom
// accent that is too close to the background, such as white on a light
// theme, uses the card text color instead.
func shareColor(accent, background, text string) string {
	if contrastRatio(accent, background) >= minShareContrast {
		return accent
	}
	return text
}

func contrastRatio(a, b string) float64 {
	lighter := relativeLuminance(a)
	darker := relativeLuminance(b)
	if darker > lighter {
		lighter, darker = darker, lighter
	}
	return (lighter + 0.05) / (darker + 0.05)
}

func relativeLuminance(hex string) float64 {
	channels := [3]float64{}
	for i := range channels {
		start := 1 + i*2
		value, err := strconv.ParseUint(hex[start:start+2], 16, 8)
		if err != nil {
			return 0
		}
		channels[i] = linearize(float64(value) / 255)
	}
	return 0.2126*channels[0] + 0.7152*channels[1] + 0.0722*channels[2]
}

func linearize(channel float64) float64 {
	if channel <= 0.03928 {
		return channel / 12.92
	}
	return math.Pow((channel+0.055)/1.055, 2.4)
}

// logoPath is the SuperPlane mark on its own 720 by 810 canvas. It matches
// web_src/src/assets/superplane.svg. The badge scales it per size and fills
// it with the text color, so the mark and the wordmark read as one lockup.
const logoPath = "M543.228 153.905C626.85 108.604 728.51 169.153 728.511 264.256V392.792C728.511 399.008 728.04 405.156 727.144 411.195C805.606 385.932 890.542 443.979 890.542 530.912V659.447C890.541 705.5 865.32 747.86 824.824 769.798L536.299 926.102C452.678 971.4 351.017 910.853 351.016 815.75V687.215C351.016 680.996 351.482 674.845 352.378 668.802C273.918 694.061 188.986 636.026 188.984 549.095V420.56C188.985 374.504 214.213 332.146 254.707 310.209L543.228 153.905ZM402.646 671.62C401.575 676.69 401.016 681.911 401.016 687.215V815.75C401.017 872.963 462.174 909.387 512.48 882.137L654.951 804.95C647.141 803.568 639.333 800.898 631.777 796.771L402.646 671.62ZM840.542 530.912C840.542 473.696 779.384 437.272 729.077 464.525L593.306 538.07L677.759 584.198C718.042 606.201 743.101 648.444 743.101 694.344V730.731C743.101 741.47 740.933 751.471 737.09 760.453L801.006 725.834C825.365 712.637 840.541 687.155 840.542 659.447V530.912ZM429.683 629.418L655.742 752.889C672.57 762.08 693.1 758.895 693.101 739.723V694.344C693.101 666.73 678.027 641.317 653.794 628.08L543.423 567.796L429.683 629.418ZM278.525 354.173C254.163 367.371 238.985 392.854 238.984 420.56V549.095C238.986 606.308 300.142 642.729 350.449 615.477L489.658 540.057L416.729 500.545C376.234 478.608 351.006 436.25 351.006 390.194V353.71C351.007 334.846 357.679 318.249 368.423 305.467L278.525 354.173ZM438.286 331.507C421.462 322.394 401.007 325.778 401.006 344.912V390.194C401.006 417.9 416.185 443.382 440.547 456.581L539.653 510.267L653.955 448.348L438.286 331.507ZM678.511 264.256C678.51 207.042 617.352 170.618 567.046 197.87L417.485 278.89C432.1 277.13 447.558 279.662 462.104 287.543L677.617 404.295C678.2 400.523 678.511 396.68 678.511 392.792V264.256Z"

const (
	logoCanvasWidth  = 720.0
	logoCanvasHeight = 810.0
)

// logoMark is the placed mark. Width lets the caller put the wordmark after
// it without knowing the scale.
type logoMark struct {
	Transform string
	Width     int
}

func placeLogo(x, top, height float64) logoMark {
	scale := height / logoCanvasHeight
	return logoMark{
		Transform: fmt.Sprintf("translate(%.1f %.1f) scale(%.6f) translate(-180 -135)", x, top, scale),
		Width:     int(math.Ceil(logoCanvasWidth * scale)),
	}
}
