import { getInstallCommand } from "@/lib/cli";

export const DEFAULT_SUPERPLANE_BASE_URL = "https://app.superplane.com";
export const API_TOKEN_PLACEHOLDER = "<YOUR_TOKEN>";

export type AgentCliInstallInstructionsInput = {
  baseUrl: string;
};

export type AgentEditPromptInput = {
  appName: string;
  appId: string;
  runId?: string | null;
  lineId?: string | null;
};

/** Runnable install lines. Paste these into a terminal. */
export function buildAgentCliInstallCommands(): string {
  return [getInstallCommand(), 'export PATH="$HOME/.local/bin:$PATH"'].join("\n");
}

export function buildAgentCliInstallInstructions(input: AgentCliInstallInstructionsInput): string {
  const baseUrl = input.baseUrl.trim() || DEFAULT_SUPERPLANE_BASE_URL;

  return [
    "Install the SuperPlane CLI.",
    "",
    "```bash",
    buildAgentCliInstallCommands(),
    "```",
    "",
    "Connect with an API token.",
    "Create a token in SuperPlane organization settings. Then run:",
    "",
    "```bash",
    `superplane connect ${baseUrl} ${API_TOKEN_PLACEHOLDER}`,
    "```",
  ].join("\n");
}

export function buildAgentEditPrompt(input: AgentEditPromptInput): string {
  const appName = input.appName.trim() || "Untitled automation";
  const appId = input.appId.trim();

  return [
    "You are editing a SuperPlane canvas.",
    "The canvas is YAML-backed. Use the SuperPlane CLI to read and update it.",
    "",
    `Canvas name: ${appName}`,
    `Canvas id: ${appId}`,
    ...buildLinkedContextLines(input.runId, input.lineId),
    "",
    "Get the canvas:",
    "",
    "```bash",
    `superplane apps canvas get ${appId} -o yaml > canvas.yaml`,
    "```",
    "",
    "Update the canvas:",
    "",
    "```bash",
    `superplane apps canvas update ${appId} -f canvas.yaml -m "Describe the change"`,
    "```",
  ].join("\n");
}

function buildLinkedContextLines(runId?: string | null, lineId?: string | null): string[] {
  const lines: string[] = [];
  const trimmedRunId = runId?.trim();
  if (trimmedRunId) {
    lines.push(`This view is linked to automation run ${trimmedRunId}.`);
    lines.push("Inspect that run before you change the canvas.");
  }
  const trimmedLineId = lineId?.trim();
  if (trimmedLineId) {
    lines.push(`This canvas belongs to line ${trimmedLineId}.`);
  }
  return lines;
}
