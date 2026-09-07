import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ComponentProps } from "react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";
import { PLANNING_REVIEW_DRAFT, type PlanningReviewStep } from "./planningReviewMockup";
import { PlanningReviewPopup } from "./PlanningReviewPopup";

vi.mock("@monaco-editor/react", () => ({
  Editor: ({ value, onChange }: { value?: string; onChange?: (value: string | undefined) => void }) => (
    <textarea value={value ?? ""} onChange={(event) => onChange?.(event.target.value)} />
  ),
}));

function renderPopup(
  props: Omit<ComponentProps<typeof PlanningReviewPopup>, "onClose"> & { onClose?: () => void } = {},
) {
  return render(
    <MemoryRouter>
      <ThemeProvider>
        <TooltipProvider>
          <PlanningReviewPopup onClose={props.onClose ?? vi.fn()} {...props} />
        </TooltipProvider>
      </ThemeProvider>
    </MemoryRouter>,
  );
}

describe("PlanningReviewPopup", () => {
  it("uses the agent name as the title", () => {
    renderPopup();

    expect(screen.getByRole("heading", { level: 2, name: "Implement From Task Description" })).toBeInTheDocument();
    expect(screen.queryByTestId("planning-review-title-input")).not.toBeInTheDocument();
  });

  it("keeps one agent even when the draft lists several", () => {
    renderPopup({
      initialDraft: {
        ...PLANNING_REVIEW_DRAFT,
        components: [
          {
            id: "plan-agent",
            title: "Agent - Plan for GH Issue",
            description: "Read the issue and write an implementation plan.",
            expanded: false,
            configuration: PLANNING_REVIEW_DRAFT.components[0].configuration,
            concurrency: { max: "1", key: "" },
          },
          ...PLANNING_REVIEW_DRAFT.components,
        ],
      },
    });

    expect(screen.queryByTestId("planning-review-component-plan-agent")).not.toBeInTheDocument();
    expect(screen.getByTestId("planning-review-component-implementation-agent")).toBeInTheDocument();
    expect(screen.getByRole("heading", { level: 2, name: "Implement From Task Description" })).toBeInTheDocument();
  });

  it("notes that agents are part of an automation and links to the automation editor", () => {
    renderPopup({ automationHref: "/organizations/org-1/factories/refunds/apps/app-1?configure=1" });

    const note = screen.getByTestId("planning-review-automation-note");
    expect(within(note).getByText(/Agents are part of an automation/)).toBeInTheDocument();
    expect(within(note).getByRole("link", { name: "Edit Automation" })).toHaveAttribute(
      "href",
      "/organizations/org-1/factories/refunds/apps/app-1?configure=1",
    );
  });

  it("keeps the note without a link when the automation is unknown", () => {
    renderPopup();

    const note = screen.getByTestId("planning-review-automation-note");
    expect(within(note).queryByRole("link")).not.toBeInTheDocument();
  });

  it("shows runner fields from the runner navigation item", async () => {
    const user = userEvent.setup();
    renderPopup();

    await user.click(screen.getByTestId("planning-review-nav-runner"));

    const row = screen.getByTestId("planning-review-environment-model-row");
    expect(within(row).getByText("Environment")).toBeInTheDocument();
    expect(within(row).getByText("Model")).toBeInTheDocument();
    expect(within(row).getByText("Working directory")).toBeInTheDocument();
    expect(within(row).getByText("Execution timeout (seconds)")).toBeInTheDocument();
  });

  it("names the agent once, in the header, and describes it below the name", () => {
    renderPopup();

    expect(screen.getAllByText("Implement From Task Description")).toHaveLength(1);
    expect(screen.getByTestId("planning-review-description")).toHaveTextContent(
      "Implement the approved plan and prepare the branch for review.",
    );
  });

  it("shows the runner settings without an extra collapsed wrapper", () => {
    renderPopup();

    const implementation = screen.getByTestId("planning-review-component-implementation-agent");

    expect(screen.queryByTestId("planning-review-component-toggle-implementation-agent")).not.toBeInTheDocument();
    expect(within(implementation).getByText("Steps")).toBeInTheDocument();
    expect(within(implementation).queryByText("Working directory")).not.toBeInTheDocument();
    expect(within(implementation).queryByText("Credentials")).not.toBeInTheDocument();
    expect(within(implementation).queryByText("Environment from")).not.toBeInTheDocument();
    expect(within(implementation).queryByText("Environment variables")).not.toBeInTheDocument();
    expect(within(implementation).queryByText("Execution timeout (seconds)")).not.toBeInTheDocument();
    expect(within(implementation).queryByText("Concurrency")).not.toBeInTheDocument();
  });

  it("shows every step collapsed with a name and kind badge", () => {
    renderPopup();

    const implementation = screen.getByTestId("planning-review-component-implementation-agent");
    const steps = PLANNING_REVIEW_DRAFT.components[0].configuration.steps as PlanningReviewStep[];

    steps.forEach((step, index) => {
      const row = within(implementation).getByTestId(`planning-review-step-${index}`);
      expect(within(row).getByTestId(`planning-review-step-summary-${index}`)).toHaveTextContent(step.name);
      expect(within(row).getByTestId(`planning-review-step-toggle-${index}`)).toHaveAttribute("aria-expanded", "false");
      expect(within(row).getByText(step.type === "prompt" ? "Prompt" : "Bash")).toBeInTheDocument();
    });
  });

  it("changes a step kind from the chip menu", async () => {
    const user = userEvent.setup();
    renderPopup();

    await user.click(screen.getByTestId("planning-review-step-kind-0"));
    await user.click(screen.getByTestId("planning-review-step-kind-0-prompt"));

    expect(screen.getByTestId("planning-review-step-kind-0")).toHaveTextContent("Prompt");
  });

  it("removes a step from the trash control", async () => {
    const user = userEvent.setup();
    renderPopup();

    const steps = PLANNING_REVIEW_DRAFT.components[0].configuration.steps as PlanningReviewStep[];
    await user.click(screen.getByTestId("planning-review-step-remove-0"));

    expect(screen.getByTestId("planning-review-step-summary-0")).toHaveTextContent(steps[1].name);
  });

  it("opens each settings group from the navigation", async () => {
    const user = userEvent.setup();
    renderPopup();
    const implementation = screen.getByTestId("planning-review-component-implementation-agent");

    await user.click(screen.getByTestId("planning-review-nav-credentials"));
    expect(within(implementation).getByText("Credentials")).toBeInTheDocument();

    await user.click(screen.getByTestId("planning-review-nav-environment"));
    expect(within(implementation).getByText("Environment from")).toBeInTheDocument();
    expect(within(implementation).getByText("Environment variables")).toBeInTheDocument();

    await user.click(screen.getByTestId("planning-review-nav-concurrency"));
    expect(screen.getByText("Max parallel executions")).toBeInTheDocument();
    expect(screen.getByText("Key")).toBeInTheDocument();
  });

  it("saves component configuration and closes", async () => {
    const onClose = vi.fn();
    const onSave = vi.fn();
    const user = userEvent.setup();
    renderPopup({ onClose, onSave });

    await user.click(screen.getByTestId("planning-review-nav-concurrency"));
    const max = screen.getByTestId("planning-review-concurrency-max-implementation-agent");
    await user.clear(max);
    await user.type(max, "8");
    expect(screen.getByTestId("planning-review-save")).toHaveTextContent("Save Agent");
    await user.click(screen.getByTestId("planning-review-save"));

    await waitFor(() => {
      expect(onSave).toHaveBeenCalledWith(
        expect.objectContaining({
          components: [
            expect.objectContaining({
              id: "implementation-agent",
              concurrency: expect.objectContaining({ max: "8" }),
            }),
          ],
        }),
      );
    });
    expect(onClose).toHaveBeenCalled();
  });

  it("keeps the popup open when save fails", async () => {
    const onClose = vi.fn();
    const onSave = vi.fn().mockRejectedValue(new Error("stage failed"));
    const user = userEvent.setup();
    renderPopup({ onClose, onSave });

    await user.click(screen.getByTestId("planning-review-save"));

    await waitFor(() => expect(onSave).toHaveBeenCalled());
    expect(onClose).not.toHaveBeenCalled();
    expect(screen.getByTestId("planning-review-save")).toBeInTheDocument();
  });

  it("shows a loading state instead of the mock agent", () => {
    renderPopup({ isLoading: true, initialDraft: { title: "Editing Agent", components: [] } });

    expect(screen.getByTestId("planning-review-loading")).toHaveTextContent("Loading agent…");
    expect(screen.queryByText("Steps")).not.toBeInTheDocument();
    expect(screen.getByTestId("planning-review-save")).toBeDisabled();
  });
});
