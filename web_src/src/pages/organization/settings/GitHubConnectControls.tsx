import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import type { IntegrationsIntegrationDefinition } from "@/api-client/types.gen";
import { CREATE_PRIVATE_GITHUB_APP_LABEL } from "@/lib/privateGitHubApp";

interface GitHubConnectControlsProps {
  definition: IntegrationsIntegrationDefinition | undefined;
  canCreateIntegrations: boolean;
  permissionsLoading: boolean;
  onConnect: () => void;
  /** Show private GitHub App and PAT setup. Factory onboarding owns the public App flow. */
  allowPrivateApp?: boolean;
}

export function GitHubConnectControls({
  definition,
  canCreateIntegrations,
  permissionsLoading,
  onConnect,
  allowPrivateApp = true,
}: GitHubConnectControlsProps) {
  if (!allowPrivateApp) return null;

  const canCreate = Boolean(definition) && canCreateIntegrations;

  return (
    <div className="flex shrink-0">
      <PermissionTooltip
        allowed={Boolean(definition) && (canCreateIntegrations || permissionsLoading)}
        message={
          definition
            ? "You don't have permission to connect integrations."
            : "This integration provider is no longer available for new connections."
        }
      >
        <Button
          variant="default"
          size="sm"
          onClick={onConnect}
          className="self-start"
          disabled={!canCreate}
          data-testid="integrations-connect-github"
        >
          {definition ? CREATE_PRIVATE_GITHUB_APP_LABEL : "Unavailable"}
        </Button>
      </PermissionTooltip>
    </div>
  );
}
