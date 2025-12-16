# End-to-End (E2E) Testing Guide

This document explains how to write and run E2E tests for Superplane.

## Table of Contents

- [Overview](#overview)
- [How to run e2e tests](#how-to-run-e2e-tests)
- [How to write good tests](#how-to-write-good-tests)
- [Anti-patterns](#anti-patterns)
- [Writing a New E2E Test (Pattern)](#writing-a-new-e2e-test-pattern)
- [Debugging](#debugging)

## Overview

Tests are written in Go and use Playwright via the `playwright-go` bindings to
drive the UI against a locally started application server.

All e2e tests live under the `test/e2e` directory.

## How to run e2e tests

Before running the tests, run the setup steps:

```
make test.setup
make setup.playwright
```

To run all e2e tests (takes 20m+):

```
make test.e2e
```

To run an individual test:

```
make test.e2e FILE=test/e2e/canvas_page_test.go LINE=19
```

To run a test from VSCode, set up the following keybindings (cmd+shift+p keybidings):

```json
  {
    "key": "cmd+t",
    "command": "workbench.action.tasks.runTask",
    "args": "Test Current Line"
  },
  {
    "key": "cmd+shift+t",
    "command": "workbench.action.tasks.runTask",
    "args": "Test Current File"
  },
```

Then to run a single test, navigate to the test file you want to run, move your text
cursor inside of the test and press `cmd+t` to run the test.

## How to write good tests

Write tests that describe behavior a user would observe. Keep UI mechanics
hidden inside step methods. Name steps like you would in Gherkin:
`Given...`, `When...`, `Then...`, `Assert...`

Golden rules:

- Name steps clearly: `givenACanvasExists`, `addANoopNode`, `assertTheNodeIsSaved`
- The test body should read like a narrative; no raw selectors in the test function.
- Keep implementation inside step methods on a `steps` struct.
- Assert observable outcomes: visible text, enabled/disabled actions, persisted records.
- Prefer stable selectors (data-testid) inside steps; avoid brittle DOM traversal.

Good example (narrative + steps):

```go
func TestNoopComponent(t *testing.T) {
  steps := &NoopSteps{t: t}

  t.Run("adding a noop node", func(t *testing.T) {
    steps.start()
    steps.givenACanvasExists()
    steps.visitTheCanvas()
    steps.addANoopNodeNamed("Hello")
    steps.assertNodeIsSaved("Hello")
  }
}
```

## Anti-Patterns

### Bad: Tests are not written like a narative, too low level

```go
func TestNoopBad(t *testing.T) {
    p := ctx.NewSession(t)

    p.Start();
    p.Login();
    p.Visit("/" + p.orgID + "/workflows/123")

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

### Bad: Tests are using DOM traversal

```go
func TestNoopBad(t *testing.T) {
    //...

    // Fragile CSS and structural assertions
    el := p.Page().Locator(".canvas .node:nth-child(2) .title")
    _ = el.Click()
    _ = p.Page().Locator("input[name=name]").Fill("Hello")

    //...
}
```

Prefer instead to hide these mechanics in step methods that use stable
`data-testid` selectors.

Use the `test/e2e/helpers/query.go` for lookup:

- `q.TestID("…")` uses `data-testid="…"` and is most stable
- `q.Text("…")` for visible text when appropriate
- `q.Locator("css or :has-text()")` for advanced cases only

Common test IDs:

- Canvas: `canvas-drop-area`, `save-canvas-button`
- Modals/Forms: `canvas-name-input`, `component-name-input`, `add-node-button`
- Building blocks: `building-block-<name>` (e.g., `building-block-noop`, `building-block-approval`)

## Writing a New E2E Test (Pattern)

1. Create a spec under `test/e2e/` ending with `_test.go`.
2. Use a steps struct and Cucumber‑style method names. The test composes steps; step methods do the work.

Example skeleton:

```go
package e2e

import (
    "testing"
    q "github.com/superplanehq/superplane/test/e2e/queries"
)

func TestExampleCanvasFlow(t *testing.T) {
    steps := &exampleSteps{t: t}

    t.Run("create and save a canvas", func(t *testing.T) {
        steps.start()
        steps.givenIAmOnTheHomePage()
        steps.createACanvas("My Canvas")
        steps.assertCanvasIsPersisted("My Canvas")
    })
}

type exampleSteps struct {
    t       *testing.T
    session *TestSession
}

func (s *exampleSteps) start() {
    s.session = ctx.NewSession(s.t)
    s.session.Start()
    s.session.Login()
}

func (s *exampleSteps) givenIAmOnTheHomePage() {
    s.session.Visit("/" + s.session.orgID + "/")
}

func (s *exampleSteps) createACanvasNamed(name string) {
    s.session.Click(q.Text("New Canvas"))
    s.session.FillIn(q.TestID("canvas-name-input"), name)
    s.session.Click(q.Text("Create canvas"))
}

func (s *exampleSteps) asseertCanvasIsPersisted(name string) {
    // lookup via models and assert
}
```

## Debugging

### Screenshots

E2E tests automatically capture screenshots on test failures. These are saved to
the `tmp/screenshots` directory in the root of this repository.

To manually capture a screenshot during a test:

```go
func (s *exampleSteps) exampleStep() {
  // ...
  s.session.TakeScreenshot()
  // ...
}
```

Screenshots are particularly useful when debugging failing tests to see the
actual state of the UI at the point of failure.
