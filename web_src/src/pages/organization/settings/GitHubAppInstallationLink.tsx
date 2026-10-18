import type { OrganizationsIntegration } from "@/api-client";
import { Button } from "@/components/ui/button";
import { isUrl } from "@/lib/utils";
import { ExternalLink } from "lucide-react";

const githubAppInstallationURLProperty = "appInstallationURL";

function githubAppInstallationURL(integration: OrganizationsIntegration): string | null {
  if (integration.metadata?.integrationName !== "github") {
    return null;
  }
  const value = integration.status?.properties
    ?.find((property) => property.name === githubAppInstallationURLProperty)
    ?.value?.trim();
  if (!value || !isUrl(value)) {
    return null;
  }
  return value;
}

export function GitHubAppInstallationLink({ integration }: { integration: OrganizationsIntegration }) {
  const url = githubAppInstallationURL(integration);
  if (!url) {
    return null;
  }

  return (
    <Button variant="outline" size="sm" asChild>
      <a
        href={url}
        target="_blank"
        rel="noopener noreferrer"
        data-testid="github-app-installation-link"
        aria-label="Open GitHub App"
      >
        GitHub App
        <ExternalLink className="size-3.5" aria-hidden />
      </a>
    </Button>
  );
}
