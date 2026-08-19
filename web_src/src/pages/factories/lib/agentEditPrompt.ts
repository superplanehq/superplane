import { getInstallCommand } from "@/lib/cli";

export const DEFAULT_SUPERPLANE_BASE_URL = "https://app.superplane.com";
export const API_TOKEN_PLACEHOLDER = "<YOUR_API_TOKEN>";

export type AgentEditPromptInput = {
  appName: string;
  appId: string;
  baseUrl: string;
  runId?: string | null;
  lineId?: string | null;
};

export function buildAgentEditPrompt(input: AgentEditPromptInput): string {
  const appName = input.appName.trim() || "Untitled automation";
  const appId = input.appId.trim();
  const baseUrl = input.baseUrl.trim() || DEFAULT_SUPERPLANE_BASE_URL;
  const installCommand = getInstallCommand();

  const sections = [
    "You are editing a SuperPlane canvas.",
    "The canvas is YAML-backed. Use the SuperPlane CLI to read and update it.",
    "",
    `Canvas name: ${appName}`,
    `Canvas id: ${appId}`,
    ...buildLinkedContextLines(input.runId, input.lineId),
    "",
    "Follow these steps.",
    "",
    "1. Install the SuperPlane CLI.",
    "",
    "```bash",
    installCommand,
    'export PATH="$HOME/.local/bin:$PATH"',
    "```",
    "",
    "2. Connect with an API token.",
    "Create a token in SuperPlane organization settings. Then run:",
    "",
    "```bash",
    `superplane connect ${baseUrl} ${API_TOKEN_PLACEHOLDER}`,
    "```",
    "",
    "3. Get the canvas YAML.",
    "",
    "```bash",
    `superplane apps canvas get ${appId} -o yaml > canvas.yaml`,
    "```",
    "",
    "4. Update the canvas.",
    "Edit canvas.yaml. Then run:",
    "",
    "```bash",
    `superplane apps canvas update ${appId} -f canvas.yaml -m "Describe the change"`,
    "```",
  ];

  return sections.join("\n");
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
