import type { FactoriesFactoryMcpClient } from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { useRevokeFactoryMCPClient } from "@/hooks/useFactoryMCPClients";
import { getApiErrorMessage } from "@/lib/errors";
import { formatRelativeTime, formatTimestampInUserTimezone } from "@/lib/timezone";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { workspaceMCPClientConfig } from "@/lib/workspaceMCPClientConfig";
import { CopyButton } from "@/ui/CopyButton";
import { Cable } from "lucide-react";
import { useState } from "react";

import { FactoryDeleteDialog } from "../../FactoryDeleteDialog";
import { FactorySettingsCard } from "./FactorySettingsCard";
import { SUPERPLANE_MCP_SERVER_COPY } from "./superplaneMCPServerCopy";

export function SuperPlaneMCPServerSection({
  organizationId,
  factoryId,
  clients,
  isLoading,
  isError,
  canUpdate,
}: {
  organizationId: string;
  factoryId: string;
  clients: FactoriesFactoryMcpClient[];
  isLoading: boolean;
  isError: boolean;
  canUpdate: boolean;
}) {
  const [pendingRevoke, setPendingRevoke] = useState<FactoriesFactoryMcpClient | undefined>();
  const revokeClient = useRevokeFactoryMCPClient(organizationId, factoryId);
  const pendingName = mcpClientDisplayName(pendingRevoke);

  return (
    <>
      <FactorySettingsCard title={SUPERPLANE_MCP_SERVER_COPY.title} data-testid="superplane-mcp-server">
        <p className="mb-3 text-[12px] text-muted-foreground">{SUPERPLANE_MCP_SERVER_COPY.helper}</p>
        {isLoading ? (
          <p className="text-[13px] text-muted-foreground">{SUPERPLANE_MCP_SERVER_COPY.loading}</p>
        ) : isError ? (
          <p className="text-[13px] text-destructive">{SUPERPLANE_MCP_SERVER_COPY.loadError}</p>
        ) : clients.length === 0 ? (
          <SuperPlaneMCPServerEmptyState />
        ) : (
          <ul className="divide-y divide-border" data-testid="superplane-mcp-clients-list">
            {clients.map((client) => (
              <SuperPlaneMCPClientRow
                key={client.id}
                client={client}
                canUpdate={canUpdate}
                isRevoking={revokeClient.isPending && pendingRevoke?.id === client.id}
                onRevoke={() => setPendingRevoke(client)}
              />
            ))}
          </ul>
        )}
      </FactorySettingsCard>
      <FactoryDeleteDialog
        open={Boolean(pendingRevoke)}
        factoryName={pendingName}
        title={SUPERPLANE_MCP_SERVER_COPY.revokeTitle(pendingName)}
        description={SUPERPLANE_MCP_SERVER_COPY.revokeDescription}
        confirmLabel={SUPERPLANE_MCP_SERVER_COPY.revoke}
        loadingText={SUPERPLANE_MCP_SERVER_COPY.revoking}
        canDelete={canUpdate}
        isDeleting={revokeClient.isPending}
        onClose={() => setPendingRevoke(undefined)}
        onConfirm={async () => {
          if (!pendingRevoke?.id) {
            return;
          }
          try {
            await revokeClient.mutateAsync(pendingRevoke.id);
            showSuccessToast(SUPERPLANE_MCP_SERVER_COPY.revoked);
          } catch (error) {
            showErrorToast(getApiErrorMessage(error) || SUPERPLANE_MCP_SERVER_COPY.revokeFailed);
            throw error;
          }
        }}
      />
    </>
  );
}

function SuperPlaneMCPServerEmptyState() {
  const config = workspaceMCPClientConfig(window.location.origin);
  return (
    <Empty className="border-0 p-6 md:p-10" data-testid="superplane-mcp-clients-empty">
      <EmptyHeader>
        <EmptyMedia variant="icon">
          <Cable />
        </EmptyMedia>
        <EmptyTitle>{SUPERPLANE_MCP_SERVER_COPY.emptyTitle}</EmptyTitle>
        <EmptyDescription>{SUPERPLANE_MCP_SERVER_COPY.emptyBody}</EmptyDescription>
      </EmptyHeader>
      <EmptyContent className="max-w-full">
        <div className="relative w-full overflow-hidden rounded-md border border-border bg-muted/40 text-left">
          <div className="absolute right-2 top-2 z-10">
            <CopyButton
              text={config}
              ariaLabel={SUPERPLANE_MCP_SERVER_COPY.copyConfig}
              copiedAriaLabel={SUPERPLANE_MCP_SERVER_COPY.copiedConfig}
              data-testid="superplane-mcp-config-copy"
            />
          </div>
          <pre className="overflow-x-auto px-4 py-3 pr-12 text-[12px] leading-relaxed text-foreground">{config}</pre>
        </div>
      </EmptyContent>
    </Empty>
  );
}

function SuperPlaneMCPClientRow({
  client,
  canUpdate,
  isRevoking,
  onRevoke,
}: {
  client: FactoriesFactoryMcpClient;
  canUpdate: boolean;
  isRevoking: boolean;
  onRevoke: () => void;
}) {
  const name = mcpClientDisplayName(client);
  const user = mcpClientUserName(client);
  const when = formatRelativeTime(client.createdAt);
  const absoluteWhen = client.createdAt ? formatTimestampInUserTimezone(client.createdAt) : undefined;

  return (
    <li className="flex items-center justify-between gap-4 py-3" data-testid={`superplane-mcp-client-${client.id}`}>
      <div className="min-w-0 flex-1">
        <p className="truncate text-[13px] font-medium text-foreground">{name}</p>
        <p className="truncate text-[12px] text-muted-foreground" title={absoluteWhen}>
          {SUPERPLANE_MCP_SERVER_COPY.connectedBy(user, when)}
        </p>
      </div>
      <PermissionTooltip allowed={canUpdate} message={SUPERPLANE_MCP_SERVER_COPY.noUpdatePermission}>
        <Button
          type="button"
          size="sm"
          variant="ghost"
          disabled={!canUpdate || isRevoking}
          onClick={onRevoke}
          data-testid={`superplane-mcp-client-revoke-${client.id}`}
        >
          {SUPERPLANE_MCP_SERVER_COPY.revoke}
        </Button>
      </PermissionTooltip>
    </li>
  );
}

function mcpClientDisplayName(client?: FactoriesFactoryMcpClient) {
  const name = client?.clientName?.trim();
  return name || SUPERPLANE_MCP_SERVER_COPY.unnamedClient;
}

function mcpClientUserName(client: FactoriesFactoryMcpClient) {
  return client.userName?.trim() || client.userEmail?.trim() || SUPERPLANE_MCP_SERVER_COPY.unnamedUser;
}
