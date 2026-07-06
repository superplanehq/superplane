import { describe, expect, it } from "vitest";
import { withEventStatusBadgeClasses } from "./eventStatusBadge";

describe("withEventStatusBadgeClasses", () => {
  it("adds lighter dark fills with dark text for saturated badge colors", () => {
    expect(withEventStatusBadgeClasses("bg-emerald-500")).toBe(
      "bg-emerald-500 dark:bg-emerald-400 dark:text-emerald-950",
    );
    expect(withEventStatusBadgeClasses("bg-violet-400")).toBe("bg-violet-400 dark:bg-violet-400 dark:text-violet-950");
  });

  it("leaves badge colors that already include dark classes unchanged", () => {
    expect(withEventStatusBadgeClasses("bg-emerald-500 dark:bg-emerald-400 dark:text-emerald-950")).toBe(
      "bg-emerald-500 dark:bg-emerald-400 dark:text-emerald-950",
    );
  });
});
