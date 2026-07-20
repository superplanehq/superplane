import { describe, expect, it } from "vitest";
import { computeDropdownPosition } from "./dropdownPosition";

const viewportHeight = 800;

describe("computeDropdownPosition", () => {
  it("opens downward when there is enough room below", () => {
    const rect = { top: 100, bottom: 130, left: 40, width: 200 };

    const position = computeDropdownPosition(rect, viewportHeight);

    expect(position.top).toBe(rect.bottom + 4);
    expect(position.bottom).toBeUndefined();
    expect(position.left).toBe(40);
    expect(position.width).toBe(200);
    expect(position.maxHeight).toBe(240);
  });

  it("opens upward when the trigger is the last field near the bottom", () => {
    const rect = { top: 740, bottom: 770, left: 40, width: 200 };

    const position = computeDropdownPosition(rect, viewportHeight);

    expect(position.top).toBeUndefined();
    // Bottom of the dropdown sits just above the trigger's top edge.
    expect(position.bottom).toBe(viewportHeight - rect.top + 4);
    expect(position.maxHeight).toBe(240);
  });

  it("caps the max height to the room available when opening upward in a short viewport", () => {
    const rect = { top: 150, bottom: 180, left: 0, width: 100 };

    const position = computeDropdownPosition(rect, 300);

    expect(position.top).toBeUndefined();
    expect(position.bottom).toBe(300 - rect.top + 4);
    // Only rect.top - gap of room exists above the trigger.
    expect(position.maxHeight).toBe(rect.top - 4);
  });

  it("stays downward and keeps full height when space below is still enough", () => {
    // ~250px below (> preferred 240) so it stays downward and keeps the full height.
    const rect = { top: 300, bottom: 546, left: 0, width: 100 };

    const position = computeDropdownPosition(rect, viewportHeight);

    expect(position.top).toBe(rect.bottom + 4);
    expect(position.maxHeight).toBe(240);
  });

  it("respects a custom gap and preferred height", () => {
    const rect = { top: 10, bottom: 40, left: 0, width: 100 };

    const position = computeDropdownPosition(rect, viewportHeight, { gap: 8, preferredMaxHeight: 300 });

    expect(position.top).toBe(rect.bottom + 8);
    expect(position.maxHeight).toBe(300);
  });
});
