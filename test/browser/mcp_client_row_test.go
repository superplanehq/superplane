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

const mcpServerConfiguredStoryID = "factories-pages-settings-mcp-server--configured"
const longMCPClientRevokeID = "superplane-mcp-client-revoke-mcp-client-codex"

type mcpClientRowLayout struct {
	CardLeft       float64 `json:"cardLeft"`
	CardRight      float64 `json:"cardRight"`
	ListWidth      float64 `json:"listWidth"`
	ButtonLeft     float64 `json:"buttonLeft"`
	ButtonRight    float64 `json:"buttonRight"`
	ButtonWidth    float64 `json:"buttonWidth"`
	ButtonHit      bool    `json:"buttonHit"`
	NameTruncated  bool    `json:"nameTruncated"`
	CursorTimeLeft float64 `json:"cursorTimeLeft"`
	CodexTimeLeft  float64 `json:"codexTimeLeft"`
}

func TestMCPClientRevokeStaysInsideNarrowCard(t *testing.T) {
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

	// Phone settings hide both sidebars below 768px, so a 760px viewport is a
	// wide card. 800px stays on the desktop shell and still squeezes the list.
	page, err := browser.NewPage(pw.BrowserNewPageOptions{
		Viewport: &pw.Size{Width: 800, Height: 900},
	})
	require.NoError(t, err)

	_, err = page.Goto(baseURL + "/iframe.html?id=" + mcpServerConfiguredStoryID + "&viewMode=story")
	require.NoError(t, err)

	revoke := page.GetByTestId(longMCPClientRevokeID)
	require.NoError(t, revoke.WaitFor(pw.LocatorWaitForOptions{Timeout: pw.Float(90000)}))
	require.NoError(t, revoke.ScrollIntoViewIfNeeded())

	narrow := waitForMCPClientRowLayout(t, page, func(layout mcpClientRowLayout) error {
		if layout.ListWidth >= 480 {
			return fmt.Errorf("client list is %.1f px wide; viewport does not reproduce a narrow card", layout.ListWidth)
		}
		return mcpClientRevokeInsideCard(layout)
	})
	require.Less(t, narrow.ListWidth, 480.0)
	require.True(t, narrow.ButtonHit)
	require.True(t, narrow.NameTruncated)

	require.NoError(t, page.SetViewportSize(1280, 900))
	require.NoError(t, revoke.ScrollIntoViewIfNeeded())
	wide := waitForMCPClientRowLayout(t, page, func(layout mcpClientRowLayout) error {
		if layout.ListWidth < 500 {
			return fmt.Errorf("client list is %.1f px wide; viewport is not wide enough to check column alignment", layout.ListWidth)
		}
		if err := mcpClientRevokeInsideCard(layout); err != nil {
			return err
		}
		if math.Abs(layout.CursorTimeLeft-layout.CodexTimeLeft) > 1 {
			return fmt.Errorf("time columns do not line up: cursor %.1f, codex %.1f", layout.CursorTimeLeft, layout.CodexTimeLeft)
		}
		return nil
	})
	require.InDelta(t, wide.CursorTimeLeft, wide.CodexTimeLeft, 1.0)
}

func mcpClientRevokeInsideCard(layout mcpClientRowLayout) error {
	if layout.ButtonWidth < 40 {
		return fmt.Errorf("Revoke is clipped: width %.1f", layout.ButtonWidth)
	}
	if layout.ButtonLeft < layout.CardLeft-1 || layout.ButtonRight > layout.CardRight+1 {
		return fmt.Errorf(
			"Revoke sits outside the card: button %.1f-%.1f, card %.1f-%.1f",
			layout.ButtonLeft,
			layout.ButtonRight,
			layout.CardLeft,
			layout.CardRight,
		)
	}
	if !layout.ButtonHit {
		return fmt.Errorf("Revoke is not reachable at its center")
	}
	if !layout.NameTruncated {
		return fmt.Errorf("long client name does not truncate")
	}
	return nil
}

func waitForMCPClientRowLayout(t *testing.T, page pw.Page, check func(mcpClientRowLayout) error) mcpClientRowLayout {
	t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	var last error
	var layout mcpClientRowLayout
	for time.Now().Before(deadline) {
		layout, last = readMCPClientRowLayout(page)
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

func readMCPClientRowLayout(page pw.Page) (mcpClientRowLayout, error) {
	raw, err := page.Evaluate(`() => {
		const card = document.querySelector('[data-testid="superplane-mcp-server"]');
		const list = document.querySelector('[data-testid="superplane-mcp-clients-list"]');
		const button = document.querySelector('[data-testid="superplane-mcp-client-revoke-mcp-client-codex"]');
		const name = document.querySelector('[data-testid="superplane-mcp-client-name-mcp-client-codex"]');
		const cursorTime = document.querySelector('[data-testid="superplane-mcp-client-time-mcp-client-cursor-leonardo"]');
		const codexTime = document.querySelector('[data-testid="superplane-mcp-client-time-mcp-client-codex"]');
		if (!card || !list || !button || !name || !cursorTime || !codexTime) {
			return null;
		}
		const cardBox = card.getBoundingClientRect();
		const listBox = list.getBoundingClientRect();
		const buttonBox = button.getBoundingClientRect();
		const centerX = buttonBox.left + buttonBox.width / 2;
		const centerY = buttonBox.top + buttonBox.height / 2;
		const hit = document.elementFromPoint(centerX, centerY);
		return {
			cardLeft: cardBox.left,
			cardRight: cardBox.right,
			listWidth: listBox.width,
			buttonLeft: buttonBox.left,
			buttonRight: buttonBox.right,
			buttonWidth: buttonBox.width,
			buttonHit: Boolean(hit && (hit === button || button.contains(hit))),
			nameTruncated: name.scrollWidth > name.clientWidth + 1,
			cursorTimeLeft: cursorTime.getBoundingClientRect().left,
			codexTimeLeft: codexTime.getBoundingClientRect().left,
		};
	}`)
	if err != nil {
		return mcpClientRowLayout{}, err
	}
	if raw == nil {
		return mcpClientRowLayout{}, fmt.Errorf("MCP client row is not in the document")
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return mcpClientRowLayout{}, err
	}
	var layout mcpClientRowLayout
	if err := json.Unmarshal(encoded, &layout); err != nil {
		return mcpClientRowLayout{}, err
	}
	return layout, nil
}
