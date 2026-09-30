package badges

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// palette is every color a badge draws with. A theme sets all of them at
// once. Theme names follow the README card projects that people already
// use, such as github-readme-stats, so a familiar name keeps working.
type palette struct {
	// Accent fills the SuperPlane series, the large share, and the small
	// badge value. OnAccent is the text that sits on top of that fill.
	Accent   string
	OnAccent string
	// Manual fills the people series.
	Manual     string
	Text       string
	Subtle     string
	Background string
	Border     string
	// Track is the empty part of the share bar on the large card.
	Track string
}

// ThemeDefault renders when the URL asks for no theme, or for a name that
// this package does not know.
const ThemeDefault = "superplane"

var themes = map[string]palette{
	ThemeDefault: {
		Accent:     "#10b981",
		OnAccent:   "#ffffff",
		Manual:     "#64748b",
		Text:       "#f0f6fc",
		Subtle:     "#8b949e",
		Background: "#0d1117",
		Border:     "#30363d",
		Track:      "#21262d",
	},
	"light": {
		Accent:     "#059669",
		OnAccent:   "#ffffff",
		Manual:     "#8c959f",
		Text:       "#1f2328",
		Subtle:     "#59636e",
		Background: "#ffffff",
		Border:     "#d1d9e0",
		Track:      "#eaeef2",
	},
	"github_dark": {
		Accent:     "#58a6ff",
		OnAccent:   "#0d1117",
		Manual:     "#484f58",
		Text:       "#c9d1d9",
		Subtle:     "#8b949e",
		Background: "#0d1117",
		Border:     "#30363d",
		Track:      "#21262d",
	},
	"github_light": {
		Accent:     "#0969da",
		OnAccent:   "#ffffff",
		Manual:     "#8c959f",
		Text:       "#1f2328",
		Subtle:     "#59636e",
		Background: "#ffffff",
		Border:     "#d1d9e0",
		Track:      "#eaeef2",
	},
	"dracula": {
		Accent:     "#50fa7b",
		OnAccent:   "#282a36",
		Manual:     "#6272a4",
		Text:       "#f8f8f2",
		Subtle:     "#6272a4",
		Background: "#282a36",
		Border:     "#44475a",
		Track:      "#44475a",
	},
	"tokyonight": {
		Accent:     "#7aa2f7",
		OnAccent:   "#1a1b27",
		Manual:     "#414868",
		Text:       "#c0caf5",
		Subtle:     "#565f89",
		Background: "#1a1b27",
		Border:     "#292e42",
		Track:      "#292e42",
	},
	"nord": {
		Accent:     "#88c0d0",
		OnAccent:   "#2e3440",
		Manual:     "#4c566a",
		Text:       "#d8dee9",
		Subtle:     "#7b88a1",
		Background: "#2e3440",
		Border:     "#3b4252",
		Track:      "#3b4252",
	},
	"gruvbox": {
		Accent:     "#b8bb26",
		OnAccent:   "#282828",
		Manual:     "#665c54",
		Text:       "#ebdbb2",
		Subtle:     "#a89984",
		Background: "#282828",
		Border:     "#3c3836",
		Track:      "#3c3836",
	},
	"catppuccin_mocha": {
		Accent:     "#a6e3a1",
		OnAccent:   "#1e1e2e",
		Manual:     "#45475a",
		Text:       "#cdd6f4",
		Subtle:     "#6c7086",
		Background: "#1e1e2e",
		Border:     "#313244",
		Track:      "#313244",
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

var accentPattern = regexp.MustCompile(`^#?([0-9a-fA-F]{6})$`)

// NormalizeAccent returns a "#rrggbb" color, or an empty string when the
// value is not a six-digit hex color. The caller treats empty as "keep the
// accent the theme sets".
func NormalizeAccent(raw string) string {
	match := accentPattern.FindStringSubmatch(strings.TrimSpace(raw))
	if match == nil {
		return ""
	}
	return "#" + strings.ToLower(match[1])
}

func resolvePalette(theme, accent string) palette {
	out := themes[NormalizeTheme(theme)]
	if custom := NormalizeAccent(accent); custom != "" {
		out.Accent = custom
		out.OnAccent = readableOn(custom)
	}
	return out
}

// readableOn picks the text color for a custom accent fill. A light accent
// takes dark text, and a dark accent takes white text.
func readableOn(fill string) string {
	if relativeLuminance(fill) > 0.4 {
		return "#101418"
	}
	return "#ffffff"
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

// logoPath is the SuperPlane mark on its own 29 by 21 canvas. It matches
// web_src/src/assets/superplane.svg. The badge scales it per size and fills
// it with the text color, so the mark and the wordmark read as one lockup.
const logoPath = "M29 14.32C28.5655 14.7542 28.1107 15.1686 27.6379 15.5626L17.112 5.16914L22.7409 18.5877C22.1782 18.8335 21.6021 19.0552 21.0139 19.2508L15.4286 5.93611V20.2791C15.1216 20.2926 14.8129 20.3 14.5025 20.3L14.2391 20.2982C14.0172 20.2955 13.7961 20.2888 13.5759 20.2791V5.88893L7.97322 19.2446C7.38516 19.0484 6.80914 18.8264 6.24652 18.58L11.8656 5.18536L1.36087 15.5578C0.888333 15.1638 0.434276 14.7489 0 14.3147L14.4975 0L29 14.32Z"

const (
	logoCanvasWidth  = 29.0
	logoCanvasHeight = 21.0
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
		Transform: fmt.Sprintf("translate(%.1f %.1f) scale(%.4f)", x, top, scale),
		Width:     int(math.Ceil(logoCanvasWidth * scale)),
	}
}
