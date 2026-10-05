package browser

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	pw "github.com/mxschmitt/playwright-go"
	"github.com/stretchr/testify/require"
)

const fourSourceStoryID = "factories-components-backlogcreatepopover--four-sources"
const hiddenTabID = "lines-backlog-create-tab-intake-sentry"

type tabLayout struct {
	MenuLeft        float64 `json:"menuLeft"`
	MenuRight       float64 `json:"menuRight"`
	TabsLeft        float64 `json:"tabsLeft"`
	TabsRight       float64 `json:"tabsRight"`
	TabsScrollWidth float64 `json:"tabsScrollWidth"`
	TabsClientWidth float64 `json:"tabsClientWidth"`
	TabsScrollLeft  float64 `json:"tabsScrollLeft"`
	LastLeft        float64 `json:"lastLeft"`
	LastRight       float64 `json:"lastRight"`
	LastState       string  `json:"lastState"`
}

func TestBacklogCreateMenuScrollsHiddenSourceTabIntoView(t *testing.T) {
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

	page, err := browser.NewPage(pw.BrowserNewPageOptions{
		Viewport: &pw.Size{Width: 1280, Height: 800},
	})
	require.NoError(t, err)

	_, err = page.Goto(baseURL + "/iframe.html?id=" + fourSourceStoryID + "&viewMode=story")
	require.NoError(t, err)

	trigger := page.GetByTestId("lines-backlog-create")
	require.NoError(t, trigger.WaitFor(pw.LocatorWaitForOptions{Timeout: pw.Float(20000)}))
	require.NoError(t, trigger.Click())

	tabs := page.GetByTestId("lines-backlog-create-tabs")
	require.NoError(t, tabs.WaitFor(pw.LocatorWaitForOptions{Timeout: pw.Float(10000)}))

	before := waitForLayout(t, page, func(layout tabLayout) error {
		if layout.TabsRight > layout.MenuRight+1 {
			return fmt.Errorf("tab strip paints past the menu: strip right %.1f, menu right %.1f", layout.TabsRight, layout.MenuRight)
		}
		if layout.TabsScrollWidth <= layout.TabsClientWidth+8 {
			return fmt.Errorf("four source tabs fit in the strip, so the hidden-tab check is not real")
		}
		if layout.LastRight <= layout.TabsRight+1 {
			return fmt.Errorf("expected Sentry tab to start outside the visible strip")
		}
		return nil
	})
	require.Greater(t, before.LastRight, before.TabsRight+1)

	_, err = tabs.Evaluate(`element => { element.scrollLeft = element.scrollWidth; }`, nil)
	require.NoError(t, err)

	after := waitForLayout(t, page, func(layout tabLayout) error {
		if layout.TabsScrollLeft <= 0 {
			return fmt.Errorf("tab strip did not scroll")
		}
		if layout.LastLeft < layout.TabsLeft-1 || layout.LastRight > layout.TabsRight+1 {
			return fmt.Errorf("Sentry tab is not inside the tab strip after scroll: tab %.1f-%.1f, strip %.1f-%.1f", layout.LastLeft, layout.LastRight, layout.TabsLeft, layout.TabsRight)
		}
		if layout.LastLeft < layout.MenuLeft-1 || layout.LastRight > layout.MenuRight+1 {
			return fmt.Errorf("Sentry tab is not inside the menu after scroll")
		}
		return nil
	})
	require.Greater(t, after.TabsScrollLeft, 0.0)

	hiddenTab := page.GetByTestId(hiddenTabID)
	require.NoError(t, hiddenTab.Click(pw.LocatorClickOptions{Timeout: pw.Float(5000)}))
	require.NoError(t, hiddenTab.WaitFor(pw.LocatorWaitForOptions{
		Timeout: pw.Float(5000),
		State:   pw.WaitForSelectorStateVisible,
	}))

	selected := waitForLayout(t, page, func(layout tabLayout) error {
		if layout.LastState != "active" {
			return fmt.Errorf("Sentry tab is not selected")
		}
		if layout.LastRight > layout.MenuRight+1 {
			return fmt.Errorf("selected tab paints past the menu")
		}
		return nil
	})
	require.Equal(t, "active", selected.LastState)
}

func waitForLayout(t *testing.T, page pw.Page, check func(tabLayout) error) tabLayout {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	var last error
	var layout tabLayout
	for time.Now().Before(deadline) {
		layout, last = readTabLayout(page)
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

func readTabLayout(page pw.Page) (tabLayout, error) {
	raw, err := page.Evaluate(`() => {
		const menu = document.querySelector('[data-testid="lines-backlog-create-menu"]');
		const tabs = document.querySelector('[data-testid="lines-backlog-create-tabs"]');
		const last = document.querySelector('[data-testid="lines-backlog-create-tab-intake-sentry"]');
		if (!menu || !tabs || !last) {
			return null;
		}
		const menuBox = menu.getBoundingClientRect();
		const tabsBox = tabs.getBoundingClientRect();
		const lastBox = last.getBoundingClientRect();
		return {
			menuLeft: menuBox.left,
			menuRight: menuBox.right,
			tabsLeft: tabsBox.left,
			tabsRight: tabsBox.right,
			tabsScrollWidth: tabs.scrollWidth,
			tabsClientWidth: tabs.clientWidth,
			tabsScrollLeft: tabs.scrollLeft,
			lastLeft: lastBox.left,
			lastRight: lastBox.right,
			lastState: last.getAttribute("data-state"),
		};
	}`)
	if err != nil {
		return tabLayout{}, err
	}
	if raw == nil {
		return tabLayout{}, fmt.Errorf("create menu tabs are not in the document")
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return tabLayout{}, err
	}
	var layout tabLayout
	if err := json.Unmarshal(encoded, &layout); err != nil {
		return tabLayout{}, err
	}
	return layout, nil
}
