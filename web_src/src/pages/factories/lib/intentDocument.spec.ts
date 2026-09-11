import { describe, expect, it } from "vitest";

import { composeIntentDocument, INTENT_DOCUMENT_TITLE, parseIntentDocument } from "./intentDocument";

describe("intentDocument", () => {
  it("uses the H1 as the title and Executive summary as the short view", () => {
    const document = parseIntentDocument(`# Clearer empty state

## Executive summary

### Goal

A person can add a payment method from the empty billing page.

The agent reads this as copy and an action on the current empty view. It does not read it as a new billing flow.

### Done when

- The empty view names the next action.
- The action opens add-payment-method.

### Out of scope

- The page after a card exists.

### Key architecture decisions

- Reuse the current empty view. Do not add a new page.

## Problem

The empty view only shows a title.

## Outcome

A person sees the next action on the billing page.
`);

    expect(document.title).toBe("Clearer empty state");
    expect(document.summary).toContain("A person can add a payment method from the empty billing page.");
    expect(document.summary).toContain("### Goal");
    expect(document.summary).toContain("### Done when");
    expect(document.summary).toContain("### Out of scope");
    expect(document.summary).toContain("### Key architecture decisions");
    expect(document.summary).not.toContain("## Problem");
    expect(document.plan).toContain("## Problem");
    expect(document.plan).toContain("## Outcome");
    expect(document.plan).not.toContain("## Executive summary");
  });

  it("falls back to the first section when Executive summary is missing", () => {
    const document = parseIntentDocument(`# Clearer empty state

## How I understand this

The billing page does not name the next action.

## What done looks like

The empty state tells the user how to add a card.
`);

    expect(document.title).toBe("Clearer empty state");
    expect(document.summary).toBe("The billing page does not name the next action.");
    expect(document.plan).toContain("## What done looks like");
    expect(document.plan).not.toContain("## How I understand this");
  });

  it("falls back to the first paragraph when there are no headings", () => {
    const document = parseIntentDocument("Ship a retry loop.\n\nCover the timeout path.");

    expect(document.title).toBe(INTENT_DOCUMENT_TITLE);
    expect(document.summary).toBe("Ship a retry loop.");
    expect(document.plan).toContain("Cover the timeout path.");
  });

  it("builds a document from the original request and a later plan", () => {
    const document = composeIntentDocument("Webhook delivery stops after one error.", "## Steps\n\nAdd a retry.");

    expect(document.title).toBe(INTENT_DOCUMENT_TITLE);
    expect(document.summary).toBe("Webhook delivery stops after one error.");
    expect(document.plan).toBe("## Steps\n\nAdd a retry.");
  });
});
