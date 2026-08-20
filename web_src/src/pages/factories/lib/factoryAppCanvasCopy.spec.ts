import { describe, expect, it } from "vitest";

import {
  isFactoryAppAgentPanelOpen,
  isFactoryAppAgentPromptOpen,
  isFactoryAppComponentsOpen,
  isFactoryAppConfigureMode,
  isFactoryAppYamlViewOpen,
  resolveFactoryAppCanvasSubtitle,
} from "./factoryAppCanvasCopy";

describe("factory app canvas search flags", () => {
  it("detects configure mode", () => {
    expect(isFactoryAppConfigureMode(new URLSearchParams("configure=1"))).toBe(true);
    expect(isFactoryAppConfigureMode(new URLSearchParams("edit=1"))).toBe(true);
    expect(isFactoryAppConfigureMode(new URLSearchParams("from=lines"))).toBe(false);
  });

  it("detects the desktop agent prompt flag", () => {
    expect(isFactoryAppAgentPromptOpen(new URLSearchParams("agentPrompt=1"))).toBe(true);
    expect(isFactoryAppAgentPromptOpen(new URLSearchParams("from=lines"))).toBe(false);
  });

  it("detects the YAML view flag", () => {
    expect(isFactoryAppYamlViewOpen(new URLSearchParams("yaml=1"))).toBe(true);
    expect(isFactoryAppYamlViewOpen(new URLSearchParams("from=lines"))).toBe(false);
  });

  it("detects the agent and components workspace flags", () => {
    expect(isFactoryAppAgentPanelOpen(new URLSearchParams("agent=1"))).toBe(true);
    expect(isFactoryAppComponentsOpen(new URLSearchParams("blocks=1"))).toBe(true);
    expect(isFactoryAppAgentPanelOpen(new URLSearchParams("from=lines"))).toBe(false);
    expect(isFactoryAppComponentsOpen(new URLSearchParams("from=lines"))).toBe(false);
  });
});

describe("resolveFactoryAppCanvasSubtitle", () => {
  it("keeps the canvas description in view and edit mode", () => {
    expect(
      resolveFactoryAppCanvasSubtitle({
        description: "Applies the plan across affected repos and opens PRs.",
        factoryName: "Refunds Factory",
      }),
    ).toBe("Applies the plan across affected repos and opens PRs.");
  });

  it("falls back to the workspace name when the canvas has no description", () => {
    expect(resolveFactoryAppCanvasSubtitle({ factoryName: "Refunds Factory" })).toBe("Canvas · Refunds Factory");
  });
});
