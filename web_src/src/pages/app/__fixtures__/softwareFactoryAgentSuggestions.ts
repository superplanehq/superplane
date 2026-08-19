import type { AgentSuggestion } from "@/ui/CanvasPage";

/** Mock post-install Agent prompts for the Software Factory Live Canvas story. */
export const softwareFactoryAgentSuggestions: AgentSuggestion[] = [
  {
    id: "add-ci",
    label: "Add CI to my loop",
    prompt:
      "Add a CI step to this Software Factory loop so every implementation run is validated before review. Prefer reusing existing components on the canvas when possible.",
  },
  {
    id: "fail-notifications",
    label: "Add notifications if something fails",
    prompt:
      "Add failure notifications to this Software Factory app so the team is alerted when a run or critical step fails. Suggest a concrete channel (e.g. Slack) and wire it into the failure path.",
  },
  {
    id: "slack-approvals",
    label: "Connect Slack for approvals",
    prompt:
      "Connect Slack and add an approval step to this Software Factory loop so humans can approve before risky changes proceed.",
  },
  {
    id: "pr-checks",
    label: "Gate merges on review checks",
    prompt:
      "Update this Software Factory loop so merges are gated on review checks passing. Keep the change minimal and aligned with the existing PR/review nodes.",
  },
];
