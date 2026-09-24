import type { ReactElement } from "react";
import { render } from "@testing-library/react";
import { vi } from "vitest";

import type { UploadedWorkOrderFile } from "@/hooks/useWorkOrderFileUpload";
import { TooltipProvider } from "@/ui/tooltip";

import { CLARITY_CHECK_NAME, CONFIDENCE_CHECK_NAME, confidenceSuitabilitySummary } from "../../lib/confidenceScore";
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

A person can add a payment method from the empty billing page.

## Problem

The empty view only shows a title. It does not name the next action.

## Proposed outcome

The empty view names the next action. The action opens add-payment-method.

## Constraints

Do not build a new billing flow. Do not change the page after a card exists.

## Scope

Copy and an action on the current empty view.

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

export const HIGH_CLARITY = {
  id: "check-clarity",
  name: CLARITY_CHECK_NAME,
  score: 4,
  maxScore: 5,
  level: "positive" as const,
  summary: "One decision is still open. Confirm the empty state copy.",
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

export function uploadedComposerImage(id: string, filename: string): UploadedWorkOrderFile {
  return {
    id,
    filename,
    contentType: "image/png",
    ref: `sp-file://${id}`,
    previewUrl: `https://cdn.example.com/${filename}`,
    isImage: true,
  };
}

export function uploadedComposerFile(id: string, filename: string): UploadedWorkOrderFile {
  return {
    id,
    filename,
    contentType: "text/plain",
    ref: `sp-file://${id}`,
    previewUrl: `https://cdn.example.com/${filename}`,
    isImage: false,
  };
}

export function composerPng(name: string) {
  return new File(["img"], name, { type: "image/png" });
}

export function composerTextFile(name: string) {
  return new File(["notes"], name, { type: "text/plain" });
}

export const WAITING_COMPOSER_VIEW = { machineStatus: "waiting" as const };
