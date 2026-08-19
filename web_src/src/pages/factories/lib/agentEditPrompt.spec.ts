import { describe, expect, it } from "vitest";

import { getInstallCommand } from "@/lib/cli";

import {
  API_TOKEN_DISPLAY,
  API_TOKEN_PLACEHOLDER,
  DEFAULT_SUPERPLANE_BASE_URL,
  buildAgentEditPrompt,
  redactAgentEditPromptForDisplay,
} from "./agentEditPrompt";

describe("buildAgentEditPrompt", () => {
  const baseInput = {
    appName: "Refund Implementer",
    appId: "app-refund-implementer",
    baseUrl: DEFAULT_SUPERPLANE_BASE_URL,
  };

  it("includes CLI install, connect, get, and update commands", () => {
    const prompt = buildAgentEditPrompt(baseInput);

    expect(prompt).toContain(getInstallCommand());
    expect(prompt).toContain(`superplane connect ${DEFAULT_SUPERPLANE_BASE_URL} ${API_TOKEN_PLACEHOLDER}`);
    expect(prompt).toContain("superplane apps canvas get app-refund-implementer -o yaml > canvas.yaml");
    expect(prompt).toContain(
      'superplane apps canvas update app-refund-implementer -f canvas.yaml -m "Describe the change"',
    );
  });

  it("names the canvas and states that it is YAML-backed", () => {
    const prompt = buildAgentEditPrompt(baseInput);

    expect(prompt).toContain("Canvas name: Refund Implementer");
    expect(prompt).toContain("Canvas id: app-refund-implementer");
    expect(prompt).toContain("The canvas is YAML-backed.");
  });

  it("links the automation run when runId is set", () => {
    const prompt = buildAgentEditPrompt({ ...baseInput, runId: "run-implement-failed" });

    expect(prompt).toContain("This view is linked to automation run run-implement-failed.");
    expect(prompt).toContain("Inspect that run before you change the canvas.");
  });

  it("omits the run reference when runId is absent", () => {
    const prompt = buildAgentEditPrompt(baseInput);

    expect(prompt).not.toContain("This view is linked to automation run");
  });

  it("names the line when lineId is set", () => {
    const prompt = buildAgentEditPrompt({ ...baseInput, lineId: "line-plan-and-implement" });

    expect(prompt).toContain("This canvas belongs to line line-plan-and-implement.");
  });

  it("redacts the API token for display and keeps the placeholder for copy", () => {
    const prompt = buildAgentEditPrompt(baseInput);
    const displayed = redactAgentEditPromptForDisplay(prompt);

    expect(prompt).toContain(API_TOKEN_PLACEHOLDER);
    expect(displayed).toContain(API_TOKEN_DISPLAY);
    expect(displayed).not.toContain(API_TOKEN_PLACEHOLDER);
    expect(displayed).toContain(`superplane connect ${DEFAULT_SUPERPLANE_BASE_URL} ${API_TOKEN_DISPLAY}`);
  });
});
