import { describe, expect, it } from "vitest";

import { CONSOLE_PANEL_BODY_SURFACE, CONSOLE_PANEL_SHELL_SURFACE } from "./consolePanelStyles";

describe("console panel surfaces", () => {
  it("lifts panels with a subtle white tint in dark mode", () => {
    expect(CONSOLE_PANEL_SHELL_SURFACE).toBe("dark:bg-white/5");
    expect(CONSOLE_PANEL_BODY_SURFACE).toBe("dark:bg-transparent");
  });
});
