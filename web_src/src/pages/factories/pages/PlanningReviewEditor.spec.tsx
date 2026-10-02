import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "bun:test";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import { PLANNING_SETTINGS_COPY } from "./planningSettingsCopy";
import { PlanningReviewEditor } from "./PlanningReviewEditor";
import type { PlanningReviewDraft, PlanningReviewStep } from "./planningReviewMockup";

vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
}));

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

function defaultStep(prompt: string): PlanningReviewStep {
  return {
    name: "Refine Task",
    type: "prompt",
    prompt,
    workingDirectory: "repo",
  };
}

function deferredStep() {
  let resolve: (step: PlanningReviewStep | null) => void = () => undefined;
  const promise = new Promise<PlanningReviewStep | null>((settle) => {
    resolve = settle;
  });
  return { promise, resolve };
}
describe("PlanningReviewEditor restore default prompt", () => {
  it("hides Restore default prompt when no callback is given", () => {
    renderEditor();

    expect(screen.queryByTestId("planning-review-restore-default-prompt")).not.toBeInTheDocument();
  });

  it("hides Restore default prompt when the prompt already matches", async () => {
    const onRestoreDefaultPrompt = vi.fn(async () => defaultStep("Custom prompt"));
    renderEditor({ onRestoreDefaultPrompt });

    await waitFor(() => expect(onRestoreDefaultPrompt).toHaveBeenCalled());
    expect(screen.queryByTestId("planning-review-restore-default-prompt")).not.toBeInTheDocument();
  });

  it("shows an error and Retry when the default prompt cannot load", async () => {
    const user = userEvent.setup();
    const pending = deferredStep();
    const onRestoreDefaultPrompt = vi
      .fn<() => Promise<PlanningReviewStep | null>>()
      .mockImplementationOnce(async () => null)
      .mockImplementationOnce(() => pending.promise);
    renderEditor({ onRestoreDefaultPrompt });

    expect(await screen.findByTestId("planning-review-restore-default-prompt-error")).toHaveTextContent(
      PLANNING_SETTINGS_COPY.restorePromptError,
    );
    expect(screen.queryByTestId("planning-review-restore-default-prompt")).not.toBeInTheDocument();

    await user.click(screen.getByTestId("planning-review-restore-default-prompt-retry"));
    pending.resolve(defaultStep(DEFAULT_PROMPT));

    expect(await screen.findByTestId("planning-review-restore-default-prompt")).toBeInTheDocument();
    expect(screen.queryByTestId("planning-review-restore-default-prompt-error")).not.toBeInTheDocument();
  });

  it("replaces the prompt after confirm and saves only when Save Agent is clicked", async () => {
    const user = userEvent.setup();
    const onSave = vi.fn();
    const onRestoreDefaultPrompt = vi.fn(async () => defaultStep(DEFAULT_PROMPT));
    renderEditor({ onSave, onRestoreDefaultPrompt });

    await user.click(await screen.findByTestId("planning-review-step-toggle-1"));
    expect(screen.getByTestId("planning-review-step-body-1")).toHaveValue("Custom prompt");

    await user.click(screen.getByTestId("planning-review-restore-default-prompt"));
    expect(screen.getByRole("heading", { name: "Restore the default prompt?" })).toBeInTheDocument();
    expect(onSave).not.toHaveBeenCalled();

    await user.click(screen.getByTestId("planning-review-restore-default-prompt-confirm"));

    await waitFor(() => {
      expect(screen.getByTestId("planning-review-step-body-1")).toHaveValue(DEFAULT_PROMPT);
    });
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

  it("disables Save until Restore finishes and retries without a second confirm", async () => {
    const user = userEvent.setup();
    const onSave = vi.fn();
    const confirmLoad = deferredStep();
    const retryLoad = deferredStep();
    const onRestoreDefaultPrompt = vi
      .fn<() => Promise<PlanningReviewStep | null>>()
      .mockImplementationOnce(async () => defaultStep(DEFAULT_PROMPT))
      .mockImplementationOnce(() => confirmLoad.promise)
      .mockImplementationOnce(() => retryLoad.promise);
    renderEditor({ onSave, onRestoreDefaultPrompt });

    await user.click(await screen.findByTestId("planning-review-restore-default-prompt"));
    await user.click(screen.getByTestId("planning-review-restore-default-prompt-confirm"));

    await waitFor(() => expect(screen.getByTestId("planning-review-save")).toBeDisabled());
    expect(onSave).not.toHaveBeenCalled();

    confirmLoad.resolve(null);
    expect(await screen.findByTestId("planning-review-restore-default-prompt-error")).toBeInTheDocument();
    expect(screen.getByTestId("planning-review-save")).toBeEnabled();

    await user.click(screen.getByTestId("planning-review-restore-default-prompt-retry"));
    await waitFor(() => expect(screen.getByTestId("planning-review-save")).toBeDisabled());
    expect(screen.queryByRole("heading", { name: "Restore the default prompt?" })).not.toBeInTheDocument();

    retryLoad.resolve(defaultStep(DEFAULT_PROMPT));
    await waitFor(() => expect(screen.getByTestId("planning-review-save")).toBeEnabled());
    await user.click(screen.getByTestId("planning-review-step-toggle-1"));
    expect(screen.getByTestId("planning-review-step-body-1")).toHaveValue(DEFAULT_PROMPT);
  });
});
