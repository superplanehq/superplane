# End-to-End (E2E) Testing Guide

This document explains how to write and run E2E tests for Superplane. Tests
are written in Go and use Playwright via the `playwright-go` bindings to
drive the UI against a locally started application server and Vite dev
server.

## The Most Important Thing: Behavior‑First Tests (BDD)

Write tests that describe behavior a user would observe. Keep UI mechanics
hidden inside step methods. Name steps like you would in Gherkin:
`Given…`, `When…`, `Then…`.

Golden rules:

- Name steps clearly: `GivenACanvasExists`, `WhenIAddANoopNodeNamed`, `ThenISeeNodeTitled`.
- The test body should read like a narrative; no raw selectors in the test function.
- Keep implementation inside step methods on a `Steps` struct.
- Assert observable outcomes: visible text, enabled/disabled actions, persisted records.
- Prefer stable selectors (data-testid) inside steps; avoid brittle DOM traversal.

Good example (narrative + steps):

```go
t.Run("adding a noop node", func(t *testing.T) {
  steps := &NoopSteps{t: t}

  steps.start()
  steps.givenACanvasExists()
  steps.whenIVisitTheCanvas()
  steps.whenIAddANoopNodeNamed("Hello")
  steps.thenISeeNodeTitled("Hello")
)
```

What NOT to do (anti‑patterns):

```go
func TestNoopBad(t *testing.T) {
    p := ctx.NewSession(t)
    p.Start(); p.Login()
    p.Visit("/" + p.orgID + "/workflows/123")

    // Fragile CSS and structural assertions
    el := p.Page().Locator(".canvas .node:nth-child(2) .title")
    _ = el.Click()
    _ = p.Page().Locator("input[name=name]").Fill("Hello")

    // Arbitrary sleep instead of waiting for state
    p.Sleep(2000)
    if count, _ := p.Page().Locator(".node").Count(); count != 3 {
        t.Fatal("expected 3 nodes")
    }
}
```

Prefer instead to hide these mechanics in step methods that use stable
`data-testid` selectors and explicit waits.

## Writing a New E2E Test (Pattern)

1. Create a spec under `test/e2e/` ending with `_test.go`.
2. Use a Steps struct and Cucumber‑style method names. The test composes steps; step methods do the work.

Example skeleton:

```go
package e2e

import (
    "testing"
    q "github.com/superplanehq/superplane/test/e2e/queries"
)

func TestExampleCanvasFlow(t *testing.T) {
    steps := &ExampleSteps{t: t}
    t.Run("create and save a canvas", func(t *testing.T) {
        steps.start()
        steps.givenIAmOnTheHomePage()
        steps.whenICreateACanvasNamed("My Canvas")
        steps.thenTheCanvasIsPersisted("My Canvas")
    })
}

type ExampleSteps struct {
    t       *testing.T
    session *TestSession
}

func (s *ExampleSteps) start() {
    s.session = ctx.NewSession(s.t)
    s.session.Start()
    s.session.Login()
}

func (s *ExampleSteps) givenIAmOnTheHomePage() {
    s.session.Visit("/" + s.session.orgID + "/")
}

func (s *ExampleSteps) whenICreateACanvasNamed(name string) {
    s.session.Click(q.Text("New Canvas"))
    s.session.FillIn(q.TestID("canvas-name-input"), name)
    s.session.Click(q.Text("Create canvas"))
}

func (s *ExampleSteps) thenTheCanvasIsPersisted(name string) {
    // lookup via models and assert
}
```

Selectors: prefer the `queries` helpers

- `q.TestID("…")` uses `data-testid="…"` and is most stable
- `q.Text("…")` for visible text when appropriate
- `q.Locator("css or :has-text()")` for advanced cases only

Common test IDs:

- Canvas: `canvas-drop-area`, `save-canvas-button`
- Modals/Forms: `canvas-name-input`, `component-name-input`, `add-node-button`
- Building blocks: `building-block-<name>` (e.g., `building-block-noop`,
  `building-block-approval`)

## Quick Commands

- One‑time setup: `make test.e2e.setup`
- Run all E2E tests: `make test.e2e`
- Open a shell in the test container: `make test.shell`
- Screenshots are saved to `tmp/screenshots/` on your host

Note: The `test.e2e` target uses `gotestsum` with `E2E_TEST_PACKAGES`
to run tests and emit a JUnit report. Use the `-run` flag to filter
tests within those packages.

## Project Layout

- E2E tests live under `test/e2e/`
  - `test/e2e/main_test.go` – global test bootstrap (server, Vite,
    Playwright)
  - `test/e2e/test_context.go` – process‑level setup: environment,
    browser, server
  - `test/e2e/test_session.go` – per‑test utilities: DB reset, auth,
    page helpers
  - `test/e2e/queries/` – small locator helpers for stable selectors
  - Example specs: `test/e2e/home_page_test.go`,
    `test/e2e/canvas_page_test.go`

## How E2E Bootstraps

- TestMain (`test/e2e/main_test.go`) creates a shared `TestContext` once for
  the run.
- TestContext (`test/e2e/test_context.go`):
  - Sets env vars for a self‑contained server (e.g., `DB_NAME=superplane_test`,
    `START_WEB_SERVER=yes`, `BASE_URL=http://127.0.0.1:8001`).
  - Starts the Go app server in‑proc via `server.Start()`.
  - Starts Vite dev server on `127.0.0.1:5173` and waits until it’s
    reachable.
  - Starts Playwright and launches a Chromium browser + context.
- Each test creates a `TestSession` which:
  - Truncates the test DB tables to a clean state.
  - Creates/ensures a user + organization and sets up RBAC.
  - Logs in by setting the `account_token` cookie with a short‑lived JWT.
  - Exposes concise helpers to drive and assert the UI.

## Useful TestSession Helpers

- Navigation and assertions:
  - `Visit(path string)` – visit a route (prefix it with `"/" + orgID +
"/"` when needed)
  - `AssertText(text string)` – wait until text is visible
  - `AssertVisible(q)` – wait until a locator is visible
  - `AssertDisabled(q)` – assert a button or control is disabled
  - `AssertURLContains(part string)` – URL assertion
- Interactions:
  - `Click(q)`, `FillIn(q, value)`
  - `DragAndDrop(source, target, offsetX, offsetY)` – helpful for canvas
    actions
- Utilities:
  - `TakeScreenshot()` – saves to `tmp/screenshots/<test>-<ts>.png`
  - `Sleep(ms int)` – use sparingly; prefer waiting for explicit UI states

## Asserting the Database

It’s common to assert server‑side state directly using model helpers
(e.g., `pkg/models`). Examples can be found in:

- `test/e2e/home_page_test.go` – asserts canvas and component persisted
- `test/e2e/canvas_page_test.go` – looks up workflow IDs to navigate
  directly to pages

Follow general repository testing conventions:

- Tests end with `_test.go`
- Prefer early returns in helpers
- When needing specific timestamps, derive from `time.Now()` rather than
  fixed `time.Date`

## Frontend Notes

- Add `data-testid` attributes to critical UI controls to keep tests
  stable.
- After editing frontend code, run `make check.build.ui` and
  `make format.js`.

## Troubleshooting

- “Playwright not found” – run `make test.e2e.setup` to install browsers,
  or run the `test.e2e` target which installs them automatically if
  missing.
- “Element not found” – add a `data-testid` or wait for a visible state
  before interacting.
- “DB state leaks between tests” – each `TestSession.Start()` truncates
  tables. Ensure you create a fresh session per sub‑test.
- Need to debug? Add `TakeScreenshot()` calls or run a subset with `-run`
  to iterate faster.

## Related Files

- `test/e2e/main_test.go`
- `test/e2e/test_context.go`
- `test/e2e/test_session.go`
- `test/e2e/queries/query.go`
- `Makefile` – targets: `test.e2e.setup`, `test.e2e`, `test.shell`
