import type { OrganizationsIntegration } from "@/api-client";
import { githubHostFromIntegration } from "@/lib/githubHost";

export function GitHubHostBadge({ integration }: { integration: OrganizationsIntegration }) {
  const host = githubHostFromIntegration(integration);
  if (!host) {
    return null;
  }

  return (
    <span
      data-testid="github-host-badge"
      className="max-w-[16rem] truncate rounded-md bg-muted px-1.5 py-0.5 font-mono text-xs text-muted-foreground"
      title={host}
    >
      {host}
    </span>
  );
}
