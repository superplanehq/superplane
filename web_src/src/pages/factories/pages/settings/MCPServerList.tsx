import type { FactoriesFactoryAgentResource } from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { factoryAgentResourcesKey, useFactoryAgentResourceTools } from "@/hooks/useFactoryAgentResources";
import { cn } from "@/lib/utils";
import { useQueryClient } from "@tanstack/react-query";
import { Plug, Settings2 } from "lucide-react";
import { useEffect } from "react";
import { Link } from "react-router";

import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";

import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";
import { connectionIsEstablished, connectionStatusLabel } from "./agentResourceDisplay";
import { mcpConnectionCatalogEntry, mcpConnectionDisplayName } from "./mcpConnectionDisplay";
import { enabledToolCount, mcpToolItems, workspaceDisabledTools } from "./mcpTools";

export function MCPServerList({
  organizationId,
  factoryId,
  resources,
  canUpdate,
  configurePath,
}: {
  organizationId: string;
  factoryId: string;
  resources: FactoriesFactoryAgentResource[];
  canUpdate: boolean;
  configurePath: (resourceId: string) => string;
}) {
  return (
    <ul className="divide-y divide-border border-t border-border" data-testid="agent-resources-connections-list">
      {resources.map((resource) => (
        <MCPServerRow
          key={resource.id || resource.name}
          organizationId={organizationId}
          factoryId={factoryId}
          resource={resource}
          canUpdate={canUpdate}
          configureHref={resource.id ? configurePath(resource.id) : "#"}
        />
      ))}
    </ul>
  );
}

function MCPServerRow({
  organizationId,
  factoryId,
  resource,
  canUpdate,
  configureHref,
}: {
  organizationId: string;
  factoryId: string;
  resource: FactoriesFactoryAgentResource;
  canUpdate: boolean;
  configureHref: string;
}) {
  const name = mcpConnectionDisplayName(resource);
  const connected = connectionIsEstablished(resource);
  const reconnect =
    resource.oauthStatus === "OAUTH_STATUS_NEEDS_RECONNECT" || resource.oauthStatus === "OAUTH_STATUS_VENDOR_REJECTED";
  const rowTitle = resource.oauthError?.trim() || undefined;
  const toolsLine = useMCPServerRowToolsLine(organizationId, factoryId, resource, connected);

  return (
    <li
      className="flex min-w-0 items-center gap-4 py-2.5"
      data-testid={`agent-resource-row-${resource.id}`}
      title={rowTitle}
    >
      <Link
        to={configureHref}
        className="flex min-w-0 flex-1 items-center gap-2.5 overflow-hidden text-left"
        data-testid={`agent-resource-edit-${resource.id}`}
        title={rowTitle}
      >
        <MCPConnectionRowIcon resource={resource} />
        <span className="min-w-0 truncate text-[13px] font-medium text-foreground">{name}</span>
      </Link>
      <div className="flex shrink-0 items-center gap-2" data-testid={`agent-resource-actions-${resource.id}`}>
        <div className="flex items-center gap-1.5">
          {toolsLine ? (
            <>
              <span
                className="whitespace-nowrap text-[12px] tabular-nums text-muted-foreground"
                data-testid={`agent-resource-tools-${resource.id}`}
              >
                {toolsLine}
              </span>
              <span className="text-[12px] text-muted-foreground" aria-hidden>
                ·
              </span>
            </>
          ) : null}
          <MCPServerConnectionStatus resource={resource} connected={connected} reconnect={reconnect} />
        </div>
        <PermissionTooltip allowed={canUpdate} message={AGENT_RESOURCES_COPY.noUpdatePermission}>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="size-8 shrink-0 text-muted-foreground"
            asChild
            disabled={!canUpdate}
          >
            <Link
              to={configureHref}
              aria-label={AGENT_RESOURCES_COPY.configure}
              data-testid={`agent-resource-configure-${resource.id}`}
            >
              <Settings2 className="size-4" strokeWidth={1.75} aria-hidden />
            </Link>
          </Button>
        </PermissionTooltip>
      </div>
    </li>
  );
}

function MCPServerConnectionStatus({
  resource,
  connected,
  reconnect,
}: {
  resource: FactoriesFactoryAgentResource;
  connected: boolean;
  reconnect: boolean;
}) {
  const statusLabel = connectionStatusLabel(resource);

  return (
    <span
      className={cn(
        "shrink-0 whitespace-nowrap text-[12px]",
        connected && "text-emerald-600 dark:text-emerald-500",
        reconnect && "text-amber-600 dark:text-amber-500",
        !connected && !reconnect && "text-muted-foreground",
      )}
      data-testid={`agent-resource-status-${resource.id}`}
    >
      {statusLabel}
    </span>
  );
}

function useMCPServerRowToolsLine(
  organizationId: string,
  factoryId: string,
  resource: FactoriesFactoryAgentResource,
  connected: boolean,
): string {
  const resourceId = resource.id ?? "";
  const workspaceOff = resource.enabled === false;
  const showTools = connected && !workspaceOff && Boolean(resourceId);
  const toolsQuery = useFactoryAgentResourceTools(organizationId, factoryId, resourceId, showTools);
  const queryClient = useQueryClient();
  useEffect(() => {
    if (!showTools || !toolsQuery.isSuccess) {
      return;
    }
    void queryClient.invalidateQueries({
      queryKey: factoryAgentResourcesKey(organizationId, factoryId, "KIND_MCP_SERVER"),
      exact: true,
    });
  }, [factoryId, organizationId, queryClient, showTools, toolsQuery.dataUpdatedAt, toolsQuery.isSuccess]);
  const tools = mcpToolItems(toolsQuery.data);
  const disabled = workspaceDisabledTools(resource);

  if (!showTools || toolsQuery.isLoading || toolsQuery.isError || tools.length === 0) {
    return "";
  }
  return AGENT_RESOURCES_COPY.connectedToolsSummary(enabledToolCount(tools, disabled), tools.length);
}

function MCPConnectionRowIcon({ resource }: { resource: FactoriesFactoryAgentResource }) {
  const catalogEntry = mcpConnectionCatalogEntry(resource);
  if (catalogEntry) {
    return (
      <span className="inline-flex size-4 shrink-0 items-center justify-center" aria-hidden>
        <IntegrationIcon
          integrationName={catalogEntry.icon}
          iconSlug={catalogEntry.icon}
          className="size-4"
          size={16}
        />
      </span>
    );
  }

  return <Plug className="size-4 shrink-0 text-muted-foreground" aria-hidden />;
}
