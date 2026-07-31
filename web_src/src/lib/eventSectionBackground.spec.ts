import { describe, expect, it } from "vitest";
import { withEventSectionDarkBackground } from "./eventSectionBackground";

describe("withEventSectionDarkBackground", () => {
  it("adds dark 900/50 backgrounds for light tint classes", () => {
    expect(withEventSectionDarkBackground("bg-violet-100")).toBe("bg-violet-100 dark:bg-violet-900/50");
    expect(withEventSectionDarkBackground("bg-green-100")).toBe("bg-green-100 dark:bg-green-900/50");
    expect(withEventSectionDarkBackground("bg-surface-subtle")).toBe("bg-surface-subtle");
  });

  it("leaves background colors that already include dark classes unchanged", () => {
    expect(withEventSectionDarkBackground("bg-violet-100 dark:bg-violet-900/50")).toBe(
      "bg-violet-100 dark:bg-violet-900/50",
    );
  });

  it("leaves non-tint backgrounds unchanged", () => {
    expect(withEventSectionDarkBackground("bg-surface-raised")).toBe("bg-surface-raised");
  });
});
