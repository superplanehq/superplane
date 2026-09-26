import type { FactoriesFactoryAgentResource } from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { useFactoryAgentResourceTools } from "@/hooks/useFactoryAgentResources";
import { cn } from "@/lib/utils";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { ChevronDown, MoreHorizontal } from "lucide-react";
import { useEffect, useRef, useState } from "react";

import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";
import {
  connectionAuthLabel,
  connectionIsEstablished,
  connectionNeedsOAuthAction,
  connectionStatusLabel,
} from "./agentResourceDisplay";
import { ConnectionStatusDot } from "./ConnectionStatusDot";
import { MCPToolsList } from "./MCPToolsList";
import { enabledToolCount, mcpToolItems, nextDisabledTools, workspaceDisabledTools } from "./mcpTools";

export function MCPServerList({
  organizationId,
  factoryId,
  resources,
  canUpdate,
  onEdit,
  onDelete,
  onDisconnect,
  onToggleEnabled,
  onToggleTools,
  onConnect,
}: {
  organizationId: string;
  factoryId: string;
  resources: FactoriesFactoryAgentResource[];
  canUpdate: boolean;
  onEdit: (resource: FactoriesFactoryAgentResource) => void;
  onDelete: (resource: FactoriesFactoryAgentResource) => void;
  onDisconnect: (resource: FactoriesFactoryAgentResource) => void;
  onToggleEnabled: (resource: FactoriesFactoryAgentResource, enabled: boolean) => void;
  onToggleTools: (resource: FactoriesFactoryAgentResource, disabledTools: string[]) => void;
  onConnect: (resource: FactoriesFactoryAgentResource) => void;
}) {
  return (
    <ul className="divide-y divide-border" data-testid="agent-resources-connections-list">
      {resources.map((resource) => (
        <MCPServerRow
          key={resource.id || resource.name}
          organizationId={organizationId}
          factoryId={factoryId}
          resource={resource}
          canUpdate={canUpdate}
          onEdit={() => onEdit(resource)}
          onDelete={() => onDelete(resource)}
          onDisconnect={() => onDisconnect(resource)}
          onToggleEnabled={(enabled) => onToggleEnabled(resource, enabled)}
          onToggleTools={(disabledTools) => onToggleTools(resource, disabledTools)}
          onConnect={() => onConnect(resource)}
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
  onEdit,
  onDelete,
  onDisconnect,
  onToggleEnabled,
  onToggleTools,
  onConnect,
}: {
  organizationId: string;
  factoryId: string;
  resource: FactoriesFactoryAgentResource;
  canUpdate: boolean;
  onEdit: () => void;
  onDelete: () => void;
  onDisconnect: () => void;
  onToggleEnabled: (enabled: boolean) => void;
  onToggleTools: (disabledTools: string[]) => void;
  onConnect: () => void;
}) {
  const [expanded, setExpanded] = useState(false);
  const name = resource.name?.trim() || AGENT_RESOURCES_COPY.unnamedResource;
  const status = connectionStatusLabel(resource);
  const needsOAuth = connectionNeedsOAuthAction(resource);
  const reconnect =
    resource.oauthStatus === "OAUTH_STATUS_NEEDS_RECONNECT" || resource.oauthStatus === "OAUTH_STATUS_VENDOR_REJECTED";
  const showTools = connectionIsEstablished(resource);
  const toolsQuery = useFactoryAgentResourceTools(organizationId, factoryId, resource.id ?? "", expanded && showTools);
  const { disabledTools, applyToolToggle } = usePendingDisabledTools(resource);
  const tools = mcpToolItems(toolsQuery.data);
  const countLabel =
    expanded && showTools && !toolsQuery.isLoading && !toolsQuery.isError && tools.length > 0
      ? AGENT_RESOURCES_COPY.toolsCount(enabledToolCount(tools, disabledTools), tools.length)
      : null;

  return (
    <li data-testid={`agent-resource-row-${resource.id}`}>
      <div className="flex flex-wrap items-center gap-3 py-3">
        <ConnectionStatusDot resource={resource} />
        <div className="min-w-0 flex-1">
          <button
            type="button"
            className="block w-full cursor-pointer truncate text-left disabled:cursor-not-allowed"
            onClick={onEdit}
            disabled={!canUpdate}
            aria-label={`${AGENT_RESOURCES_COPY.edit} ${name}`}
            data-testid={`agent-resource-edit-${resource.id}`}
          >
            <span className="block truncate text-[13px] font-medium text-foreground">{name}</span>
            <span className="block truncate text-[12px] text-muted-foreground">{resource.url}</span>
          </button>
          {resource.oauthError ? <p className="mt-1 text-[12px] text-destructive">{resource.oauthError}</p> : null}
        </div>
        {countLabel ? <span className="text-[12px] tabular-nums text-muted-foreground">{countLabel}</span> : null}
        <span className="text-[12px] text-muted-foreground">{connectionAuthLabel(resource.auth)}</span>
        <span className={cn("text-[12px]", reconnect ? "text-destructive" : "text-muted-foreground")}>{status}</span>
        <Switch
          checked={resource.enabled !== false}
          disabled={!canUpdate}
          onCheckedChange={onToggleEnabled}
          aria-label={`Enable ${name}`}
          data-testid={`agent-resource-enabled-${resource.id}`}
        />
        {showTools ? (
          <Button
            type="button"
            size="sm"
            variant="outline"
            onClick={() => setExpanded((open) => !open)}
            data-testid={`agent-resource-view-tools-${resource.id}`}
          >
            <ChevronDown className={cn("size-3.5 transition-transform", expanded && "rotate-180")} aria-hidden />
            {AGENT_RESOURCES_COPY.toolsToggle}
          </Button>
        ) : null}
        {needsOAuth ? (
          <PermissionTooltip allowed={canUpdate} message={AGENT_RESOURCES_COPY.noUpdatePermission}>
            <Button type="button" size="sm" variant="outline" onClick={onConnect} disabled={!canUpdate}>
              {reconnect ? AGENT_RESOURCES_COPY.reconnect : AGENT_RESOURCES_COPY.connect}
            </Button>
          </PermissionTooltip>
        ) : null}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              type="button"
              variant="ghost"
              size="icon-xs"
              className="text-muted-foreground"
              aria-label={`${name} menu`}
              data-testid={`agent-resource-menu-${resource.id}`}
            >
              <MoreHorizontal className="size-3.5" aria-hidden />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuItem disabled={!canUpdate} onClick={onEdit}>
              {AGENT_RESOURCES_COPY.edit}
            </DropdownMenuItem>
            {resource.auth === "AUTH_OAUTH" && resource.oauthStatus === "OAUTH_STATUS_CONNECTED" ? (
              <DropdownMenuItem disabled={!canUpdate} onClick={onDisconnect}>
                {AGENT_RESOURCES_COPY.disconnect}
              </DropdownMenuItem>
            ) : null}
            <DropdownMenuItem disabled={!canUpdate} className="text-destructive" onClick={onDelete}>
              {AGENT_RESOURCES_COPY.delete}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
      {expanded && showTools ? (
        <div className="pb-3 pl-5">
          <MCPToolsList
            tools={toolsQuery.data ?? []}
            isLoading={toolsQuery.isLoading}
            isError={toolsQuery.isError}
            disabledTools={disabledTools}
            canUpdate={canUpdate && resource.enabled !== false}
            onToggleTool={(toolName, enabled) => onToggleTools(applyToolToggle(toolName, enabled))}
          />
        </div>
      ) : null}
    </li>
  );
}

function usePendingDisabledTools(resource: FactoriesFactoryAgentResource) {
  const serverDisabled = workspaceDisabledTools(resource);
  const serverKey = serverDisabled.join("\0");
  const [pending, setPending] = useState<string[] | undefined>();
  const disabledRef = useRef(serverDisabled);

  useEffect(() => {
    setPending((current) => (current && current.join("\0") === serverKey ? undefined : current));
  }, [resource.id, serverKey]);

  const disabledTools = pending ?? serverDisabled;
  disabledRef.current = disabledTools;
  return {
    disabledTools,
    applyToolToggle: (toolName: string, enabled: boolean) => {
      const next = nextDisabledTools(disabledRef.current, toolName, enabled);
      disabledRef.current = next;
      setPending(next);
      return next;
    },
  };
}
