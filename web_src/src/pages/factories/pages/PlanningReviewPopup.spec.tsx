import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ComponentProps } from "react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";
import { PLANNING_REVIEW_DRAFT, type PlanningReviewStep } from "./planningReviewMockup";
import { PlanningReviewPopup } from "./PlanningReviewPopup";

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

    expect(
      screen.getByRole("heading", { level: 2, name: "Agent - Implement from order description" }),
    ).toBeInTheDocument();
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
    expect(
      screen.getByRole("heading", { level: 2, name: "Agent - Implement from order description" }),
    ).toBeInTheDocument();
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

  it("places Environment and Model on one row under Steps", () => {
    renderPopup();

    const steps = screen.getByRole("heading", { name: "Steps" });
    const row = screen.getByTestId("planning-review-environment-model-row");
    expect(within(row).getByText("Environment")).toBeInTheDocument();
    expect(within(row).getByText("Model")).toBeInTheDocument();
    expect(steps.compareDocumentPosition(row) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
  });

  it("shows the runner component expanded", () => {
    renderPopup();

    const implementation = screen.getByTestId("planning-review-component-implementation-agent");

    expect(within(implementation).getByRole("button", { name: /Agent - Implement/ })).toHaveAttribute(
      "aria-expanded",
      "true",
    );
    expect(within(implementation).getByText("Steps")).toBeInTheDocument();
    expect(within(implementation).queryByText("Working directory")).not.toBeInTheDocument();
    expect(within(implementation).queryByText("Credentials")).not.toBeInTheDocument();
    expect(within(implementation).queryByText("Environment from")).not.toBeInTheDocument();
    expect(within(implementation).queryByText("Environment variables")).not.toBeInTheDocument();
    expect(within(implementation).queryByText("Execution timeout (seconds)")).not.toBeInTheDocument();
    expect(within(implementation).queryByText("Concurrency")).not.toBeInTheDocument();
  });

  it("shows every step as a row with a kind chip, a name, and a body", () => {
    renderPopup();

    const implementation = screen.getByTestId("planning-review-component-implementation-agent");
    const steps = PLANNING_REVIEW_DRAFT.components[0].configuration.steps as PlanningReviewStep[];

    steps.forEach((step, index) => {
      const row = within(implementation).getByTestId(`planning-review-step-${index}`);
      expect(within(row).getByTestId(`planning-review-step-name-${index}`)).toHaveValue(step.name);
      expect(within(row).getByTestId(`planning-review-step-body-${index}`)).toHaveValue(step.command ?? step.prompt);
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

    expect(screen.getByTestId("planning-review-step-name-0")).toHaveValue(steps[1].name);
  });

  it("reveals environment, timeout, and concurrency in more settings", async () => {
    const user = userEvent.setup();
    renderPopup();

    const implementation = screen.getByTestId("planning-review-component-implementation-agent");
    await user.click(within(implementation).getByTestId("planning-review-more-settings-toggle"));

    expect(within(implementation).getByText("Working directory")).toBeInTheDocument();
    expect(within(implementation).getByText("Credentials")).toBeInTheDocument();
    expect(within(implementation).getByText("Environment from")).toBeInTheDocument();
    expect(within(implementation).getByText("Environment variables")).toBeInTheDocument();
    expect(within(implementation).getByText("Execution timeout (seconds)")).toBeInTheDocument();
    expect(within(implementation).getByText("Concurrency")).toBeInTheDocument();
  });

  it("collapses the agent block", async () => {
    const user = userEvent.setup();
    renderPopup();

    const implementationToggle = screen.getByTestId("planning-review-component-toggle-implementation-agent");

    await user.click(implementationToggle);
    expect(implementationToggle).toHaveAttribute("aria-expanded", "false");
    expect(screen.queryByText("Steps")).not.toBeInTheDocument();
  });

  it("saves component configuration and closes", async () => {
    const onClose = vi.fn();
    const onSave = vi.fn();
    const user = userEvent.setup();
    renderPopup({ onClose, onSave });

    await user.click(screen.getByTestId("planning-review-more-settings-toggle"));
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
