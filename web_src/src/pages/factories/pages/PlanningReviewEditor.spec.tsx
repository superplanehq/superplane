import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import { RESTORE_DEFAULT_PROMPT_COPY } from "../lib/defaultAgentPrompt";
import { loadDefaultAgentPrompt } from "../lib/loadDefaultAgentPrompt";
import { PlanningReviewEditor } from "./PlanningReviewEditor";
import type { PlanningReviewDraft, PlanningReviewStep } from "./planningReviewMockup";

vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
}));

vi.mock("../lib/loadDefaultAgentPrompt", () => ({
  loadDefaultAgentPrompt: vi.fn(),
}));

const DEFAULT_PROMPT = "Plan the task from the factory default.";
const IMPLEMENT_PROMPT = "Implement the task from the factory default.";
const PR_PROMPT = "Write the pull request from the factory default.";

const loadDefault = vi.mocked(loadDefaultAgentPrompt);

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

function implementationDraft(): PlanningReviewDraft {
  return {
    title: "Implement",
    components: [
      {
        id: "implementation-agent-no-issue",
        title: "Implement From Task Description",
        description: "",
        expanded: true,
        configuration: {
          steps: [
            { name: "Clone Repo", type: "bash", command: "git clone" },
            { name: "Implementation", type: "prompt", prompt: "Custom implement", workingDirectory: "repo" },
            {
              name: "Generate PR title and description",
              type: "prompt",
              prompt: "Custom pr",
              workingDirectory: "repo",
            },
          ],
        },
        concurrency: { max: "1", key: "" },
      },
    ],
  };
}

function renderEditor(
  props: {
    initialDraft?: PlanningReviewDraft;
    onSave?: (draft: PlanningReviewDraft) => void | Promise<void>;
    organizationId?: string;
    factoryId?: string;
    automationId?: string;
  } = {},
) {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <PlanningReviewEditor
              initialDraft={props.initialDraft ?? refineDraft("Custom prompt")}
              onSave={props.onSave}
              organizationId={props.organizationId ?? "org-1"}
              factoryId={props.factoryId ?? "factory-1"}
              automationId={props.automationId ?? "canvas-1"}
              showAutomationNote={false}
              showCancel={false}
            />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

function defaultSteps(prompt: string): PlanningReviewStep[] {
  return [
    {
      name: "Refine Task",
      type: "prompt",
      prompt,
      workingDirectory: "repo",
    },
  ];
}

function deferredSteps() {
  let resolve: (steps: PlanningReviewStep[]) => void = () => undefined;
  const promise = new Promise<PlanningReviewStep[]>((settle) => {
    resolve = settle;
  });
  return { promise, resolve };
}

describe("PlanningReviewEditor restore default prompt", () => {
  beforeEach(() => {
    loadDefault.mockReset();
  });

  it("hides Restore default prompt when the automation id is missing", () => {
    renderEditor({ automationId: "" });

    expect(screen.queryByTestId("planning-review-restore-default-prompt")).not.toBeInTheDocument();
    expect(loadDefault).not.toHaveBeenCalled();
  });

  it("hides Restore default prompt when the prompt already matches", async () => {
    loadDefault.mockResolvedValue(defaultSteps("Custom prompt"));
    renderEditor();

    await waitFor(() => expect(loadDefault).toHaveBeenCalled());
    expect(screen.queryByTestId("planning-review-restore-default-prompt")).not.toBeInTheDocument();
  });

  it("shows an error and Retry when the default prompt cannot load", async () => {
    const user = userEvent.setup();
    const pending = deferredSteps();
    loadDefault.mockResolvedValueOnce([]).mockImplementationOnce(() => pending.promise);
    renderEditor();

    expect(await screen.findByTestId("planning-review-restore-default-prompt-error")).toHaveTextContent(
      RESTORE_DEFAULT_PROMPT_COPY.error,
    );
    expect(screen.queryByTestId("planning-review-restore-default-prompt")).not.toBeInTheDocument();

    await user.click(screen.getByTestId("planning-review-restore-default-prompt-retry"));
    pending.resolve(defaultSteps(DEFAULT_PROMPT));

    expect(await screen.findByTestId("planning-review-restore-default-prompt")).toBeInTheDocument();
    expect(screen.queryByTestId("planning-review-restore-default-prompt-error")).not.toBeInTheDocument();
  });

  it("replaces the prompt after confirm and saves only when Save Agent is clicked", async () => {
    const user = userEvent.setup();
    const onSave = vi.fn();
    loadDefault.mockResolvedValue(defaultSteps(DEFAULT_PROMPT));
    renderEditor({ onSave });

    await user.click(await screen.findByTestId("planning-review-step-toggle-1"));
    expect(screen.getByTestId("planning-review-step-body-1")).toHaveValue("Custom prompt");

    await user.click(screen.getByTestId("planning-review-restore-default-prompt"));
    expect(screen.getByRole("heading", { name: RESTORE_DEFAULT_PROMPT_COPY.confirmTitle })).toBeInTheDocument();
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

  it("replaces every prompt on an implementation agent after confirm", async () => {
    const user = userEvent.setup();
    loadDefault.mockResolvedValue([
      { name: "Implementation", type: "prompt", prompt: IMPLEMENT_PROMPT },
      { name: "Generate PR title and description", type: "prompt", prompt: PR_PROMPT },
    ]);
    renderEditor({ initialDraft: implementationDraft() });

    expect(await screen.findByTestId("planning-review-restore-default-prompt")).toBeInTheDocument();
    await user.click(screen.getByTestId("planning-review-restore-default-prompt"));
    await user.click(screen.getByTestId("planning-review-restore-default-prompt-confirm"));

    await user.click(await screen.findByTestId("planning-review-step-toggle-1"));
    expect(screen.getByTestId("planning-review-step-body-1")).toHaveValue(IMPLEMENT_PROMPT);
    await user.click(screen.getByTestId("planning-review-step-toggle-2"));
    expect(screen.getByTestId("planning-review-step-body-2")).toHaveValue(PR_PROMPT);
    expect(loadDefault).toHaveBeenCalledWith({
      organizationId: "org-1",
      factoryId: "factory-1",
      automationId: "canvas-1",
      agentNodeId: "implementation-agent-no-issue",
    });
  });

  it("disables Save until Restore finishes and retries without a second confirm", async () => {
    const user = userEvent.setup();
    const onSave = vi.fn();
    const confirmLoad = deferredSteps();
    const retryLoad = deferredSteps();
    loadDefault
      .mockResolvedValueOnce(defaultSteps(DEFAULT_PROMPT))
      .mockImplementationOnce(() => confirmLoad.promise)
      .mockImplementationOnce(() => retryLoad.promise);
    renderEditor({ onSave });

    await user.click(await screen.findByTestId("planning-review-restore-default-prompt"));
    await user.click(screen.getByTestId("planning-review-restore-default-prompt-confirm"));

    await waitFor(() => expect(screen.getByTestId("planning-review-save")).toBeDisabled());
    expect(onSave).not.toHaveBeenCalled();

    confirmLoad.resolve([]);
    expect(await screen.findByTestId("planning-review-restore-default-prompt-error")).toBeInTheDocument();
    expect(screen.getByTestId("planning-review-save")).toBeEnabled();

    await user.click(screen.getByTestId("planning-review-restore-default-prompt-retry"));
    await waitFor(() => expect(screen.getByTestId("planning-review-save")).toBeDisabled());
    expect(screen.queryByRole("heading", { name: RESTORE_DEFAULT_PROMPT_COPY.confirmTitle })).not.toBeInTheDocument();

    retryLoad.resolve(defaultSteps(DEFAULT_PROMPT));
    await waitFor(() => expect(screen.getByTestId("planning-review-save")).toBeEnabled());
    await user.click(screen.getByTestId("planning-review-step-toggle-1"));
    expect(screen.getByTestId("planning-review-step-body-1")).toHaveValue(DEFAULT_PROMPT);
  });
});
