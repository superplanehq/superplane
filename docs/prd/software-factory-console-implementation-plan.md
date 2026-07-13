# Software Factory Console Implementation Plan

> **For agentic workers:** Implement task-by-task. Steps use checkbox syntax for tracking.

**Goal:** Register Spotlight as a real console panel and ship a `Pages/AppPage/Console` → **Software Factory** Storybook story (AI vs human throughput narrative).

**Architecture:** Port prototype Spotlight widget + content mapping onto current `main` patterns (mirror Scorecard: `panelTypes` → card → `useWidgetData` → top-row props). Reuse Scorecard/Table/Chart/Number for the rest of the factory board. Hand-craft an AppPage fixture like SuperPlane SaaS.

**Tech Stack:** React/Vite Storybook, existing console panelTypes + `pkg/yaml/console.go`, AppPageHarness fixtures.

**Spec:** [docs/prd/software-factory-console-storybook.md](software-factory-console-storybook.md)

---

## File map

| Create | Responsibility |
|--------|----------------|
| `web_src/.../widget/WidgetSpotlight.tsx` | Presentational spotlight banner |
| `web_src/.../widget/WidgetSpotlight.stories.tsx` | Widget stories |
| `web_src/.../spotlightContent.ts` | Content model, `spotlightPropsFromContent`, validate |
| `web_src/.../spotlightContent.spec.ts` | Unit tests |
| `web_src/.../SpotlightPanelForm.tsx` | Panel editor form |
| `web_src/.../SpotlightPanelCard.tsx` | Card + `useWidgetData` + `rows[0]` binding |
| `web_src/.../panelTypes.spotlight.spec.ts` | Normalize/validate FE |
| `web_src/.../consoleYaml.spotlight.spec.ts` | YAML round-trip FE |
| `web_src/.../__fixtures__/console/softwareFactory.json` | Canvas + consoleYaml + memory |
| Modify | |
| `panelTypes.ts`, `ConsolePanelCards.tsx`, `ConsoleView.tsx` | Register spotlight |
| `pkg/yaml/console.go`, `console_test.go` | Go YAML validation |
| `consoleFixtures.ts`, `AppPageConsole.stories.tsx` | Story wiring |
| `docs/prd/console-and-widgets.md` | Spotlight docs |

---

### Task 1: Port Spotlight widget + content model

- [ ] Copy `WidgetSpotlight.tsx` + stories from `origin/prototype/new-console-panels`; fix imports for current `main` (`formatValue`, `WidgetEmptyState`, Timestamp if needed).
- [ ] Copy `spotlightContent.ts` + `spotlightContent.spec.ts`; align with `WidgetDataSource` / `compileFieldResolver` on main.
- [ ] Run: `docker compose -f docker-compose.dev.yml exec -T app sh -c 'cd web_src && npx vitest run src/pages/app/console/spotlightContent.spec.ts'`
- [ ] Commit: `feat: port spotlight widget and content model`

### Task 2: Register Spotlight panel (FE)

- [ ] Add `"spotlight"` to `PANEL_TYPES`, `PANEL_TYPE_META`, `templateForPanelType`, `validatePanelContent`.
- [ ] Add `SpotlightPanelForm.tsx` (from prototype, adapted) and `SpotlightPanelCard.tsx` mirroring `ScorecardPanelCard` but binding `spotlightPropsFromContent(content, rows[0])`.
- [ ] Wire `ConsolePanelCards` + `ConsoleView` icon (`Rocket` or `Sparkles`).
- [ ] Add `panelTypes.spotlight.spec.ts` + `consoleYaml.spotlight.spec.ts`.
- [ ] Commit: `feat: register spotlight console panel`

### Task 3: Register Spotlight panel (Go YAML)

- [ ] Add `ConsolePanelTypeSpotlight`, allowlist, `validateSpotlightPanelContent` in `pkg/yaml/console.go`.
- [ ] Tests in `pkg/yaml/console_test.go`.
- [ ] Run: `make test PKG_TEST_PACKAGES=./pkg/yaml`
- [ ] Commit: `feat: validate spotlight panels in console YAML`

### Task 4: Software Factory AppPage fixture + story

- [ ] Create `softwareFactory.json` with layout: spotlight + scorecards + rich tables + chart/numbers for AI vs human narrative.
- [ ] Seed memory namespaces with coherent fake data (authors, AI flags, spend, PR metrics).
- [ ] Register in `consoleFixtures.ts`; add `SoftwareFactory` story in `AppPageConsole.stories.tsx`.
- [ ] Commit: `feat: add Software Factory console Storybook fixture`

### Task 5: Docs + polish

- [ ] Document spotlight in `docs/prd/console-and-widgets.md`.
- [ ] `make format.js && make format.go`; run targeted vitest + yaml tests.
- [ ] Commit: `docs: document spotlight console panel`

---

## Verification

- Storybook: `Pages/AppPage/Console` → Software Factory loads offline.
- Spotlight empty/loading/success states covered by widget stories + content specs.
- Existing console stories unchanged.
