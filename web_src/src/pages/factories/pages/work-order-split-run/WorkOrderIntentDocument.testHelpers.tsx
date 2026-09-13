import type { ReactElement } from "react";
import { render } from "@testing-library/react";
import { vi } from "vitest";

import { TooltipProvider } from "@/ui/tooltip";

import { CONFIDENCE_CHECK_NAME, confidenceSuitabilitySummary } from "../../lib/confidenceScore";
import type { CreateWithAgentView } from "../createWithAgentTypes";
import type { IntentAnalysisChat } from "./WorkOrderIntentDocument";

export const INTENT_DOC = {
  title: "Show a clearer empty state",
  description: "Imported from GitHub: billing empty state is unclear.",
};

export const INTENT = {
  id: "art-intent",
  type: "TYPE_MARKDOWN" as const,
  data: {
    name: "intent.md",
    title: "intent.md",
    body: `# Clearer empty state

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

The empty state tells the user how to add a payment method.
`,
  },
};

export const HIGH_CONFIDENCE = {
  id: "check-confidence",
  name: CONFIDENCE_CHECK_NAME,
  score: 4,
  maxScore: 5,
  level: "positive" as const,
  summary: confidenceSuitabilitySummary("High"),
};

const EMPTY_ANALYSIS_VIEW: CreateWithAgentView = {
  repository: "acme/payments",
  machineStatus: "starting",
  canvasId: "",
  canvasRunId: "",
  executionId: "",
  messages: [],
  composer: "",
  created: [],
  right: { kind: "empty" },
  endConfirmOpen: false,
  selectableModelKey: "",
  refining: false,
};

export function analysisChat(
  extra: Partial<Omit<IntentAnalysisChat, "view">> & { view?: Partial<CreateWithAgentView> } = {},
): IntentAnalysisChat {
  const { view, ...rest } = extra;
  return {
    organizationId: "org-1",
    view: { ...EMPTY_ANALYSIS_VIEW, ...view },
    composer: "",
    canSend: true,
    onComposerChange: vi.fn(),
    onSend: vi.fn(),
    onSubmitSurvey: vi.fn(),
    ...rest,
  };
}

export function renderIntentDocument(ui: ReactElement) {
  return render(<TooltipProvider>{ui}</TooltipProvider>);
}

export let notifyIntentResize = () => {};

export class IntentDocumentResizeObserver {
  constructor(callback: ResizeObserverCallback) {
    notifyIntentResize = () => callback([], this as unknown as ResizeObserver);
  }

  observe() {}
  unobserve() {}
  disconnect() {}
}
