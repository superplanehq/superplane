# Token usage tracking and billing

> Status: Draft  
> Audience: Product and engineering  
> Slides: [PDF](token-usage-and-billing-slides.pdf) · [source](token-usage-and-billing-slides.md)

This draft is the project playbook for SuperPlane LLM usage tracking, hosted
credits, bring-your-own-key (BYOK), and Stripe. Build Phase 1 first. Later
phases stay design-only until the ledger numbers are trustworthy.

## Locked decisions

1. **Org wallet, workspace budgets.** The Stripe customer and credit balance
   live on the organization. Each workspace (factory) can later receive a spend
   budget from that wallet. Reports roll up both ways from day one.
2. **Charge hosted spend only.** Welcome credit and later Stripe top-ups pay
   for SuperPlane-held provider keys. BYOK is tracked (tokens and estimated
   dollars) and is not billed.
3. **Providers: Anthropic, OpenAI, OpenRouter.** Cursor and Bedrock are out of
   this program. Existing Cursor components stay as they are.

## Goal

New organizations receive a welcome credit. They pick models from an admin
allowlist, connect a repository, and spend that credit on factory agents.

When the credit is gone, they set up billing and continue on SuperPlane-hosted
models. They can also connect their own Anthropic, OpenAI, or OpenRouter keys.
BYOK usage is reported. It does not debit SuperPlane credit.

## What exists today

Do not reinvent these pieces:

- Factory UI already shows spend when `total_tokens` / `cost_cents` are
  non-zero. Serialization already sums executions onto the work order.
  **Nothing writes those columns.**
- Org LLM integrations already store customer API keys. OpenAI and Anthropic
  clients exist. There is no OpenRouter client and no SuperPlane-hosted key
  path for factory agents.
- `protos/usage.proto` and `/settings/billing` are **SaaS plan limits**
  (canvases, events, canvas-agent tokens, runner minutes). Keep that service.
  Do not use it as the spend ledger.
- The factory PRD defines tracked cost as model tokens plus execution compute.
  It excludes third-party charges and human labor.

## Product rules

| Topic | Rule |
| --- | --- |
| Billing unit | Ledger stores tokens by type and USD cents. Money is the source of truth. |
| Welcome grant | Once per organization, not per user. |
| Hosted catalog | Admin picks one hosted provider and an allowlist of models. |
| BYOK catalog | User connects keys. SuperPlane lists models. User picks the allowlist. |
| BYOK cost | Store estimated or provider-reported dollars. Mark `funding_source=byok`. |
| Compute | Same ledger, `usage_kind=compute`, later. Phase 1 is model usage only. |
| Self-hosted | Tracking ships. Welcome credit and Stripe are cloud-only. |
| Prompts | Do not store prompt or completion text on usage rows. |
| Failed runs | Record tokens already consumed. Pass or fail does not erase spend. |

## Domain model

```
Organization (wallet, Stripe customer, model allowlist, BYOK keys)
  └── Workspace / Factory (optional budget, reports)
        └── Work order → line step execution
              └── llm_usage_events (append-only)
```

- **Org** owns the Stripe customer, credit balance, hosted provider config,
  default model allowlist, and BYOK keys.
- **Workspace** owns optional spend caps and reports. Agents may further subset
  the org allowlist.
- Do not create a second Stripe customer per workspace.

**Usage event** — one append-only row per LLM call (or per terminal usage
snapshot for long-running Anthropic managed agents):

- Scope: organization, factory, work order, line, dispatch, execution, canvas
  run, node execution
- What: `provider` (`anthropic` \| `openai` \| `openrouter`), `model`,
  `usage_kind` (`model` \| `compute`), `funding_source` (`hosted` \| `byok`)
- Amounts: input, output, cache read/write, reasoning, total tokens,
  `cost_cents`, currency, `price_book_version`
- Safety: unique `idempotency_key`, `occurred_at`, no prompt payload

Execution `total_tokens` / `cost_cents` stay as **cached rollups** for the
existing work-order API and UI. Reports that need “by model” read the ledger.

## Phases

### Phase 1 — Tracking and reporting (build now)

No Stripe, no wallet, no new provider product.

1. Add `llm_usage_events` via `make db.migration.create`.
2. Add a versioned price book (`provider + model + token_type` → cents per
   million tokens).
3. Record usage at the call site (`RecordUsage`). Do not scrape canvas events.
4. Start with `claude.textPrompt`, `openai.textPrompt`, and
   `perplexity.runAgent` on factory-linked runs. Perplexity is instrumented
   because it already emits usage. It is not an in-scope billing provider.
5. Roll up into `factory_work_order_executions.total_tokens` / `cost_cents`.
6. Add org, workspace, work-order, and per-model report RPCs plus a thin UI.
   Do not overload the plan-limits Usage page (`/settings/billing`).

**Done when:** a factory work order that runs a text-prompt node shows tokens
and estimated USD, and an org or workspace report can break spend down by
model.

### Phase 2 — Provider clients

Shared client surface: list models, complete or stream, normalized usage.
Wrap existing integrations. Add OpenRouter. Fill Anthropic managed-agent usage
holes. Out of scope: Cursor, Bedrock.

### Phase 3 — Hosted catalog and welcome credit

Admin picks one hosted provider and an allowlist. Grant credit on org create.
Hosted events debit the wallet. Soft warning at a threshold. Hard stop waits
for Phase 5.

### Phase 4 — BYOK model pools

Reuse org integrations. Add OpenRouter. User builds the selected-model list.
Same ledger, `funding_source=byok`. Wallet is untouched.

### Phase 5 — Stripe

One Stripe customer per org. Prepaid top-up for v1. Block new hosted calls
when hosted balance is zero and no payment method exists. Cloud-only.

### Phase 6 — Workspace budgets

Per-factory cap against the org wallet. Soft then hard. Add a reservation for
in-flight agents. Optional `usage_kind=compute` later.

## Assumptions

These are treated as true unless product changes them:

- Credit is granted once per organization, not per user or per workspace.
- USD cents are the billing source of truth. Token counts are attributes.
- Existing Usage gRPC stays for SaaS plan limits. This program does not
  replace it.
- Factory execution cost columns are caches. The ledger is the source of
  truth.
- Record usage at the provider call site. Do not parse admin “Get Usage”
  APIs or scrape node payloads as the ledger.
- Current org integrations are BYOK until hosted keys exist.
- Self-hosted deployments get tracking. They do not get welcome credit or
  Stripe.
- Stripe v1 is prepaid top-up, not metered invoices.
- Canvas sidebar agent usage stays on the external Usage service in Phase 1.
- Cursor and Bedrock stay out of the price book, catalog, and picker.

## Open questions

Decide these while Phase 1 is in progress or before Phase 3:

1. **Welcome credit amount.** How much USD (or displayed tokens) does a new
   org receive?
2. **Display unit.** Does the UI say “tokens”, USD, or both for remaining
   credit?
3. **Default hosted provider.** OpenRouter is the fastest multi-model option.
   Confirm Anthropic, OpenAI, or OpenRouter for the welcome pool.
4. **Markup.** Pass through provider cost, or apply a SuperPlane markup?
   If markup comes later, store provider cost and billed cost as separate
   fields.
5. **Hard stop timing.** Wait for Stripe (Phase 5), or stop hosted calls when
   credit is zero?
6. **Workspace model subset.** Can a workspace narrow the org allowlist in
   Phase 4, or only in Phase 6?
7. **Ledger retention.** How long do we keep usage events? This is separate
   from `usage_retention_window_days` on the plan-limits service.
8. **Rename `/settings/billing`.** That page is plan limits. When do we split
   or rename it so users do not confuse it with LLM credit?
9. **Compute in the same reports.** When do runner minutes join the ledger?
10. **Tax and invoicing entity.** Legal work for Stripe. Not schema work now.

## Non-goals (this program)

- Cursor and Bedrock as hosted or BYOK providers
- Replacing the external Usage gRPC plan-limits service
- Billing BYOK spend
- A Stripe customer per workspace
- Storing prompts on usage rows
- Attributing human labor or third-party SaaS charges

## First implementation slice

1. Keep this PRD as the source of truth.
2. Implement **Phase 1 only**: ledger, price book, record from factory-linked
   components that already return usage, roll up execution columns, report API,
   thin UI.
3. Leave Phases 2–6 as design until Phase 1 numbers look correct.
