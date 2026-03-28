# SuperPlane Quality Experience (QX) Analysis Report

**Project**: SuperPlane - Workflow Orchestration Platform
**Scope**: Frontend application at `/workspaces/superplane/web_src/src/`
**Date**: 2026-03-28
**Analysis Type**: READ-ONLY static code analysis across 7 QX dimensions
**Overall QX Score**: 71/100 (C+)

---

## 1. Error Handling UX

### 1.1 Error Boundary Implementation -- Score: 72/100

**Strengths:**
- A top-level Sentry `ErrorBoundary` wraps the entire app in `main.tsx` (line 14), providing a global crash safety net with a user-friendly `<ErrorPage />` fallback that displays "Something went wrong" with "Try Again" and "Go Home" buttons.
- Sentry integration (`sentry.ts`) is well-configured with console capture, browser API error tracking, and global handlers for unhandled rejections.

**Issues Found:**

**CRITICAL: Only one error boundary exists for the entire application.** There is no route-level or component-level error boundary. If any individual page crashes (e.g., the workflow editor, settings pages), the entire application resets to the error page, causing the user to lose all context. This is particularly damaging for the workflow editor which contains complex state.

**MODERATE: The `NotFoundPage` component at `components/NotFoundPage.tsx` uses `window.location.href = "/"` (line 23) for navigation, causing a full page reload instead of using React Router's `useNavigate`.** This destroys all in-memory state and forces a complete re-authentication flow.

**MODERATE: The catch-all route at `App.tsx` (line 116) silently redirects to `/` with `<Navigate to="/" />` instead of showing the 404 page.** Users who mistype a URL never receive feedback that their URL was wrong.

### 1.2 API Error Display -- Score: 68/100

**Strengths:**
- A centralized toast utility (`utils/toast.ts`) provides standardized `showErrorToast`, `showSuccessToast`, `showInfoToast`, and `showWarningToast` functions using Sonner.
- Usage limit errors are handled with excellent specificity through `utils/usageLimits.ts`, which maps 8 distinct usage limit error types to user-friendly messages with actionable links.
- The API interceptor (`lib/api-interceptor.ts`) handles 401 responses with redirect preservation.

**Issues Found:**

**MODERATE: Many error handlers expose raw API error messages directly to users.** For example, in `pages/workflowv2/index.tsx` (line 2532):
```typescript
const errorMessage = error?.response?.data?.message || error?.message || "Failed to save changes to the canvas";
```
The fallback to `error?.message` may expose technical JavaScript error messages.

**MODERATE: The `AccountContext` (`contexts/AccountContext.tsx`, line 65) silently swallows all errors during account fetch with an empty catch block `catch (_error) {}`.** Users experiencing network issues see no feedback.

### 1.3 Form Validation Feedback -- Score: 82/100

**Strengths:**
- The `useRealtimeValidation` hook provides debounced, real-time validation with three error types: `required`, `validation_rule`, and `visibility`.
- Validation is performance-optimized with hash-based deduplication.
- `aria-invalid` styling is consistently used across input, textarea, button, and badge components.

---

## 2. Loading States -- Score: 62/100

**Strengths:**
- A `LoadingButton` component provides consistent loading state for buttons with animated spinner.
- Auth-related pages display contextual loading messages: "Signing in...", "Creating...", "Verifying..."
- A `Skeleton` component exists for shimmer-style loading placeholders.

**Issues Found:**

**CRITICAL: Skeleton loaders are defined but barely used in the application.** Only 3 files reference the Skeleton component. Major pages use either plain "Loading..." text or a simple spinner, causing layout shifts.

**MODERATE: No optimistic updates are implemented.** Despite using TanStack Query with `useMutation`, none of the mutation hooks use `onMutate` for optimistic updates.

**MODERATE: The organization settings page has three sequential loading gates, each showing a minimal "Loading..." message, creating a multi-step loading experience.**

---

## 3. User Journey Quality

### 3.1 Workflow Creation/Editing -- Score: 70/100

**Strengths:**
- Comprehensive canvas versioning, change request workflows, auto-layout, YAML import/export.
- Unsaved changes tracking with auto-save capability.
- Real-time collaboration through WebSocket updates.

**Issues Found:**

**CRITICAL: The workflow page file is 6,589 lines long.** Significant code smell indicating potential for complex, difficult-to-maintain user flows.

**CRITICAL: No undo/redo support exists for canvas editing.** Users editing workflow canvases cannot undo node additions, deletions, or repositioning.

**MODERATE: The revert function only restores to the initial state, not to arbitrary undo steps.**

### 3.2 Navigation -- Score: 76/100

**Strengths:**
- Well-implemented Breadcrumbs component with proper ARIA.
- Organization-scoped routing provides clear URL structure.
- Dynamic page titles via `usePageTitle` hook.

### 3.3 Feedback for Destructive Actions -- Score: 60/100

**CRITICAL: The application uses `window.confirm()` for destructive operations in at least 12 locations:**
- Group deletion (`settings/Groups.tsx`, line 60)
- Service account deletion (`settings/ServiceAccounts.tsx`, line 102)
- Secret deletion (`settings/SecretDetail.tsx`, line 163)
- Role deletion (`settings/Roles.tsx`, line 50)
- Canvas version reset (`workflowv2/index.tsx`, line 4833)
- Queue item cancellation (`workflowv2/useOnCancelQueueItemHandler.ts`, line 26)
- Node deletion (`CanvasPage/index.tsx`, line 3169)

An `AlertDialog` component already exists but is only used in Storybook examples.

---

## 4. Accessibility (a11y) -- Score: 68/100

### 4.1 ARIA Attributes -- Score: 74/100

**Strengths:**
- Select component implements proper ARIA: `role="button"`, `aria-haspopup="listbox"`, `aria-expanded`.
- Switch component uses `role="switch"`, `aria-checked`, `aria-label`.
- Decorative icons consistently use `aria-hidden="true"`.
- `sr-only` class used across ~20 locations.

**Issues Found:**

**MODERATE: The custom Dialog component is missing ARIA attributes.** No `role="dialog"`, `aria-modal`, `aria-labelledby`, or Escape key handling.

**MODERATE: Switch component uses `focus:outline-none` removing visible focus indicator entirely.** Should use `focus-visible:ring-2` for keyboard accessibility.

### 4.2 Keyboard Navigation -- Score: 70/100

**MODERATE: Tab component does not implement WAI-ARIA Tab Pattern.** Missing `role="tablist"`, `role="tab"`, `aria-selected`. No arrow key navigation between tabs.

### 4.3 Color and Visual Accessibility -- Score: 65/100

**MODERATE: Inconsistent dark mode color contrast.** `text-gray-500` appears frequently without dark mode overrides, potentially failing WCAG AA.

---

## 5. Responsive Design -- Score: 55/100

**Strengths:**
- `useIsMobile` hook exists with 768px breakpoint.
- Home page grid uses responsive `grid-cols-1 md:grid-cols-2 lg:grid-cols-3`.

**Issues Found:**

**CRITICAL: No `@media` queries in `index.css`.** Responsive design relies entirely on Tailwind utilities — any component missing responsive classes won't adapt.

**CRITICAL: The workflow canvas editor has no mobile/tablet adaptations.** No mobile fallback, viewport warning, or responsive adaptation.

**MODERATE: `useIsMobile` hook imported in only 1 location** — the vast majority of the app doesn't consume mobile state.

---

## 6. Feedback and Communication

### 6.1 Toasts -- Score: 78/100
- Sonner positioned at bottom-center, four toast levels used consistently.

### 6.2 Progress Indicators -- Score: 52/100
- No progress bars for long operations (YAML import, bulk actions).

### 6.3 Real-time Updates -- Score: 85/100
- WebSocket integration well-implemented with auto-reconnection, per-node message queuing, 7 event types, and automatic TanStack Query cache invalidation.
- **LOW:** No user-visible WebSocket connection status indicator.

---

## 7. Consistency -- Score: 74/100

**CRITICAL: Two competing component systems exist side by side.** Hand-built components in `components/` (lacking accessibility) alongside shadcn/ui components in `components/ui/` and `ui/`.

**MODERATE: Inconsistent error handling patterns.** Canvas deletion uses proper Dialog; group/role/secret deletion uses `window.confirm()`.

**MODERATE: Mixed loading state patterns.** `animate-spin`, `<Text>Loading...</Text>`, `<p>Loading user...</p>`, `<LoadingButton>` all used for similar purposes.

**LOW: Mixed terminology** — "Canvas", "Workflow", and "Bundle" used for related concepts.

---

## Prioritized Recommendations

### Priority 1 -- High Impact

| # | Finding | Impact | Effort |
|---|---------|--------|--------|
| 1 | Replace `window.confirm()` with AlertDialog across 12+ locations | High | Medium |
| 2 | Add route-level Error Boundaries for workflow editor and settings | High | Low |
| 3 | Add ARIA attributes to custom Dialog (`role="dialog"`, `aria-modal`, Escape) | High | Low |
| 4 | Implement skeleton loading for home page and settings pages | High | Medium |

### Priority 2 -- Moderate Impact

| # | Finding | Impact | Effort |
|---|---------|--------|--------|
| 5 | Add undo/redo for canvas editing | High | High |
| 6 | Show 404 page instead of silent redirect for unknown routes | Medium | Low |
| 7 | Fix Switch `focus:outline-none` to `focus-visible:ring-2` | Medium | Low |
| 8 | Add mobile viewport warning for canvas editor | Medium | Low |
| 9 | Consolidate component systems — migrate to shadcn equivalents | Medium | High |
| 10 | Add WebSocket connection status indicator | Medium | Low |

### Priority 3 -- Lower Impact

| # | Finding | Impact | Effort |
|---|---------|--------|--------|
| 11 | Use `useNavigate` instead of `window.location.href` in error pages | Low | Low |
| 12 | Add optimistic updates for CRUD mutations | Low | Medium |
| 13 | Unify loading state components | Low | Medium |
| 14 | Align terminology: Canvas vs Workflow vs Bundle | Low | Low |
| 15 | Add error recovery in AccountContext | Low | Low |

---

## Score Summary

| Dimension | Score | Grade |
|-----------|-------|-------|
| Error Handling UX | 72/100 | C+ |
| Loading States | 62/100 | D+ |
| User Journey Quality | 69/100 | D+ |
| Accessibility | 68/100 | D+ |
| Responsive Design | 55/100 | F |
| Feedback & Communication | 73/100 | C |
| Consistency | 74/100 | C |
| **Overall QX Score** | **71/100** | **C+** |

---
*Generated by AQE v3 QX Partner Agent*
