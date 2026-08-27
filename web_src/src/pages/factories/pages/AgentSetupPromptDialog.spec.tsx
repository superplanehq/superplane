import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { getInstallCommand } from "@/lib/cli";

vi.mock("@/components/AgentSidebar/widgets/MarkdownCode", () => ({
  MarkdownCode: ({ children }: { children?: string }) => <code data-testid="markdown-code">{children}</code>,
}));

import {
  API_TOKEN_PLACEHOLDER,
  DEFAULT_SUPERPLANE_BASE_URL,
  buildAgentCliInstallCommands,
  buildAgentCliInstallInstructions,
  buildAgentEditPrompt,
} from "../lib/agentEditPrompt";
import { AgentSetupPromptDialog } from "./AgentSetupPromptDialog";

describe("AgentSetupPromptDialog", () => {
  const installInstructions = buildAgentCliInstallInstructions({ baseUrl: DEFAULT_SUPERPLANE_BASE_URL });
  const installCommands = buildAgentCliInstallCommands();
  const prompt = buildAgentEditPrompt({
    appName: "Refund Implementer",
    appId: "app-refund-implementer",
  });

  it("shows install instructions and a prompt without a token", () => {
    render(
      <AgentSetupPromptDialog
        open
        onOpenChange={vi.fn()}
        installInstructions={installInstructions}
        installCommands={installCommands}
        prompt={prompt}
      />,
    );

    const install = screen.getByTestId("agent-setup-install-markdown");
    expect(install).toHaveTextContent(getInstallCommand());
    expect(install).toHaveTextContent(API_TOKEN_PLACEHOLDER);
    expect(install).not.toHaveTextContent("***REDACTED***");
    expect(install).not.toHaveTextContent("```bash");

    const promptBody = screen.getByTestId("agent-setup-prompt-markdown");
    expect(promptBody).toHaveTextContent("Canvas id: app-refund-implementer");
    expect(promptBody).not.toHaveTextContent("```bash");
    expect(promptBody).not.toHaveTextContent(getInstallCommand());
    expect(promptBody).not.toHaveTextContent(API_TOKEN_PLACEHOLDER);

    expect(screen.getByTestId("agent-setup-install-copy")).toHaveTextContent("Copy install commands");
    expect(screen.getByTestId("agent-setup-prompt-copy")).toHaveTextContent("Copy prompt");
  });

  it("copies only the runnable install commands", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText },
    });

    render(
      <AgentSetupPromptDialog
        open
        onOpenChange={vi.fn()}
        installInstructions={installInstructions}
        installCommands={installCommands}
        prompt={prompt}
      />,
    );
    fireEvent.click(screen.getByTestId("agent-setup-install-copy"));

    expect(writeText).toHaveBeenCalledWith(installCommands);
    expect(writeText.mock.calls[0]?.[0]).toContain(getInstallCommand());
    expect(writeText.mock.calls[0]?.[0]).not.toContain("```");
    expect(writeText.mock.calls[0]?.[0]).not.toContain("superplane connect");
  });

  it("copies the prompt without a token", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", {
      configurable: true,
      value: { writeText },
    });

    render(
      <AgentSetupPromptDialog
        open
        onOpenChange={vi.fn()}
        installInstructions={installInstructions}
        installCommands={installCommands}
        prompt={prompt}
      />,
    );
    fireEvent.click(screen.getByTestId("agent-setup-prompt-copy"));

    expect(writeText).toHaveBeenCalledWith(prompt);
    expect(writeText.mock.calls[0]?.[0]).not.toContain(API_TOKEN_PLACEHOLDER);
    expect(writeText.mock.calls[0]?.[0]).not.toContain(getInstallCommand());
  });
});
