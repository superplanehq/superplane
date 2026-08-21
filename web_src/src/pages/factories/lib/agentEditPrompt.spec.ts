import { describe, expect, it } from "vitest";

import { getInstallCommand } from "@/lib/cli";

import {
  API_TOKEN_PLACEHOLDER,
  DEFAULT_SUPERPLANE_BASE_URL,
  buildAgentCliInstallCommands,
  buildAgentCliInstallInstructions,
  buildAgentEditPrompt,
} from "./agentEditPrompt";

describe("buildAgentCliInstallCommands", () => {
  it("returns only runnable install lines", () => {
    const commands = buildAgentCliInstallCommands();

    expect(commands).toBe(`${getInstallCommand()}\nexport PATH="$HOME/.local/bin:$PATH"`);
    expect(commands).not.toContain("```");
    expect(commands).not.toContain("superplane connect");
  });
});

describe("buildAgentCliInstallInstructions", () => {
  it("includes CLI install and connect with a token placeholder", () => {
    const instructions = buildAgentCliInstallInstructions({ baseUrl: DEFAULT_SUPERPLANE_BASE_URL });

    expect(instructions).toContain(getInstallCommand());
    expect(instructions).toContain('export PATH="$HOME/.local/bin:$PATH"');
    expect(instructions).toContain(`superplane connect ${DEFAULT_SUPERPLANE_BASE_URL} ${API_TOKEN_PLACEHOLDER}`);
    expect(instructions).toContain("<YOUR_TOKEN>");
    expect(instructions).not.toContain("***REDACTED***");
  });
});

describe("buildAgentEditPrompt", () => {
  const baseInput = {
    appName: "Refund Implementer",
    appId: "app-refund-implementer",
  };

  it("includes canvas IDs and get/update hints without install or connect", () => {
    const prompt = buildAgentEditPrompt(baseInput);

    expect(prompt).toContain("Canvas name: Refund Implementer");
    expect(prompt).toContain("Canvas id: app-refund-implementer");
    expect(prompt).toContain("The canvas is YAML-backed.");
    expect(prompt).toContain("superplane apps canvas get app-refund-implementer -o yaml > canvas.yaml");
    expect(prompt).toContain(
      'superplane apps canvas update app-refund-implementer -f canvas.yaml -m "Describe the change"',
    );
    expect(prompt).not.toContain(getInstallCommand());
    expect(prompt).not.toContain("superplane connect");
    expect(prompt).not.toContain(API_TOKEN_PLACEHOLDER);
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
});
