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

const messageSentTimeStoryID = "factories-pages-task-split-run-message-sent-time--narrow-pane"

type messageTimeLayout struct {
	PaneWidth         float64 `json:"paneWidth"`
	ScrollOverflow    float64 `json:"scrollOverflow"`
	LongTimeLeft      float64 `json:"longTimeLeft"`
	LongTimeRight     float64 `json:"longTimeRight"`
	LongTimeWidth     float64 `json:"longTimeWidth"`
	LongBubbleLeft    float64 `json:"longBubbleLeft"`
	LongBubbleRight   float64 `json:"longBubbleRight"`
	LongBodyLeft      float64 `json:"longBodyLeft"`
	LongBodyTop       float64 `json:"longBodyTop"`
	LongOpacity       float64 `json:"longOpacity"`
	LongFocused       bool    `json:"longFocused"`
	ShortTimeLeft     float64 `json:"shortTimeLeft"`
	ShortTimeRight    float64 `json:"shortTimeRight"`
	ShortTimeWidth    float64 `json:"shortTimeWidth"`
	ShortBubbleLeft   float64 `json:"shortBubbleLeft"`
	ShortBubbleRight  float64 `json:"shortBubbleRight"`
	ShortBodyTop      float64 `json:"shortBodyTop"`
	ShortOpacity      float64 `json:"shortOpacity"`
	SurveyTimeLeft    float64 `json:"surveyTimeLeft"`
	SurveyTimeRight   float64 `json:"surveyTimeRight"`
	SurveyTimeWidth   float64 `json:"surveyTimeWidth"`
	SurveyBubbleLeft  float64 `json:"surveyBubbleLeft"`
	SurveyBubbleRight float64 `json:"surveyBubbleRight"`
	AgentTimeLeft     float64 `json:"agentTimeLeft"`
	AgentTimeRight    float64 `json:"agentTimeRight"`
	AgentBodyLeft     float64 `json:"agentBodyLeft"`
	AgentBodyTop      float64 `json:"agentBodyTop"`
	AgentOpacity      float64 `json:"agentOpacity"`
	PaneLeft          float64 `json:"paneLeft"`
	PaneRight         float64 `json:"paneRight"`
}

func TestTaskMessageTimeStaysInsideNarrowPane(t *testing.T) {
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
		Viewport: &pw.Size{Width: 800, Height: 800},
	})
	require.NoError(t, err)

	_, err = page.Goto(baseURL + "/iframe.html?id=" + messageSentTimeStoryID + "&viewMode=story")
	require.NoError(t, err)
	_, err = page.Evaluate(`() => document.fonts ? document.fonts.ready : Promise.resolve()`)
	require.NoError(t, err)

	longNote := page.GetByTestId("split-run-intent-user-note").First()
	require.NoError(t, longNote.WaitFor(pw.LocatorWaitForOptions{Timeout: pw.Float(20000)}))

	rest := waitForMessageTimeLayout(t, page, func(layout messageTimeLayout) error {
		if err := narrowPane(layout); err != nil {
			return err
		}
		return timesHiddenBesideBubbles(layout)
	})
	require.LessOrEqual(t, rest.LongOpacity, 0.05)
	require.LessOrEqual(t, rest.ShortOpacity, 0.05)

	require.NoError(t, longNote.GetByTestId("work-order-description").Hover())
	hovered := waitForMessageTimeLayout(t, page, func(layout messageTimeLayout) error {
		if err := timeVisibleInsidePane(layout.LongOpacity, layout.LongTimeLeft, layout.LongTimeRight, layout.LongTimeWidth, layout); err != nil {
			return err
		}
		return sameMessagePosition(layout.LongBodyLeft, layout.LongBodyTop, rest.LongBodyLeft, rest.LongBodyTop)
	})
	require.InDelta(t, rest.LongBodyLeft, hovered.LongBodyLeft, 1)
	require.InDelta(t, rest.LongBodyTop, hovered.LongBodyTop, 1)

	require.NoError(t, page.Mouse().Move(700, 20))
	hidden := waitForMessageTimeLayout(t, page, func(layout messageTimeLayout) error {
		if layout.LongOpacity > 0.05 {
			return fmt.Errorf("sent time stayed visible after the pointer left: opacity %.2f", layout.LongOpacity)
		}
		return sameMessagePosition(layout.LongBodyLeft, layout.LongBodyTop, rest.LongBodyLeft, rest.LongBodyTop)
	})
	require.LessOrEqual(t, hidden.LongOpacity, 0.05)

	require.NoError(t, page.Keyboard().Press("Tab"))
	focused := waitForMessageTimeLayout(t, page, func(layout messageTimeLayout) error {
		if !layout.LongFocused {
			return fmt.Errorf("keyboard focus did not reach the long note")
		}
		if err := timeVisibleInsidePane(layout.LongOpacity, layout.LongTimeLeft, layout.LongTimeRight, layout.LongTimeWidth, layout); err != nil {
			return err
		}
		return sameMessagePosition(layout.LongBodyLeft, layout.LongBodyTop, rest.LongBodyLeft, rest.LongBodyTop)
	})
	require.True(t, focused.LongFocused)

	require.NoError(t, page.Keyboard().Press("Tab"))
	blurred := waitForMessageTimeLayout(t, page, func(layout messageTimeLayout) error {
		if layout.LongFocused {
			return fmt.Errorf("long note still has keyboard focus")
		}
		if layout.LongOpacity > 0.05 {
			return fmt.Errorf("sent time stayed visible after keyboard focus left: opacity %.2f", layout.LongOpacity)
		}
		return sameMessagePosition(layout.LongBodyLeft, layout.LongBodyTop, rest.LongBodyLeft, rest.LongBodyTop)
	})
	require.LessOrEqual(t, blurred.LongOpacity, 0.05)
	require.GreaterOrEqual(t, blurred.ShortOpacity, 0.95)
}

func narrowPane(layout messageTimeLayout) error {
	if layout.PaneWidth > 240 {
		return fmt.Errorf("pane is %.1f px wide; the clip check needs the 14rem pane", layout.PaneWidth)
	}
	if layout.ScrollOverflow > 1 {
		return fmt.Errorf("chat scroll is %.1f px wider than its frame", layout.ScrollOverflow)
	}
	if err := timeInsideBubble(layout.LongTimeLeft, layout.LongTimeRight, layout.LongTimeWidth, layout.LongBubbleLeft, layout.LongBubbleRight, layout); err != nil {
		return fmt.Errorf("long note: %w", err)
	}
	if err := timeBesideBubble(layout.ShortTimeLeft, layout.ShortTimeRight, layout.ShortTimeWidth, layout.ShortBubbleLeft, layout); err != nil {
		return fmt.Errorf("short note: %w", err)
	}
	if err := timeInsideBubble(layout.SurveyTimeLeft, layout.SurveyTimeRight, layout.SurveyTimeWidth, layout.SurveyBubbleLeft, layout.SurveyBubbleRight, layout); err != nil {
		return fmt.Errorf("survey answer: %w", err)
	}
	if layout.AgentTimeLeft < layout.PaneLeft-1 || layout.AgentTimeRight > layout.PaneRight+1 {
		return fmt.Errorf("agent time sits outside the pane: %.1f-%.1f", layout.AgentTimeLeft, layout.AgentTimeRight)
	}
	return nil
}

func timesHiddenBesideBubbles(layout messageTimeLayout) error {
	if layout.LongOpacity > 0.05 || layout.ShortOpacity > 0.05 || layout.AgentOpacity > 0.05 {
		return fmt.Errorf("sent time is visible before hover or focus")
	}
	return nil
}

func timeInsideBubble(timeLeft, timeRight, timeWidth, bubbleLeft, bubbleRight float64, layout messageTimeLayout) error {
	if err := timeInsidePane(timeLeft, timeRight, timeWidth, layout); err != nil {
		return err
	}
	if timeLeft < bubbleLeft-1 || timeRight > bubbleRight+1 {
		return fmt.Errorf("sent time is not inside the message: time %.1f-%.1f, message %.1f-%.1f", timeLeft, timeRight, bubbleLeft, bubbleRight)
	}
	return nil
}

func timeBesideBubble(timeLeft, timeRight, timeWidth, bubbleLeft float64, layout messageTimeLayout) error {
	if err := timeInsidePane(timeLeft, timeRight, timeWidth, layout); err != nil {
		return err
	}
	if timeRight > bubbleLeft+1 || bubbleLeft-timeRight > 12 {
		return fmt.Errorf("sent time is not beside the message: time right %.1f, message left %.1f", timeRight, bubbleLeft)
	}
	return nil
}

func timeInsidePane(timeLeft, timeRight, timeWidth float64, layout messageTimeLayout) error {
	if timeWidth < 48 {
		return fmt.Errorf("sent time is clipped: width %.1f", timeWidth)
	}
	if timeLeft < layout.PaneLeft-1 || timeRight > layout.PaneRight+1 {
		return fmt.Errorf("sent time sits outside the pane: %.1f-%.1f, pane %.1f-%.1f", timeLeft, timeRight, layout.PaneLeft, layout.PaneRight)
	}
	return nil
}

func timeVisibleInsidePane(opacity, timeLeft, timeRight, timeWidth float64, layout messageTimeLayout) error {
	if opacity < 0.95 {
		return fmt.Errorf("sent time is not visible: opacity %.2f", opacity)
	}
	if timeWidth < 48 || timeLeft < layout.PaneLeft-1 || timeRight > layout.PaneRight+1 {
		return fmt.Errorf("visible sent time is outside the pane: %.1f-%.1f width %.1f", timeLeft, timeRight, timeWidth)
	}
	return nil
}

func sameMessagePosition(left, top, previousLeft, previousTop float64) error {
	if abs(left-previousLeft) > 1 || abs(top-previousTop) > 1 {
		return fmt.Errorf("message moved from %.1f,%.1f to %.1f,%.1f", previousLeft, previousTop, left, top)
	}
	return nil
}

func abs(value float64) float64 {
	if value < 0 {
		return -value
	}
	return value
}

func waitForMessageTimeLayout(t *testing.T, page pw.Page, check func(messageTimeLayout) error) messageTimeLayout {
	t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	var last error
	var layout messageTimeLayout
	for time.Now().Before(deadline) {
		layout, last = readMessageTimeLayout(page)
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

func readMessageTimeLayout(page pw.Page) (messageTimeLayout, error) {
	raw, err := page.Evaluate(`() => {
		const pane = document.querySelector('[data-testid="message-sent-time-pane"]');
		const scroll = document.querySelector('[data-testid="message-sent-time-scroll"]');
		const notes = [...document.querySelectorAll('[data-testid="split-run-intent-user-note"]')];
		const survey = document.querySelector('[data-testid="split-run-intent-survey-answer"]');
		const agent = document.querySelector('[data-testid="split-run-intent-agent-message"]');
		const longNote = notes[0];
		const shortNote = notes[1];
		const longBody = longNote?.querySelector('[data-testid="work-order-description"]');
		const shortBody = shortNote?.querySelector('[data-testid="work-order-description"]');
		const surveyBody = survey?.querySelector('[data-testid="split-run-intent-survey-pick"]');
		const longTime = longNote?.querySelector('time');
		const shortTime = shortNote?.querySelector('time');
		const surveyTime = survey?.querySelector('time');
		const agentTime = agent?.querySelector('time');
		if (!pane || !scroll || !longBody || !shortBody || !surveyBody || !longTime || !shortTime || !surveyTime || !agent || !agentTime) {
			return null;
		}
		const paneBox = pane.getBoundingClientRect();
		const longBubbleBox = longBody.parentElement.getBoundingClientRect();
		const shortBubbleBox = shortBody.parentElement.getBoundingClientRect();
		const surveyBubbleBox = surveyBody.getBoundingClientRect();
		const agentBox = agent.getBoundingClientRect();
		const longTimeBox = longTime.getBoundingClientRect();
		const shortTimeBox = shortTime.getBoundingClientRect();
		const surveyTimeBox = surveyTime.getBoundingClientRect();
		const agentTimeBox = agentTime.getBoundingClientRect();
		const opacity = (element) => Number(getComputedStyle(element).opacity);
		return {
			paneLeft: paneBox.left,
			paneRight: paneBox.right,
			paneWidth: paneBox.width,
			scrollOverflow: scroll.scrollWidth - scroll.clientWidth,
			longTimeLeft: longTimeBox.left,
			longTimeRight: longTimeBox.right,
			longTimeWidth: longTimeBox.width,
			longBubbleLeft: longBubbleBox.left,
			longBubbleRight: longBubbleBox.right,
			longBodyLeft: longBody.getBoundingClientRect().left,
			longBodyTop: longBody.getBoundingClientRect().top,
			longOpacity: opacity(longTime),
			longFocused: document.activeElement === longNote.parentElement,
			shortTimeLeft: shortTimeBox.left,
			shortTimeRight: shortTimeBox.right,
			shortTimeWidth: shortTimeBox.width,
			shortBubbleLeft: shortBubbleBox.left,
			shortBubbleRight: shortBubbleBox.right,
			shortBodyTop: shortBody.getBoundingClientRect().top,
			shortOpacity: opacity(shortTime),
			surveyTimeLeft: surveyTimeBox.left,
			surveyTimeRight: surveyTimeBox.right,
			surveyTimeWidth: surveyTimeBox.width,
			surveyBubbleLeft: surveyBubbleBox.left,
			surveyBubbleRight: surveyBubbleBox.right,
			agentTimeLeft: agentTimeBox.left,
			agentTimeRight: agentTimeBox.right,
			agentBodyLeft: agentBox.left,
			agentBodyTop: agentBox.top,
			agentOpacity: opacity(agentTime),
		};
	}`)
	if err != nil {
		return messageTimeLayout{}, err
	}
	if raw == nil {
		return messageTimeLayout{}, fmt.Errorf("message sent time story is not in the document")
	}
	encoded, err := json.Marshal(raw)
	if err != nil {
		return messageTimeLayout{}, err
	}
	var layout messageTimeLayout
	if err := json.Unmarshal(encoded, &layout); err != nil {
		return messageTimeLayout{}, err
	}
	return layout, nil
}
