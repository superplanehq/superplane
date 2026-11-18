import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";

import type { useHostedGitHubConnect } from "./useHostedGitHubConnect";

type HostedConnect = ReturnType<typeof useHostedGitHubConnect>;

export function HostedGitHubConnectButton(props: {
  connect: HostedConnect;
  canCreateIntegrations: boolean;
  permissionsLoading: boolean;
}) {
  return (
    <PermissionTooltip
      allowed={props.canCreateIntegrations || props.permissionsLoading}
      message="You don't have permission to connect integrations."
    >
      <Button
        size="sm"
        className="self-start"
        disabled={!props.canCreateIntegrations || props.connect.opening}
        onClick={() => void props.connect.start()}
        data-testid="integrations-connect-github"
      >
        {props.connect.opening ? "Opening GitHub…" : "Connect"}
      </Button>
    </PermissionTooltip>
  );
}

export function HostedGitHubInstallations(props: { connect: HostedConnect; canCreateIntegrations: boolean }) {
  const { installations, connectInstallation } = props.connect;
  if (installations.length === 0) return null;
  return (
    <ul className="mt-3 divide-y divide-border rounded-lg border border-border" data-testid="github-installations">
      {installations.map((installation) => (
        <li key={installation.installationId} className="flex items-center justify-between gap-3 px-3 py-2">
          <span className="min-w-0 truncate text-[13px]">
            <span className="font-medium">{installation.accountLogin}</span>
            <span className="text-muted-foreground"> · Installed, not connected</span>
          </span>
          <Button
            size="sm"
            variant="outline"
            disabled={
              !props.canCreateIntegrations ||
              (connectInstallation.isPending &&
                connectInstallation.variables?.installationId === installation.installationId)
            }
            onClick={() => connectInstallation.mutate(installation)}
            data-testid={`github-installation-connect-${installation.accountLogin}`}
          >
            Connect
          </Button>
        </li>
      ))}
    </ul>
  );
}
