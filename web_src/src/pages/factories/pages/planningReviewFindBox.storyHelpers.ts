import type { SuperplaneComponentsNode } from "@/api-client";
import { expect, userEvent, waitFor, within } from "storybook/test";

import { PLANNING_REVIEW_DRAFT, type PlanningReviewDraft, type PlanningReviewStep } from "./planningReviewMockup";

export const LONG_BASH_COMMAND = Array.from({ length: 16 }, (_, index) => `echo line-${index + 1}`).join("\n");

export function draftWithBashCommand(command: string): PlanningReviewDraft {
  const component = PLANNING_REVIEW_DRAFT.components[0];
  const steps = component?.configuration.steps;
  const nextSteps = Array.isArray(steps)
    ? (steps as PlanningReviewStep[]).map((step, index) => (index === 0 ? { ...step, command } : step))
    : steps;

  return {
    ...PLANNING_REVIEW_DRAFT,
    components: PLANNING_REVIEW_DRAFT.components.map((entry, index) =>
      index === 0 ? { ...entry, configuration: { ...entry.configuration, steps: nextSteps } } : entry,
    ),
  };
}

export function agentNodeWithCommand(command: string): SuperplaneComponentsNode {
  const draft = draftWithBashCommand(command);
  const component = draft.components[0];
  return {
    id: component?.id ?? "implementation-agent",
    name: component?.title ?? "Implement",
    type: "TYPE_ACTION",
    component: "runnerClaudeCode",
    configuration: component?.configuration ?? {},
    concurrency: { max: 5 },
  };
}

/** Rest the pointer on Close, then click the element under the pointer. */
export async function playFindBoxCloses(
  { canvasElement }: { canvasElement: HTMLElement },
  dismissed: { calls: number },
) {
  dismissed.calls = 0;
  const editor = await openBashStep(canvasElement);
  const surface = monacoSurface(editor);
  await openFind(surface, editor);
  await clickCloseUnderPointer(editor);
  await dismissFindWithEscape(surface, editor);
  const toggle = within(canvasElement).getByTestId("planning-review-step-toggle-0");
  expect(dismissed.calls).toBe(0);
  expect(editor).toBeVisible();
  expect(toggle).toHaveAttribute("aria-expanded", "true");
}

async function openBashStep(canvasElement: HTMLElement) {
  const canvas = within(canvasElement);
  const toggle = await canvas.findByTestId("planning-review-step-toggle-0");
  if (toggle.getAttribute("aria-expanded") !== "true") {
    await userEvent.click(toggle);
  }
  const editor = await canvas.findByTestId("planning-review-step-body-0-editor");
  await waitFor(() => {
    expect(editor.querySelector(".monaco-editor")).toBeTruthy();
  });
  return editor;
}

function monacoSurface(editor: HTMLElement) {
  const surface = editor.querySelector(".monaco-editor");
  if (!(surface instanceof HTMLElement)) {
    throw new Error("Monaco editor did not render");
  }
  return surface;
}

async function openFind(surface: HTMLElement, editor: HTMLElement) {
  await userEvent.click(surface);
  await userEvent.keyboard("{Control>}f{/Control}");
  await waitFor(() => {
    expect(findIsOpen(editor)).toBe(true);
  });
}

async function clickCloseUnderPointer(editor: HTMLElement) {
  const close = editor.querySelector(".codicon-widget-close");
  if (!(close instanceof HTMLElement)) {
    throw new Error("Find close button did not render");
  }
  close.dispatchEvent(new MouseEvent("mouseover", { bubbles: true }));
  await waitFor(() => {
    expect(closeTooltipStaysOnOneLine(editor)).toBe(true);
  });
  const rect = close.getBoundingClientRect();
  const target = document.elementFromPoint(rect.left + rect.width / 2, rect.top + rect.height / 2);
  expect(close.contains(target)).toBe(true);
  target?.dispatchEvent(new MouseEvent("click", { bubbles: true }));
  await waitFor(() => {
    expect(findIsOpen(editor)).toBe(false);
  });
}

async function dismissFindWithEscape(surface: HTMLElement, editor: HTMLElement) {
  await openFind(surface, editor);
  await userEvent.keyboard("{Escape}");
  await waitFor(() => {
    expect(findIsOpen(editor)).toBe(false);
  });
}

function findIsOpen(editor: HTMLElement) {
  return editor.querySelector(".find-widget")?.classList.contains("visible") ?? false;
}

function closeTooltipStaysOnOneLine(editor: HTMLElement) {
  const hover = editor.querySelector(".workbench-hover");
  const contents = hover?.querySelector(".hover-contents");
  if (!hover || !contents) {
    return false;
  }
  return hover.getBoundingClientRect().height < 32 && contents.scrollWidth <= contents.clientWidth + 1;
}
