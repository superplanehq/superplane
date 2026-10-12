package browser

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"testing"
	"time"

	pw "github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/require"
)

const phoneOverrideStoryID = "factories-pages-task-split-run-phone-override--discouraged"

type phoneOverrideLayout struct {
	ViewportWidth   float64 `json:"viewportWidth"`
	DrawerWidth     float64 `json:"drawerWidth"`
	HeadlineLeft    float64 `json:"headlineLeft"`
	HeadlineRight   float64 `json:"headlineRight"`
	HeadlineTop     float64 `json:"headlineTop"`
	HeadlineBottom  float64 `json:"headlineBottom"`
	HeadlineClipped bool    `json:"headlineClipped"`
	BodyLeft        float64 `json:"bodyLeft"`
	BodyRight       float64 `json:"bodyRight"`
	BodyClipped     bool    `json:"bodyClipped"`
	ModelTop        float64 `json:"modelTop"`
	ModelBottom     float64 `json:"modelBottom"`
	ModelLeft       float64 `json:"modelLeft"`
	ModelRight      float64 `json:"modelRight"`
	ModelWidth      float64 `json:"modelWidth"`
	StartTop        float64 `json:"startTop"`
	StartBottom     float64 `json:"startBottom"`
	StartLeft       float64 `json:"startLeft"`
	StartRight      float64 `json:"startRight"`
	StartWidth      float64 `json:"startWidth"`
	Animating       bool    `json:"animating"`
}

func TestPhoneOverrideDrawerFitsTheScreen(t *testing.T) {
	baseURL := os.Getenv("STORYBOOK_URL")
	if baseURL == "" {
		t.Fatal("STORYBOOK_URL is required")
	}

	runner, err := pw.Run()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, runner.Stop())
	})

	browser, err := runner.Chromium.Launch()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, browser.Close())
	})

	// useIsMobile follows the window, not the component width. 390px is below
	// the 768px phone breakpoint and matches a phone screen.
	page, err := browser.NewPage(pw.BrowserNewPageOptions{
		Viewport: &pw.Size{Width: 390, Height: 800},
	})
	require.NoError(t, err)

	_, err = page.Goto(baseURL + "/iframe.html?id=" + phoneOverrideStoryID + "&viewMode=story")
	require.NoError(t, err)

	override := page.GetByRole("button", pw.PageGetByRoleOptions{Name: "Override"})
	require.NoError(t, override.WaitFor(pw.LocatorWaitForOptions{Timeout: pw.Float(90000)}))
	startCount, err := page.GetByRole("button", pw.PageGetByRoleOptions{Name: "Start", Exact: pw.Bool(true)}).Count()
	require.NoError(t, err)
	require.Equal(t, 0, startCount)

	require.NoError(t, override.Click())
	require.NoError(t, page.GetByRole("button", pw.PageGetByRoleOptions{Name: "Start", Exact: pw.Bool(true)}).WaitFor(
		pw.LocatorWaitForOptions{Timeout: pw.Float(10000)},
	))

	layout := waitForPhoneOverrideLayout(t, page, phoneOverrideFits)
	require.Equal(t, 390.0, layout.ViewportWidth)
	require.Greater(t, layout.DrawerWidth, 300.0)
	require.False(t, layout.HeadlineClipped)
	require.False(t, layout.BodyClipped)
	require.InDelta(t, layout.ModelTop, layout.StartTop, 2.0)
	require.LessOrEqual(t, layout.HeadlineBottom, layout.ModelTop+1)
	require.LessOrEqual(t, layout.ModelRight, layout.StartLeft+1)
}

func phoneOverrideFits(layout phoneOverrideLayout) error {
	if layout.Animating {
		return fmt.Errorf("drawer animation is still running")
	}
	if layout.ViewportWidth != 390 {
		return fmt.Errorf("viewport is %.1f px, want 390", layout.ViewportWidth)
	}
	if layout.DrawerWidth <= 300 {
		return fmt.Errorf("drawer is %.1f px wide; phone width is not under test", layout.DrawerWidth)
	}
	if layout.HeadlineClipped {
		return fmt.Errorf(
			"readiness headline is clipped: %.1f-%.1f in a %.1f px viewport",
			layout.HeadlineLeft,
			layout.HeadlineRight,
			layout.ViewportWidth,
		)
	}
	if layout.BodyClipped {
		return fmt.Errorf(
			"readiness body is clipped: %.1f-%.1f in a %.1f px viewport",
			layout.BodyLeft,
			layout.BodyRight,
			layout.ViewportWidth,
		)
	}
	if math.Abs(layout.ModelTop-layout.StartTop) > 2 {
		return fmt.Errorf("model and Start are not on one row: model top %.1f, Start top %.1f", layout.ModelTop, layout.StartTop)
	}
	if layout.HeadlineBottom > layout.ModelTop+1 {
		return fmt.Errorf("readiness message shares a row with the model control")
	}
	if layout.ModelWidth < 40 || layout.StartWidth < 40 {
		return fmt.Errorf("controls are clipped: model %.1f, Start %.1f", layout.ModelWidth, layout.StartWidth)
	}
	if layout.ModelLeft < -1 || layout.StartRight > layout.ViewportWidth+1 {
		return fmt.Errorf(
			"controls leave the screen: model %.1f-%.1f, Start %.1f-%.1f, viewport %.1f",
			layout.ModelLeft,
			layout.ModelRight,
			layout.StartLeft,
			layout.StartRight,
			layout.ViewportWidth,
		)
	}
	if layout.ModelRight > layout.StartLeft+1 {
		return fmt.Errorf("model control overlaps Start")
	}
	return nil
}

func waitForPhoneOverrideLayout(t *testing.T, page pw.Page, check func(phoneOverrideLayout) error) phoneOverrideLayout {
	t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	var last error
	var layout phoneOverrideLayout
	for time.Now().Before(deadline) {
		layout, last = readPhoneOverrideLayout(page)
		if last == nil {
			last = check(layout)
		}
		if last == nil {
			return layout
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatal(last)
	return layout
}

func readPhoneOverrideLayout(page pw.Page) (phoneOverrideLayout, error) {
	value, err := page.Evaluate(`() => {
		const drawer = document.querySelector('[data-testid="phone-override-drawer"]');
		const headline = drawer && drawer.querySelector("h3");
		const body = drawer && drawer.querySelector("p");
		const model = document.querySelector('[data-testid="split-run-draft-model"]');
		const start = [...document.querySelectorAll("button")].find((button) => button.textContent.trim() === "Start");
		if (!drawer || !headline || !body || !model || !start) {
			return null;
		}
		if (headline.textContent.trim() !== "Review the plan before you start") {
			return null;
		}
		const textBox = (element) => {
			const range = document.createRange();
			range.selectNodeContents(element);
			return range.getBoundingClientRect();
		};
		const clipped = (element) => {
			const text = textBox(element);
			if (text.width < 1 || text.height < 1) {
				return true;
			}
			if (text.left < -1 || text.right > window.innerWidth + 1) {
				return true;
			}
			let node = element;
			while (node) {
				const style = getComputedStyle(node);
				const box = node.getBoundingClientRect();
				const clipsX = style.overflowX === "hidden" || style.overflowX === "clip" || style.overflowX === "scroll" || style.overflowX === "auto";
				const clipsY = style.overflowY === "hidden" || style.overflowY === "clip" || style.overflowY === "scroll" || style.overflowY === "auto";
				if (clipsX && (text.left < box.left - 1 || text.right > box.right + 1)) {
					return true;
				}
				if (clipsY && (text.top < box.top - 1 || text.bottom > box.bottom + 1)) {
					return true;
				}
				node = node.parentElement;
			}
			return false;
		};
		const headlineBox = textBox(headline);
		const bodyBox = textBox(body);
		const modelBox = model.getBoundingClientRect();
		const startBox = start.getBoundingClientRect();
		const drawerBox = drawer.getBoundingClientRect();
		return {
			viewportWidth: window.innerWidth,
			drawerWidth: drawerBox.width,
			headlineLeft: headlineBox.left,
			headlineRight: headlineBox.right,
			headlineTop: headlineBox.top,
			headlineBottom: headlineBox.bottom,
			headlineClipped: clipped(headline),
			bodyLeft: bodyBox.left,
			bodyRight: bodyBox.right,
			bodyClipped: clipped(body),
			modelTop: modelBox.top,
			modelBottom: modelBox.bottom,
			modelLeft: modelBox.left,
			modelRight: modelBox.right,
			modelWidth: modelBox.width,
			startTop: startBox.top,
			startBottom: startBox.bottom,
			startLeft: startBox.left,
			startRight: startBox.right,
			startWidth: startBox.width,
			animating: document.getAnimations().some((animation) => animation.playState === "running"),
		};
	}`)
	if err != nil {
		return phoneOverrideLayout{}, err
	}
	if value == nil {
		return phoneOverrideLayout{}, fmt.Errorf("phone override drawer is not open")
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return phoneOverrideLayout{}, err
	}
	var layout phoneOverrideLayout
	if err := json.Unmarshal(encoded, &layout); err != nil {
		return phoneOverrideLayout{}, err
	}
	return layout, nil
}
