import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import type { ComponentProps } from "react";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

import type { FactoryApp } from "@/api-client";

import { AutomationCard } from "./automationsPageParts";
import { duplicateAutomationName } from "./automationCardActions";

const app: FactoryApp = {
  id: "app-refund-planner",
  name: "Refund Planner",
  description: "Plans reconciliation work across ledger + payment services.",
};

function renderCard(props: Partial<ComponentProps<typeof AutomationCard>> = {}) {
  return render(
    <MemoryRouter>
      <AutomationCard app={app} tick="passed" statusLabel="Passed" emphasized {...props} />
    </MemoryRouter>,
  );
}

describe("duplicateAutomationName", () => {
  it("appends copy to the trimmed name", () => {
    expect(duplicateAutomationName("Refund Planner")).toBe("Refund Planner copy");
  });

  it("falls back when the name is empty", () => {
    expect(duplicateAutomationName("  ")).toBe("Unnamed automation copy");
  });
});

describe("AutomationCard menu", () => {
  const actions = {
    onEdit: vi.fn(),
    onDuplicate: vi.fn(),
    onDelete: vi.fn(),
    canEdit: true,
    canDuplicate: true,
    canDelete: true,
  };

  it("hides the list-card menu until hover, then opens Edit, Duplicate, and Delete", async () => {
    const user = userEvent.setup();

    render(
      <MemoryRouter>
        <AutomationCard
          app={app}
          tick="passed"
          statusLabel="Passed"
          href="/automations/app-refund-planner"
          actions={actions}
        />
      </MemoryRouter>,
    );

    const trigger = screen.getByTestId("automations-card-menu");
    expect(trigger).toHaveClass("opacity-0");

    await user.click(trigger);
    expect(screen.getByTestId("automations-card-edit")).toHaveTextContent("Edit");
    expect(screen.getByTestId("automations-card-duplicate")).toHaveTextContent("Duplicate");
    expect(screen.getByRole("separator")).toBeInTheDocument();
    expect(screen.getByTestId("automations-card-delete")).toHaveTextContent("Delete");
  });

  it("keeps the detail-card menu visible and opens Edit, Duplicate, and Delete", async () => {
    const user = userEvent.setup();

    renderCard({ actions });

    const trigger = screen.getByTestId("automations-card-menu");
    expect(trigger).not.toHaveClass("opacity-0");

    await user.click(trigger);
    expect(screen.getByTestId("automations-card-edit")).toHaveTextContent("Edit");
    expect(screen.getByTestId("automations-card-duplicate")).toHaveTextContent("Duplicate");
    expect(screen.getByRole("separator")).toBeInTheDocument();
    expect(screen.getByTestId("automations-card-delete")).toHaveTextContent("Delete");
    expect(screen.getByTestId("automations-card-delete")).toHaveClass("text-destructive");

    await user.click(screen.getByTestId("automations-card-edit"));
    expect(actions.onEdit).toHaveBeenCalledTimes(1);
  });
});
