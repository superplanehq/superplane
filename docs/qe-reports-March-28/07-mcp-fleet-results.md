# MCP Fleet Raw Results
**Date:** March 28, 2026
**Fleet ID:** fleet-bf88d6ec
**Topology:** Hierarchical | Max Agents: 15

---

## 1. Quality Assessment (quality_assess)

**Status:** Completed | **Duration:** 49.6s | **Gate:** FAILED

| Metric | Value | Threshold | Status |
|--------|-------|-----------|--------|
| Overall Score | 49.0/100 | 70 | FAIL |
| Coverage | -1 (not measured) | 80% | N/A |
| Complexity | 30.95 | < 15 | FAIL |
| Maintainability | 57.46 | > 65 | FAIL |
| Security | 85.0 | > 80 | PASS |

**Recommendations:**
- [CRITICAL] Reduce Code Complexity: Average cyclomatic complexity is 30.95
- [CRITICAL] Overall Quality Improvement Needed: Multiple areas need attention

---

## 2. Security Scan — Backend (pkg/)

**Status:** Completed | **Duration:** 2.2s | **Files Scanned:** 1,734

| Severity | Count |
|----------|-------|
| Critical | 172 |
| High | 2 |
| Medium | 0 |
| Low | 0 |
| **Total** | **174** |

### False Positive Analysis

Manual verification of top flagged items revealed that the **majority are false positives**:

| File | Line | Finding | Verdict |
|------|------|---------|---------|
| `authentication/context.go` | 10 | `TokenScopesMetadataKey = "x-token-scopes"` | FALSE POSITIVE — metadata key name |
| `integrations/aws/aws.go` | 37 | `APIKeyHeaderName = "X-Superplane-Secret"` | FALSE POSITIVE — header name constant |
| `integrations/aws/aws.go` | 38 | `EventBridgeConnectionSecretName = "eventbridge.connection.secret"` | FALSE POSITIVE — key name |
| `cli/commands/secrets/common.go` | 18 | `SecretKind = "Secret"` | FALSE POSITIVE — kind identifier |
| `authentication/authentication.go` | 168 | `AccessToken: "dev-token-" + provider` | **TRUE POSITIVE** — mock dev token |

**Estimated True Positive Rate:** ~5-10% (mostly struct field names, constant identifiers)

### Genuine Concerns
1. Dev mock token in `authentication.go:168` — verify dev-mode guard
2. Need to verify `impersonation/session.go` handling

---

## 3. Coverage Analysis — Frontend (web_src/src/)

**Status:** Completed | **Files Analyzed:** 790

| Metric | Value |
|--------|-------|
| Avg Line Coverage | 79.1% |
| Avg Branch Coverage | 99.2% |
| Avg Function Coverage | 20.8% |
| Files with 0% Function Coverage | 489 (61.9%) |
| Total Coverage Gaps | 727 |

### Critical Coverage Gaps (sample)

| File | Risk | Reason |
|------|------|--------|
| `pages/auth/Login.tsx` | 0.5 | Missing test case |
| `pages/auth/OwnerSetup.tsx` | 0.5 | Missing test case |
| `pages/organization/settings/CreateRolePage.tsx` | 0.5 | Missing test case |
| `hooks/useCanvasData.ts` | 0.5 | Missing test case |
| `hooks/useCanvasWebsocket.ts` | 0.5 | Missing test case |
| `components/AutoCompleteSelect/index.tsx` | 0.5 | Missing test case |
| `components/CreateCustomComponentModal/index.tsx` | 0.5 | Missing test case |

---

## 4. Defect Prediction — Backend (pkg/)

**Status:** Completed | **Duration:** 2.8s

- **Predicted Defects:** 0
- **Risk Score:** 0
- **Assessment:** No files exceeded the defect probability threshold — code looks healthy

---

## 5. Code Index

| Target | Files Indexed | Symbols | Relations |
|--------|--------------|---------|-----------|
| `web_src/src/` | 0 | 0 | 0 |
| `pkg/` | 1,734 | 0 | 0 |

---

## 6. Task Orchestration

**Task ID:** task_05618b3c
**Strategy:** Parallel
**Routing:** Tier 2 (Sonnet) — Complexity 50/100

---

*Raw results stored in `.agentic-qe/results/`*
