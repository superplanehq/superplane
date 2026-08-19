---
marp: true
paginate: true
title: SuperPlane — Token usage and billing
description: Playbook slides. theme-factories tokens. Export to PDF with Marp or print from HTML after agent mode.
---

<style>
@import url("https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600&display=swap");

section {
  --radius: 0.5rem;
  --background: #ffffff;
  --foreground: #26251e;
  --muted: #f7f7f7;
  --muted-foreground: #737373;
  --border: #e5e5e5;
  --status-running-fg: #1d4ed8;
  --status-running-bg: #eff6ff;
  --status-running-border: #bfdbfe;
  --status-waiting-fg: #b45309;
  --status-waiting-bg: #fffbeb;
  --status-waiting-border: #fde68a;
  --status-draft-fg: #52525b;
  --status-draft-bg: #f4f4f5;
  --status-draft-border: #e4e4e7;

  font-family: "Inter", ui-sans-serif, system-ui, sans-serif;
  font-feature-settings: "cv02", "cv03", "cv04", "cv11";
  font-size: 28px;
  line-height: 1.4;
  color: var(--foreground);
  background: var(--background);
  padding: 56px 64px 48px;
}

h1, h2 {
  font-weight: 600;
  letter-spacing: -0.02em;
  color: var(--foreground);
}

h1 { font-size: 44px; }
h2 { font-size: 28px; }

p, li { font-size: 18px; line-height: 1.45; }

.kicker {
  color: var(--muted-foreground);
  font-size: 14px;
  font-weight: 500;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  margin-bottom: 8px;
}

.lede {
  color: var(--muted-foreground);
  font-size: 18px;
  max-width: 42rem;
}

.pills { display: flex; gap: 8px; margin-top: 24px; }

.pill {
  display: inline-flex;
  align-items: center;
  height: 28px;
  padding: 0 10px;
  border-radius: 6px;
  border: 1px solid var(--status-draft-border);
  background: var(--status-draft-bg);
  color: var(--status-draft-fg);
  font-size: 13px;
  font-weight: 500;
  letter-spacing: 0.04em;
}

.pill.now {
  background: var(--status-running-bg);
  border-color: var(--status-running-border);
  color: var(--status-running-fg);
}

.pill.later {
  background: var(--status-waiting-bg);
  border-color: var(--status-waiting-border);
  color: var(--status-waiting-fg);
}

.columns { display: grid; grid-template-columns: 1fr 1fr 1fr; gap: 16px; margin-top: 28px; }
.columns-2 { display: grid; grid-template-columns: 1fr 1fr; gap: 20px 36px; margin-top: 24px; }

.card {
  background: var(--muted);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 16px 18px;
}

.card h3 { font-size: 18px; margin: 0 0 8px; }
.card p, .card li { font-size: 15px; color: var(--muted-foreground); margin: 0; }

footer {
  color: var(--muted-foreground);
  font-size: 12px;
  letter-spacing: 0.04em;
}
</style>

<!-- _class: title -->

<p class="kicker">SuperPlane</p>

# Token usage and billing

<p class="lede">Track factory LLM spend. Give new organizations a welcome credit. Charge SuperPlane-hosted models after that credit is gone.</p>

<div class="pills">
<span class="pill now">Phase 1 first</span>
<span class="pill">Anthropic · OpenAI · OpenRouter</span>
<span class="pill">Draft playbook</span>
</div>

---

<p class="kicker">Locked</p>

# Three decisions

<div class="columns">
<div class="card">
<h3>Org wallet</h3>
<p>One Stripe customer and one credit balance per organization. Workspaces get optional budgets later.</p>
</div>
<div class="card">
<h3>Hosted spend only</h3>
<p>SuperPlane bills its own keys. BYOK is reported. BYOK does not debit credit.</p>
</div>
<div class="card">
<h3>Three providers</h3>
<p>Anthropic, OpenAI, and OpenRouter. Cursor and Bedrock are out of this program.</p>
</div>
</div>

Reports roll up from work order to workspace to org from day one.

---

<p class="kicker">Build now</p>

# Phase 1 — Tracking and reporting

<p class="lede">No Stripe. No wallet. No new provider product. Write the ledger first.</p>

<div class="columns-2">
<div class="card">
<h3>Do</h3>
<p>Add usage events and a versioned price book. Record usage at the call site. Roll up into work-order cost columns. Add org, workspace, and per-model reports.</p>
</div>
<div class="card">
<h3>Do not</h3>
<p>Reuse plan-limits Usage as the ledger. Scrape node payloads. Overload /settings/billing. Instrument Cursor, Bedrock, or canvas-sidebar chat.</p>
</div>
</div>

Done when a text-prompt work order shows tokens and estimated USD.

---

<p class="kicker">Later</p>

# Phases 2 to 6

- <span class="pill later">2</span> Shared clients and model lists for the three providers.
- <span class="pill later">3</span> Hosted catalog and welcome credit on org create.
- <span class="pill later">4</span> BYOK model picker. Same ledger. Wallet untouched.
- <span class="pill later">5</span> Stripe prepaid top-up. Hard stop when hosted balance is zero.
- <span class="pill later">6</span> Workspace spend budgets and in-flight reservations.

Leave 2–6 as design until Phase 1 numbers look correct.

---

<p class="kicker">Working set</p>

# Assumptions

<div class="columns-2">

- **Grant once per org.** Not per user or workspace.
- **USD cents** are the billing source of truth.
- **Plan-limits Usage** stays a separate service.
- **Execution cost columns** are caches. The ledger is source of truth.
- **Record at the call site.** Do not scrape payloads.

- **Current integrations are BYOK** until hosted keys exist.
- **Self-hosted gets tracking** only. No credit. No Stripe.
- **Stripe v1 is prepaid top-up,** not metered invoices.
- **Canvas sidebar chat** stays on the external Usage service.
- **Cursor and Bedrock** stay out of catalog and picker.

</div>

---

<p class="kicker">Still open</p>

# Questions

<div class="columns-2">

1. **Welcome amount.** How much USD does a new org receive?
2. **Display unit.** Tokens, USD, or both for remaining credit?
3. **Default hosted provider.** OpenRouter, Anthropic, or OpenAI?
4. **Markup.** Pass through cost, or add a SuperPlane margin?
5. **Hard stop.** Wait for Stripe, or stop when credit is zero?

6. **Workspace model subset.** In Phase 4, or only with budgets?
7. **Ledger retention.** How long do we keep usage events?
8. **Rename /settings/billing.** That page is plan limits.
9. **Compute.** When do runner minutes join the same reports?
10. **Tax and invoicing entity.** Legal work for Stripe. Not schema work now.

</div>

Decide 1–5 before Phase 3. Decide 6–10 before Phase 5–6.

---

<p class="kicker">Next</p>

# First slice

Implement Phase 1 only. Keep the PRD as the source of truth.

<div class="columns">
<div class="card">
<h3>Ledger</h3>
<p>llm_usage_events, price book, RecordUsage, tests.</p>
</div>
<div class="card">
<h3>Instrument</h3>
<p>Factory-linked Claude, OpenAI, and Perplexity nodes that already return usage.</p>
</div>
<div class="card">
<h3>Report</h3>
<p>Org, workspace, work order, and per-model spend. Thin UI.</p>
</div>
</div>
