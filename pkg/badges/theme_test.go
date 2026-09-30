package badges

import (
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

func TestNormalizeAccent(t *testing.T) {
	assert.Equal(t, "#ff8800", NormalizeAccent("#FF8800"))
	assert.Equal(t, "#ff8800", NormalizeAccent(" ff8800 "))
	assert.Empty(t, NormalizeAccent(""))
	assert.Empty(t, NormalizeAccent("#fff"))
	assert.Empty(t, NormalizeAccent("red"))
	assert.Empty(t, NormalizeAccent("#ff88zz"))
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
	in.Accent = "#F59E0B"
	svg := mustRender(t, in)
	parseSVG(t, svg)

	assert.Contains(t, svg, "#f59e0b")
	assert.NotContains(t, svg, themes[ThemeDefault].Accent)
	// A light accent takes dark text, so the value stays readable.
	assert.Equal(t, "#101418", readableOn("#f59e0b"))
	assert.Equal(t, "#ffffff", readableOn(themes[ThemeDefault].Accent))
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
