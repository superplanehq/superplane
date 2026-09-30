package badges

import (
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeTheme(t *testing.T) {
	assert.Equal(t, ThemeDefault, NormalizeTheme(""))
	assert.Equal(t, ThemeDefault, NormalizeTheme("not-a-theme"))
	assert.Equal(t, "tokyonight", NormalizeTheme(" TokyoNight "))
	for _, name := range ThemeNames() {
		assert.Equal(t, name, NormalizeTheme(name))
	}
}

func TestNormalizeColor(t *testing.T) {
	assert.Equal(t, "#ff8800", NormalizeColor("#FF8800"))
	assert.Equal(t, "#ff8800", NormalizeColor(" ff8800 "))
	assert.Empty(t, NormalizeColor(""))
	assert.Empty(t, NormalizeColor("#fff"))
	assert.Empty(t, NormalizeColor("red"))
	assert.Empty(t, NormalizeColor("#ff88zz"))
}

func TestColorsFromQuery(t *testing.T) {
	query := url.Values{}
	query.Set("accent", "FF8800")
	query.Set("bg", "#101010")
	query.Set("muted", "not-a-color")

	colors := ColorsFromQuery(query)
	assert.Equal(t, "#ff8800", colors.Accent)
	assert.Equal(t, "#101010", colors.Background)
	assert.Empty(t, colors.Subtle)
	assert.Empty(t, colors.Border)
}

func TestResolvePalette_ReplacesOneColorAndKeepsTheRest(t *testing.T) {
	base := themes["nord"]
	out := resolvePalette("nord", Colors{Background: "#000000"})

	assert.Equal(t, "#000000", out.Background)
	assert.Equal(t, base.Accent, out.Accent)
	assert.Equal(t, base.Text, out.Text)
	assert.Equal(t, base.Border, out.Border)
}

func TestRender_ThemeSetsCardColors(t *testing.T) {
	for _, size := range []Size{SizeSmall, SizeLarge, SizeWide} {
		in := exampleInput()
		in.Size = size
		in.Theme = "github_light"
		svg := mustRender(t, in)
		parseSVG(t, svg)

		light := themes["github_light"]
		assert.Contains(t, svg, light.Background, "size %s", size)
		assert.Contains(t, svg, light.Accent, "size %s", size)
		assert.NotContains(t, svg, themes[ThemeDefault].Accent, "size %s", size)
	}
}

func TestRender_AccentOverridesTheThemeAccent(t *testing.T) {
	in := exampleInput()
	in.Size = SizeLarge
	in.Colors = Colors{Accent: "#F59E0B"}
	svg := mustRender(t, in)
	parseSVG(t, svg)

	assert.Contains(t, svg, "#f59e0b")
	assert.NotContains(t, svg, themes[ThemeDefault].Accent)
	// A light accent takes dark text, so the value stays readable.
	assert.Equal(t, "#101418", readableOn("#f59e0b"))
	assert.Equal(t, "#ffffff", readableOn(themes[ThemeDefault].Accent))
}

func TestRender_CustomColorsBuildAThemeOnTopOfAPreset(t *testing.T) {
	in := exampleInput()
	in.Size = SizeWide
	in.Theme = "dracula"
	in.Colors = Colors{Background: "#000000", Text: "#fafafa"}
	svg := mustRender(t, in)
	parseSVG(t, svg)

	assert.Contains(t, svg, `fill="#000000"`)
	assert.Contains(t, svg, "fill: #fafafa")
	// The colors the user left alone still come from the preset.
	assert.Contains(t, svg, themes["dracula"].Accent)
	assert.NotContains(t, svg, themes["dracula"].Background)
}

func TestRender_UnknownThemeFallsBackToTheDefault(t *testing.T) {
	in := exampleInput()
	in.Size = SizeWide
	in.Theme = "not-a-theme"
	svg := mustRender(t, in)
	parseSVG(t, svg)

	assert.Contains(t, svg, themes[ThemeDefault].Background)
}

func TestRender_CarriesTheLogoMarkOnEverySize(t *testing.T) {
	for _, size := range []Size{SizeSmall, SizeLarge, SizeWide} {
		in := exampleInput()
		in.Size = size
		svg := mustRender(t, in)
		parseSVG(t, svg)

		assert.Equal(t, 1, strings.Count(svg, `class="logo"`), "size %s", size)
		assert.Contains(t, svg, logoPath, "size %s", size)
	}
}

// The wordmark must clear the mark, otherwise the lockup overlaps.
func TestRender_WordmarkStartsAfterTheLogo(t *testing.T) {
	for _, size := range []Size{SizeLarge, SizeWide} {
		in := exampleInput()
		in.Size = size
		svg := mustRender(t, in)

		wordmarkX, _ := textPosition(t, svg, "SuperPlane")
		assert.Greater(t, wordmarkX, 28+10, "size %s", size)
	}
}

// Two badges on one page must not share a stylesheet or a clip path.
func TestRender_ScopesItsStylesheetToOneBadge(t *testing.T) {
	in := exampleInput()
	in.Size = SizeLarge
	first := mustRender(t, in)
	second := mustRender(t, in)

	firstID := rootID(t, first)
	assert.NotEqual(t, firstID, rootID(t, second))
	assert.Contains(t, first, "#"+firstID+" .share")
	assert.NotContains(t, second, firstID)
}

// The settings page keeps its own copy of the palettes, because it fills
// the color fields before it asks for a badge. This test fails when the two
// tables drift apart.
func TestThemes_MatchTheSettingsPageTable(t *testing.T) {
	const path = "../../web_src/src/pages/factories/lib/badgeThemes.ts"
	source, err := os.ReadFile(path)
	require.NoError(t, err)

	entry := regexp.MustCompile(`value: "(\w+)",\n\s+label: "[^"]+",\n\s+colors: \{([^}]+)\}`)
	matches := entry.FindAllStringSubmatch(string(source), -1)
	require.Len(t, matches, len(themes), "%s lists a different number of themes", path)

	for _, match := range matches {
		name, fields := match[1], match[2]
		want, ok := themes[name]
		require.True(t, ok, "%s has theme %q that pkg/badges does not", path, name)

		assert.Equal(t, map[string]string{
			"accent": want.Accent,
			"manual": want.Manual,
			"text":   want.Text,
			"muted":  want.Subtle,
			"bg":     want.Background,
			"border": want.Border,
		}, parseColorFields(t, fields), "theme %q", name)
	}
}

func parseColorFields(t *testing.T, fields string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, pair := range regexp.MustCompile(`(\w+): "(#[0-9a-f]{6})"`).FindAllStringSubmatch(fields, -1) {
		out[pair[1]] = pair[2]
	}
	return out
}

func rootID(t *testing.T, svg string) string {
	t.Helper()
	const marker = `id="`
	start := strings.Index(svg, marker)
	require.NotEqual(t, -1, start)
	rest := svg[start+len(marker):]
	end := strings.Index(rest, `"`)
	require.NotEqual(t, -1, end)
	return rest[:end]
}
