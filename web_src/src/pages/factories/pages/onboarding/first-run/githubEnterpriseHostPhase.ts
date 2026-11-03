export type GitHubEnterpriseHostPhase = "host" | "organization" | "app";

export function githubEnterpriseHostPhase(args: {
  integrationId?: string;
  ready?: boolean;
  currentStepName?: string;
  showHost?: boolean;
}): GitHubEnterpriseHostPhase {
  if (args.showHost || args.ready) return "host";
  if (args.currentStepName === "createApp") return "app";
  if (args.integrationId?.trim()) return "organization";
  return "host";
}
