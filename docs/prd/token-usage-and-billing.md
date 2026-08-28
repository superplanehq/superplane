# Token usage tracking and billing

> Status: Draft  
> Audience: Product and engineering  
> Slides: [PDF](token-usage-and-billing-slides.pdf) · [source](token-usage-and-billing-slides.md)

This draft is the project playbook for SuperPlane LLM usage tracking, hosted
credits, bring-your-own-key (BYOK), and Polar prepaid billing. Phases 1–3 are
shipped. This document describes Phases 4–6 as built: BYOK model pools, Polar
sandbox checkout, and factory hosted spend limits.

## Locked decisions

1. **Org wallet, workspace budgets.** The Polar customer and credit balance
   live on the organization. Each workspace (factory) can receive a hosted
   spend limit from that wallet. Reports roll up both ways from day one.
2. **Charge hosted spend only.** Welcome credit and Polar prepaid packs pay
   for SuperPlane-held provider keys. BYOK is tracked (tokens and estimated
   dollars) and is not billed.
3. **Providers: Anthropic, OpenAI, OpenRouter.** Cursor and Bedrock are out of
   this program. Existing Cursor components stay as they are.

## Goal

New organizations receive a welcome credit. They pick models from an admin
allowlist, connect a repository, and spend that credit on factory agents.

When the credit is gone, owners add hosted credit and continue on SuperPlane-hosted
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
| BYOK catalog | User connects keys. SuperPlane lists models. The list guides the picker. It does not gate a run. |
| BYOK cost | Store estimated or provider-reported dollars. Mark `funding_source=byok`. |
| Compute | Same ledger, `usage_kind=compute`, later. Phase 1 is model usage only. |
| Self-hosted | Tracking ships. Welcome credit and Polar checkout are cloud-only. |
| Prompts | Do not store prompt or completion text on usage rows. |
| Failed runs | Record tokens already consumed. Pass or fail does not erase spend. |

## Domain model

```
Organization (wallet, Polar customer, model allowlist, BYOK keys)
  └── Workspace / Factory (optional hosted spend limit, reports)
        └── Work order → line step execution
              └── llm_usage_events (append-only)
```

- **Org** owns the Polar customer, credit balance, hosted provider config,
  default model allowlist, and BYOK keys.
- **Workspace** owns optional hosted spend limits and reports. Agents may further subset
  the org allowlist.
- Do not create a second Polar customer per workspace.

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

### Phase 1 — Tracking and reporting (shipped)

Ledger, price book, `RecordUsage`, execution rollups, org and workspace reports.
Do not overload the plan-limits Usage page (`/settings/billing`).

### Phase 2 — Provider clients (shipped)

Shared client surface for Anthropic, OpenAI, and OpenRouter. Out of scope:
Cursor, Bedrock.

### Phase 3 — Hosted catalog and welcome credit (shipped)

Admin picks hosted providers and allowlists. Grant credit on org create.
Hosted events debit the wallet. Soft warning at a threshold. Hard stop when
hosted remaining credit is empty.

### Phase 4 — BYOK model pools

Reuse org integrations. The organization selects the BYOK model list per
provider. Factory Settings → Models may subset the org BYOK list and the
installation hosted allowlist. An empty factory list inherits the parent list.
The agent picker uses the resolved list for hosted and BYOK credentials.
`funding_source=byok` is tracked and does not debit the wallet.

The BYOK list guides the picker only. It does not stop a run. A BYOK run
spends the key of the organization, not SuperPlane credit, so the list gives
SuperPlane no cost to protect. A gate there only stops the organization from
using the key it connected. It also rejects an agent CLI alias such as `opus`,
because the list holds full model ids. Only `PrepareHostedRun` keeps a
selected-model gate, because hosted spend debits the wallet.

### Phase 5 — Polar prepaid checkout

Polar is the payment bookkeeper. SuperPlane keeps the wallet, markup, and
hosted run gate. Polar does not proxy inference and does not ingest usage in
this phase.

- One Polar customer per organization (`external_id` = org UUID).
- Prepaid one-time credit packs ($25 / $100 / $500) discovered by product
  metadata `superplane_credit_pack=true`.
- `order.paid` inserts an `organization_llm_credit_grants` row of kind `polar`.
  Wallet credit equals pack face value. Tax is extra on the Polar invoice.
- Org LLM spend shows **Add hosted credit** and **Manage invoices** when Polar
  is configured. Hide those actions when Polar env is empty (self-hosted).
- Do not put this UI on `/settings/billing` (SaaS plan limits).
- Polar usage meters and PAYG invoices are deferred. SuperPlane remaining
  credit stays the source of truth.

### Phase 6 — Workspace budgets

Per-factory hosted spend limit against the org wallet. Null means no factory
cap. Zero means hosted runs cannot start in that factory. Remaining factory
budget is cap minus hosted billed spend for that factory. Effective remaining
is the minimum of factory remaining and org remaining. Soft warning uses the
installation threshold. Hard stop in `PrepareHostedRun` when effective
remaining is 0. BYOK ignores the cap. Org-level one-in-flight hosted hold
stays. Do not invent a second wallet.

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
  Polar checkout.
- Polar v1 is prepaid top-up, not metered invoices. Polar usage meters and
  PAYG invoices stay out of this program until prepaid checkout is proven.
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
5. **Hard stop timing.** Hosted calls stop when remaining credit is zero.
   Polar checkout is how owners add credit.
6. **Workspace model subset.** Factory Settings → Models can subset the org
   BYOK list and the installation hosted allowlist in Phase 4.
7. **Ledger retention.** How long do we keep usage events? This is separate
   from `usage_retention_window_days` on the plan-limits service.
8. **Rename `/settings/billing`.** That page is plan limits. LLM credit lives
   on LLM spend. Keep the nav label **Usage** for plan limits.
9. **Compute in the same reports.** When do runner minutes join the ledger?
10. **Tax and invoicing entity.** Polar is the merchant of record. Polar
    invoices the customer and collects tax. SuperPlane is not the tax filer.

## Non-goals (this program)

- Cursor and Bedrock as hosted or BYOK providers
- Replacing the external Usage gRPC plan-limits service
- Billing BYOK spend
- A Polar customer per workspace
- Polar usage meters, Credits benefits, or PAYG invoices in v1
- Storing prompts on usage rows
- Attributing human labor or third-party SaaS charges

## First implementation slice

1. Keep this PRD as the source of truth.
2. Phases 1–3 are shipped.
3. Implement Phases 4–6: BYOK model pools, Polar prepaid checkout, factory
   hosted spend limits.
4. Defer Polar meters until prepaid checkout works. If meters are added later,
   ingest billed cents, not tokens.
