import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import type { IntegrationsIntegrationDefinition } from "@/api-client/types.gen";
import { CREATE_PRIVATE_GITHUB_APP_LABEL, startPrivateGitHubAppSetup } from "@/lib/privateGitHubApp";
import { useNavigate } from "react-router";

interface GitHubConnectControlsProps {
  organizationId: string;
  definition: IntegrationsIntegrationDefinition | undefined;
  canCreateIntegrations: boolean;
  permissionsLoading: boolean;
  onConnect: () => void;
}

export function GitHubConnectControls({
  organizationId,
  definition,
  canCreateIntegrations,
  permissionsLoading,
  onConnect,
}: GitHubConnectControlsProps) {
  const navigate = useNavigate();
  const canCreate = Boolean(definition) && canCreateIntegrations;

  return (
    <div className="flex shrink-0 flex-col items-end gap-2">
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
          {definition ? "Connect" : "Unavailable"}
        </Button>
      </PermissionTooltip>
      {definition && canCreateIntegrations ? (
        <button
          type="button"
          className="text-xs text-gray-500 hover:underline dark:text-gray-400"
          data-testid="integrations-create-private-github-app"
          onClick={() =>
            startPrivateGitHubAppSetup({
              organizationId,
              returnTo: `/${organizationId}/settings/integrations`,
              goTo: navigate,
            })
          }
        >
          {CREATE_PRIVATE_GITHUB_APP_LABEL}
        </button>
      ) : null}
    </div>
  );
}
