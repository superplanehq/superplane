import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { KanbanCardMotionItem } from "./KanbanCardMotionItem";
import { KANBAN_CARD_TRANSITION_CLASS, kanbanViewTransitionName } from "./kanbanCardMotion";

describe("KanbanCardMotionItem", () => {
  it("sets a view-transition name from the work-order id", () => {
    render(
      <ul>
        <KanbanCardMotionItem id="9f2a291e-057d-4159-8eba-b4c59a125df2">Card</KanbanCardMotionItem>
      </ul>,
    );

    const item = screen.getByText("Card");
    expect(item.style.viewTransitionName).toBe(kanbanViewTransitionName("9f2a291e-057d-4159-8eba-b4c59a125df2"));
    expect(item.style.viewTransitionClass).toBe(KANBAN_CARD_TRANSITION_CLASS);
  });

  it("leaves the create ghost without a view-transition name", () => {
    render(
      <ul>
        <KanbanCardMotionItem data-testid="ghost">Create</KanbanCardMotionItem>
      </ul>,
    );

    const item = screen.getByTestId("ghost");
    expect(item.style.viewTransitionName).toBe("");
  });
});
