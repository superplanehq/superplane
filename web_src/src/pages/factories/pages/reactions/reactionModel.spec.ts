import { describe, expect, it } from "vitest";

import { getMyReaction, toggleWorkOrderReaction } from "./reactionModel";

describe("toggleWorkOrderReaction", () => {
  it("adds a new reaction as a new group when no one reacted with that emoji yet", () => {
    const next = toggleWorkOrderReaction([], "👍", "Storybook User");
    expect(next).toEqual([{ emoji: "👍", reactorNames: ["Storybook User"] }]);
  });

  it("joins an existing group when someone else already used that emoji", () => {
    const next = toggleWorkOrderReaction([{ emoji: "👍", reactorNames: ["Alex Reviewer"] }], "👍", "Storybook User");
    expect(next).toEqual([{ emoji: "👍", reactorNames: ["Alex Reviewer", "Storybook User"] }]);
  });

  it("removes the reactor's own reaction on a second click of the same emoji (toggle off)", () => {
    const start = [{ emoji: "👍", reactorNames: ["Alex Reviewer", "Storybook User"] }];
    const next = toggleWorkOrderReaction(start, "👍", "Storybook User");
    expect(next).toEqual([{ emoji: "👍", reactorNames: ["Alex Reviewer"] }]);
  });

  it("drops an emoji group entirely once its last reactor removes their reaction", () => {
    const next = toggleWorkOrderReaction([{ emoji: "👍", reactorNames: ["Storybook User"] }], "👍", "Storybook User");
    expect(next).toEqual([]);
  });

  it("switches the reactor from their old emoji to a new one — only one active reaction per user", () => {
    const start = [
      { emoji: "👍", reactorNames: ["Alex Reviewer", "Storybook User"] },
      { emoji: "🎉", reactorNames: ["Jamie Operator"] },
    ];
    const next = toggleWorkOrderReaction(start, "🎉", "Storybook User");
    expect(next).toEqual([
      { emoji: "👍", reactorNames: ["Alex Reviewer"] },
      { emoji: "🎉", reactorNames: ["Jamie Operator", "Storybook User"] },
    ]);
  });
});

describe("getMyReaction", () => {
  it("returns the emoji the reactor is in, or null when they haven't reacted", () => {
    const groups = [{ emoji: "🚀", reactorNames: ["Alex Reviewer", "Storybook User"] }];
    expect(getMyReaction(groups, "Storybook User")).toBe("🚀");
    expect(getMyReaction(groups, "Jamie Operator")).toBeNull();
  });
});
