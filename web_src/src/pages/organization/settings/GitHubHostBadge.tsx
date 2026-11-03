import type { OrganizationsIntegration } from "@/api-client";
import { Badge } from "@/components/ui/badge";
import { githubHostFromIntegration } from "@/lib/githubHost";

export function GitHubHostBadge({ integration }: { integration: OrganizationsIntegration }) {
  const host = githubHostFromIntegration(integration);
  if (!host) {
    return null;
  }

  return (
    <Badge
      data-testid="github-host-badge"
      variant="secondary"
      className="max-w-[16rem] min-w-0 truncate font-mono font-normal text-muted-foreground"
      title={host}
    >
      {host}
    </Badge>
  );
}
