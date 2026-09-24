import { describe, expect, it } from "bun:test";

import jiraIcon from "@/assets/icons/integrations/jira.svg";
import sentryIcon from "@/assets/icons/integrations/sentry.svg";

import { logoDarkInvertClass } from "./logoDarkMode";

describe("logoDarkInvertClass", () => {
  it("returns the dark-mode invert class for the Sentry logo", () => {
    expect(logoDarkInvertClass(sentryIcon)).toBe("dark:brightness-0 dark:invert");
  });

  it("returns nothing for an unrelated logo", () => {
    expect(logoDarkInvertClass(jiraIcon)).toBeUndefined();
  });
});
