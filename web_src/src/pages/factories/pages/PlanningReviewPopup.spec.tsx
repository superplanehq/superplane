import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ComponentProps } from "react";
import { describe, expect, it, vi } from "vitest";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";
import { PLANNING_REVIEW_DRAFT, type PlanningReviewStep } from "./planningReviewMockup";
import { PlanningReviewPopup } from "./PlanningReviewPopup";

function renderPopup(
  props: Omit<ComponentProps<typeof PlanningReviewPopup>, "onClose"> & { onClose?: () => void } = {},
) {
  return render(
    <ThemeProvider>
      <TooltipProvider>
        <PlanningReviewPopup onClose={props.onClose ?? vi.fn()} {...props} />
      </TooltipProvider>
    </ThemeProvider>,
  );
}

describe("PlanningReviewPopup", () => {
  it("shows one collapsed component and one expanded runner component", () => {
    renderPopup();

    const planning = screen.getByTestId("planning-review-component-plan-agent");
    const implementation = screen.getByTestId("planning-review-component-implementation-agent");

    expect(within(planning).getByRole("button")).toHaveAttribute("aria-expanded", "false");
    expect(within(planning).getByText("Read the issue and write an implementation plan.")).toBeInTheDocument();
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
    const steps = PLANNING_REVIEW_DRAFT.components[1].configuration.steps as PlanningReviewStep[];

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

    const steps = PLANNING_REVIEW_DRAFT.components[1].configuration.steps as PlanningReviewStep[];
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

  it("expands and collapses component blocks", async () => {
    const user = userEvent.setup();
    renderPopup();

    const planningToggle = screen.getByTestId("planning-review-component-toggle-plan-agent");
    const implementationToggle = screen.getByTestId("planning-review-component-toggle-implementation-agent");

    await user.click(planningToggle);
    expect(planningToggle).toHaveAttribute("aria-expanded", "true");
    expect(
      within(screen.getByTestId("planning-review-component-plan-agent")).getByTestId("planning-review-step-name-0"),
    ).toHaveValue("Clone Repo");

    await user.click(implementationToggle);
    expect(implementationToggle).toHaveAttribute("aria-expanded", "false");
  });

  it("renames the column from the title immediately", async () => {
    const onRename = vi.fn();
    const user = userEvent.setup();
    renderPopup({ onRename });

    await user.click(screen.getByTestId("planning-review-title"));
    const input = screen.getByTestId("planning-review-title-input");
    await user.clear(input);
    await user.type(input, "Plan review{Enter}");

    expect(screen.getByTestId("planning-review-title")).toHaveTextContent("Plan review");
    expect(onRename).toHaveBeenCalledWith("Plan review");
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
    await user.click(screen.getByTestId("planning-review-save"));

    expect(onSave).toHaveBeenCalledWith(
      expect.objectContaining({
        components: expect.arrayContaining([
          expect.objectContaining({
            id: "implementation-agent",
            concurrency: expect.objectContaining({ max: "8" }),
          }),
        ]),
      }),
    );
    expect(onClose).toHaveBeenCalled();
  });
});
