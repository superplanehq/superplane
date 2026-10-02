import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "bun:test";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import { PlanningReviewEditor } from "./PlanningReviewEditor";
import type { PlanningReviewDraft, PlanningReviewStep } from "./planningReviewMockup";

const DEFAULT_PROMPT = "Plan the task from the factory default.";

function refineDraft(prompt: string): PlanningReviewDraft {
  return {
    title: "Refine Task",
    components: [
      {
        id: "refine-task",
        title: "Refine Task",
        description: "",
        expanded: true,
        configuration: {
          steps: [
            { name: "Clone repository", type: "bash", command: "git clone" },
            { name: "Refine Task", type: "prompt", prompt, workingDirectory: "repo" },
          ],
        },
        concurrency: { max: "1", key: "" },
      },
    ],
  };
}

function renderEditor(
  props: {
    onSave?: (draft: PlanningReviewDraft) => void | Promise<void>;
    onRestoreDefaultPrompt?: () => Promise<PlanningReviewStep | null>;
  } = {},
) {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <PlanningReviewEditor
              initialDraft={refineDraft("Custom prompt")}
              onSave={props.onSave}
              showAutomationNote={false}
              showCancel={false}
              onRestoreDefaultPrompt={props.onRestoreDefaultPrompt}
            />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("PlanningReviewEditor restore default prompt", () => {
  it("hides Restore default prompt when no callback is given", () => {
    renderEditor();

    expect(screen.queryByTestId("planning-review-restore-default-prompt")).not.toBeInTheDocument();
  });

  it("replaces the prompt after confirm and saves only when Save Agent is clicked", async () => {
    const user = userEvent.setup();
    const onSave = vi.fn();
    const onRestoreDefaultPrompt = vi.fn(async () => ({
      name: "Refine Task",
      type: "prompt" as const,
      prompt: DEFAULT_PROMPT,
      workingDirectory: "repo",
    }));
    renderEditor({ onSave, onRestoreDefaultPrompt });

    await user.click(screen.getByTestId("planning-review-step-toggle-1"));
    expect(screen.getByTestId("planning-review-step-body-1")).toHaveValue("Custom prompt");

    await user.click(screen.getByTestId("planning-review-restore-default-prompt"));
    expect(screen.getByRole("heading", { name: "Restore the default prompt?" })).toBeInTheDocument();
    expect(onRestoreDefaultPrompt).not.toHaveBeenCalled();
    expect(onSave).not.toHaveBeenCalled();

    await user.click(screen.getByTestId("planning-review-restore-default-prompt-confirm"));

    await waitFor(() => {
      expect(screen.getByTestId("planning-review-step-body-1")).toHaveValue(DEFAULT_PROMPT);
    });
    expect(onRestoreDefaultPrompt).toHaveBeenCalledTimes(1);
    expect(onSave).not.toHaveBeenCalled();

    await user.click(screen.getByTestId("planning-review-save"));

    await waitFor(() => {
      expect(onSave).toHaveBeenCalledWith(
        expect.objectContaining({
          components: [
            expect.objectContaining({
              configuration: expect.objectContaining({
                steps: [
                  expect.objectContaining({ name: "Clone repository", command: "git clone" }),
                  expect.objectContaining({ name: "Refine Task", prompt: DEFAULT_PROMPT }),
                ],
              }),
            }),
          ],
        }),
      );
    });
  });
});
