import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { API_TOKEN_DISPLAY, API_TOKEN_PLACEHOLDER, buildAgentEditPrompt } from "../lib/agentEditPrompt";
import { AgentSetupPromptDialog } from "./AgentSetupPromptDialog";

describe("AgentSetupPromptDialog", () => {
  const prompt = buildAgentEditPrompt({
    appName: "Refund Implementer",
    appId: "app-refund-implementer",
    baseUrl: "https://app.superplane.com",
  });

  it("shows the raw markdown prompt with a redacted token", () => {
    render(<AgentSetupPromptDialog open onOpenChange={vi.fn()} prompt={prompt} />);

    const body = screen.getByTestId("agent-setup-prompt-markdown");
    expect(body.tagName).toBe("PRE");
    expect(body).toHaveClass("font-mono");
    expect(body).toHaveTextContent("```bash");
    expect(body).toHaveTextContent(API_TOKEN_DISPLAY);
    expect(body).not.toHaveTextContent(API_TOKEN_PLACEHOLDER);
    expect(screen.getByTestId("agent-setup-prompt-copy")).toHaveTextContent("Copy prompt and embed API key");
  });

  it("copies the prompt with the token placeholder", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText },
    });

    render(<AgentSetupPromptDialog open onOpenChange={vi.fn()} prompt={prompt} />);
    fireEvent.click(screen.getByTestId("agent-setup-prompt-copy"));

    expect(writeText).toHaveBeenCalledWith(prompt);
    expect(writeText.mock.calls[0]?.[0]).toContain(API_TOKEN_PLACEHOLDER);
  });
});
