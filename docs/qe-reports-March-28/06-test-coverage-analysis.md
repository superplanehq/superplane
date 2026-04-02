# SuperPlane Test Suite & Coverage Analysis Report

**Date**: 2026-03-28
**Scope**: All test layers — Go backend, React frontend, Python agent, E2E
**Total Test Files**: 534

---

## Executive Summary

SuperPlane has **strong backend test coverage** with 493 Go test files across `pkg/`, but a **critically thin frontend test layer** with only 7 spec files covering 716+ source files (~1% file-level coverage). The E2E suite is well-structured but carries flakiness risk from 148 `Sleep` calls. Significant gaps exist in security, accessibility, performance, contract, and visual regression testing.

---

## 1. Test Inventory

| Category | Framework | Count | Notes |
|---|---|---|---|
| Go Unit Tests | Go testing + testify | 493 | All in `pkg/` |
| Go E2E Tests | Go + Playwright-go | 25 | In `test/e2e/` |
| Frontend Unit Tests | Vitest 4.0.18 | 7 | In `web_src/src/` |
| Python Agent Tests | Pytest | 9 | In `agent/tests/` |
| Storybook Stories | Storybook 9 | 73 | Visual documentation (no assertions) |
| Consumer Tests | Go test helper | 1 | AMQP consumer |

### All 7 Frontend Test Files

1. `web_src/src/components/AutoCompleteInput/core.spec.ts`
2. `web_src/src/pages/workflowv2/autoLayout.spec.ts`
3. `web_src/src/pages/workflowv2/conflictResolverUtils.spec.ts`
4. `web_src/src/pages/workflowv2/mappers/http.spec.ts`
5. `web_src/src/utils/errors.spec.ts`
6. `web_src/src/utils/usageLimits.spec.ts`
7. `web_src/src/utils/withOrganizationHeader.spec.ts`

---

## 2. Test Quality Assessment

### Naming Conventions
- **Frontend (GOOD)**: Descriptive `it("does X when Y")` naming consistently
- **Backend Go (MIXED)**: Some double-underscore style (`Test__AESGCMEncryptor`), some conventional
- **E2E (GOOD)**: BDD-style with step objects

### AAA Pattern Adherence
- **Frontend (EXCELLENT)**: All 7 files follow clear Arrange-Act-Assert
- **Backend (GOOD)**: Clear setup, action, assertion phases
- **E2E (GOOD)**: Step-based patterns mapping to Given-When-Then

### Assertion Quality
- **Frontend (EXCELLENT)**: Specific matchers (`toContain`, `toEqual`, `toMatchObject`, `not.toThrow`)
- **Backend (GOOD)**: Consistent testify usage (`require.NoError`, `assert.Equal`)
- **E2E (GOOD)**: Custom `session.AssertVisible`, `session.AssertText`

### Flakiness Indicators
- **E2E (HIGH RISK)**: 148 `Sleep` calls. `--rerun-fails=3` confirms flakiness is operational.
- **Frontend (LOW RISK)**: No sleeps or timeouts
- **Backend (LOW RISK)**: No time-dependent patterns

---

## 3. Coverage Gaps

### 3.1 Frontend: CRITICAL (~1% File Coverage)

**Pages with ZERO tests (all 8 page directories untested):**

| Page | Files | Risk |
|---|---|---|
| `pages/admin/` | 11 files | HIGH (security-sensitive) |
| `pages/auth/` | 6 files | CRITICAL (authentication flow) |
| `pages/canvas/` | 4 files | HIGH (core product) |
| `pages/home/` | 4 files | MEDIUM |
| `pages/organization/settings/` | 19 files | HIGH |
| `pages/workflowv2/` | ~20 non-tested files | HIGH (core editor) |

**Other untested frontend layers:**

| Layer | Source Files | Tests | Coverage |
|---|---|---|---|
| Components | 33 directories | 1 test | 3% |
| UI Layer | 146 files | 0 tests | 0% |
| Hooks | 16 files | 0 tests | 0% |
| Stores | 1 file | 0 tests | 0% |
| Contexts | 2 files | 0 tests | 0% |
| Lib | 5 files | 0 tests | 0% |
| Utils | 14 files | 3 tested | 21% |

### 3.2 Backend: Packages Without Tests

| Package | Risk |
|---|---|
| `pkg/core` | HIGH (core business logic) |
| `pkg/database` | HIGH (data integrity) |
| `pkg/config` | HIGH (misconfiguration cascades) |
| `pkg/secrets` | CRITICAL (security) |
| `pkg/server` | MEDIUM |
| `pkg/retry` | MEDIUM |
| `pkg/web`, `pkg/widgets`, `pkg/templates` | LOW-MEDIUM |

### 3.3 gRPC Actions: 9 of ~17 Packages Have No Tests

Missing: `agents`, `blueprints`, `components`, `integrations`, `me`, `messages`, `serviceaccounts`, `triggers`, `widgets`

### 3.4 Integration Coverage: 42/43 Tested (Excellent)

Only `hetzner` (0/8 files) lacks tests. This is a strength.

---

## 4. Test Pyramid Health

| Layer | Count | Percentage | Ideal |
|---|---|---|---|
| Unit (Go + Frontend + Python) | 509 | 89.4% | 70% |
| Integration (gRPC actions) | ~35 | 6.1% | 20% |
| E2E (Playwright-Go) | 25 | 4.4% | 10% |

**Assessment:** Pyramid shape is correct (not inverted), but:
- Over-indexed on unit tests at expense of integration layer
- Integration layer too thin (gRPC action tests underrepresented)
- **Frontend pyramid collapses entirely** — only layer is Storybook + E2E

---

## 5. Test Infrastructure

### CI/CD
- **GitHub Actions**: Only PR title validation and release notifications. **No test execution in GitHub Actions.**
- **Semaphore CI** (inferred): Primary CI with sharded E2E execution
- **gotestsum**: JUnit XML reporting, `--rerun-fails=3`
- **Docker-based**: All tests via `docker-compose` with test database

### Parallelization
- **Backend**: `-p 1` (serial) — likely due to shared DB state
- **E2E**: Shard-based parallelism across CI workers (well-implemented)
- **Frontend**: Vitest native (only 7 files, not a concern)

### Test Data Management
- **E2E**: Clean-slate per test (`resetDatabase()` + `setupUserAndOrganization()`)
- **Backend**: Real DB (`superplane_test`), VCR cassettes for HTTP recording
- **Fixtures**: Minimal (1 file), data created programmatically

### Mock/Stub Patterns
- **Backend**: VCR (`go-vcr`) for HTTP API mocking
- **Frontend**: Minimal mocking configured
- **E2E**: Fully integrated (real backend + real Vite dev server)

---

## 6. E2E Test Quality

### Step Object Pattern: WELL IMPLEMENTED
- `TestLoginPageSteps`, `CanvasPageSteps`, `CanvasSteps`, `TestSession` — mature reusable patterns

### Browser Coverage
- **Chromium only** — no Firefox or WebKit
- **Desktop only** (2560x1440) — no mobile viewport testing

### Covered User Journeys
Authentication, canvas CRUD, organization management, admin ops, versioning, change requests

### Missing User Journeys
Integration setup, canvas execution end-to-end, custom component builder, onboarding, multi-org switching, import/export beyond basic YAML

---

## 7. Missing Test Categories

| Category | Status | Priority |
|---|---|---|
| Security tests | ABSENT | P0 |
| Performance tests / benchmarks | ABSENT | P1 |
| Accessibility tests (axe-core) | ABSENT | P2 |
| Contract tests (Pact/OpenAPI) | ABSENT | P2 |
| Visual regression tests | ABSENT | P2 |
| Mutation testing | ABSENT | P3 |

---

## 8. Prioritized Recommendations

### P0 — Critical

1. **Frontend unit test coverage**: Prioritize `pages/auth/`, `lib/expressionParser.ts`, `lib/exprEvaluator.ts`, custom hooks (`useCanvasData`, `useCanvasWebsocket`), `utils/canvasLinter.ts`
2. **Security test gaps**: Add tests for `pkg/secrets`, `pkg/authorization` boundary cases, JWT edge cases
3. **Backend foundational packages**: Test `pkg/core`, `pkg/database`, `pkg/config`

### P1 — High Priority

4. **Reduce E2E flakiness**: Replace 148 `Sleep` calls with proper waits (follow `WaitForCanvasSaveStatusSaved` pattern)
5. **gRPC action test coverage**: 9 untested action packages
6. **Integration test layer**: Service-level tests between gRPC, business logic, and database
7. **Frontend component testing**: Start with most interactive components

### P2 — Medium Priority

8. Cross-browser E2E (Firefox + WebKit)
9. API contract testing (OpenAPI validation)
10. Accessibility testing (axe-core in Storybook/Vitest)
11. Visual regression testing (leverage 73 Storybook stories)
12. Go benchmarks for critical paths

### P3 — Strategic

13. Mutation testing (Stryker for TS)
14. Mobile viewport E2E tests
15. Test data factories
16. CI/CD test pipeline in GitHub Actions

---

## Key Metrics

| Metric | Value | Assessment |
|---|---|---|
| Go test file:source ratio | 493:1241 (40%) | Good |
| Frontend test:source ratio | 7:716 (~1%) | **CRITICAL** |
| Backend packages without tests | 12/35 (34%) | Needs work |
| All frontend pages untested | 8/8 (100%) | **CRITICAL** |
| All frontend hooks untested | 16/16 (100%) | **CRITICAL** |
| Entire UI layer untested | 146/146 (100%) | **CRITICAL** |
| E2E Sleep calls | 148 | High flakiness |
| Browser coverage | Chromium only | Single browser |
| Missing test categories | 6 (security, perf, a11y, contract, visual, mutation) | Significant gaps |

---
*Generated by AQE v3 Test Architect Agent*
