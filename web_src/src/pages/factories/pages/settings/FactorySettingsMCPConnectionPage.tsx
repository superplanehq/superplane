import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import { useFactoryAgentResourceTools } from "@/hooks/useFactoryAgentResources";
import { usePageTitle } from "@/hooks/usePageTitle";
import { cn } from "@/lib/utils";

import { FactoryDeleteDialog } from "../../FactoryDeleteDialog";
import { AgentResourceConnectionDialog } from "./AgentResourceConnectionDialog";
import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";
import {
  connectionAuthLabel,
  connectionIsEstablished,
  connectionNeedsOAuthAction,
  connectionStatusLabel,
} from "./agentResourceDisplay";
import { FactorySettingsCard, FactorySettingsPageFrame } from "./FactorySettingsCard";
import { MCPCatalogSetupDialog } from "./MCPCatalogSetupDialog";
import { MCPToolsList } from "./MCPToolsList";
import { useMCPConnectionPage } from "./useMCPConnectionPage";
import { useMCPDisabledTools } from "./useMCPDisabledTools";
export function FactorySettingsMCPConnectionPage() {
  const page = useMCPConnectionPage();
  usePageTitle([page.displayName, AGENT_RESOURCES_COPY.mcpTitle, "Settings", page.factory.name ?? "Workspace"]);

  const backProps = {
    backHref: page.listPath,
    backLabel: AGENT_RESOURCES_COPY.backToMcpServers,
    backTestId: "mcp-connection-back",
  };

  if (!page.isLoading && !page.resource) {
    return (
      <FactorySettingsPageFrame title={AGENT_RESOURCES_COPY.mcpTitle} {...backProps}>
        <p className="text-[13px] text-destructive">{AGENT_RESOURCES_COPY.mcpConnectionNotFound}</p>
      </FactorySettingsPageFrame>
    );
  }

  if (!page.resource) {
    return (
      <FactorySettingsPageFrame title={AGENT_RESOURCES_COPY.mcpTitle} {...backProps}>
        <p className="text-[13px] text-muted-foreground">{AGENT_RESOURCES_COPY.loading}</p>
      </FactorySettingsPageFrame>
    );
  }

  return <MCPConnectionContent page={page} resource={page.resource} listPath={page.listPath} />;
}

function MCPConnectionContent({
  page,
  resource,
  listPath,
}: {
  page: ReturnType<typeof useMCPConnectionPage>;
  resource: NonNullable<ReturnType<typeof useMCPConnectionPage>["resource"]>;
  listPath: string;
}) {
  const backProps = {
    backHref: listPath,
    backLabel: AGENT_RESOURCES_COPY.backToMcpServers,
    backTestId: "mcp-connection-back",
  };
  const showTools = connectionIsEstablished(resource);
  const toolsQuery = useFactoryAgentResourceTools(page.organizationId, page.factoryId, resource.id ?? "", showTools);
  const { disabledTools, applyToolToggle } = useMCPDisabledTools(resource);
  const needsOAuth = connectionNeedsOAuthAction(resource);
  const reconnect =
    resource.oauthStatus === "OAUTH_STATUS_NEEDS_RECONNECT" || resource.oauthStatus === "OAUTH_STATUS_VENDOR_REJECTED";
  const connected = connectionIsEstablished(resource);
  const statusLabel = connectionStatusLabel(resource);
  const url = resource.url?.trim() ?? "";

  return (
    <>
      <FactorySettingsPageFrame
        {...backProps}
        title={page.displayName}
        subtitle={
          <span className="text-[12px] text-muted-foreground">
            <span
              className={cn(
                connected && "text-emerald-600 dark:text-emerald-500",
                reconnect && "text-amber-600 dark:text-amber-500",
              )}
            >
              {statusLabel}
            </span>
            {url ? <> · {url}</> : null}
          </span>
        }
        actions={
          <div className="flex items-center gap-3">
            <span className="text-[12px] text-muted-foreground">Enabled</span>
            <Switch
              checked={resource.enabled !== false}
              disabled={!page.canUpdate}
              onCheckedChange={page.toggleEnabled}
              aria-label={`Enable ${page.displayName}`}
              data-testid="mcp-connection-enabled"
            />
          </div>
        }
      >
        {needsOAuth ? (
          <FactorySettingsCard title={AGENT_RESOURCES_COPY.authSignIn}>
            <p className="mb-3 text-[12px] text-muted-foreground">
              {reconnect ? AGENT_RESOURCES_COPY.statusReconnect : AGENT_RESOURCES_COPY.statusNotConnected}
            </p>
            <PermissionTooltip allowed={page.canUpdate} message={AGENT_RESOURCES_COPY.noUpdatePermission}>
              <Button type="button" size="sm" onClick={page.startOAuthRedirect} disabled={!page.canUpdate}>
                {reconnect ? AGENT_RESOURCES_COPY.reconnect : AGENT_RESOURCES_COPY.connect}
              </Button>
            </PermissionTooltip>
          </FactorySettingsCard>
        ) : null}

        {showTools ? (
          <FactorySettingsCard title={AGENT_RESOURCES_COPY.mcpConnectionToolsTitle} data-testid="mcp-connection-tools">
            <MCPToolsList
              tools={toolsQuery.data ?? []}
              isLoading={toolsQuery.isLoading}
              isError={toolsQuery.isError}
              disabledTools={disabledTools}
              canUpdate={page.canUpdate && resource.enabled !== false}
              onToggleTool={(toolName, enabled) => page.toggleTools(applyToolToggle(toolName, enabled))}
            />
          </FactorySettingsCard>
        ) : null}

        <FactorySettingsCard
          title={AGENT_RESOURCES_COPY.mcpConnectionSettingsTitle}
          action={
            <PermissionTooltip allowed={page.canUpdate} message={AGENT_RESOURCES_COPY.noUpdatePermission}>
              <Button type="button" size="sm" variant="outline" onClick={page.openEdit} disabled={!page.canUpdate}>
                {AGENT_RESOURCES_COPY.edit}
              </Button>
            </PermissionTooltip>
          }
          data-testid="mcp-connection-settings"
        >
          <dl className="grid gap-2 text-[13px]">
            <div>
              <dt className="text-[11px] font-medium uppercase tracking-[0.04em] text-muted-foreground">URL</dt>
              <dd className="mt-0.5 break-all text-foreground">{url || "—"}</dd>
            </div>
            <div>
              <dt className="text-[11px] font-medium uppercase tracking-[0.04em] text-muted-foreground">Auth</dt>
              <dd className="mt-0.5 text-foreground">{connectionAuthLabel(resource.auth)}</dd>
            </div>
          </dl>
          {resource.auth === "AUTH_OAUTH" && resource.oauthStatus === "OAUTH_STATUS_CONNECTED" ? (
            <PermissionTooltip allowed={page.canUpdate} message={AGENT_RESOURCES_COPY.noUpdatePermission}>
              <Button
                type="button"
                size="sm"
                variant="outline"
                className="mt-4"
                onClick={page.disconnect}
                disabled={!page.canUpdate}
              >
                {AGENT_RESOURCES_COPY.disconnect}
              </Button>
            </PermissionTooltip>
          ) : null}
        </FactorySettingsCard>

        <FactorySettingsCard title="Danger zone">
          <PermissionTooltip allowed={page.canUpdate} message={AGENT_RESOURCES_COPY.noUpdatePermission}>
            <Button
              type="button"
              variant="outline"
              onClick={() => page.setPendingDelete(true)}
              disabled={!page.canUpdate}
              data-testid="mcp-connection-delete"
            >
              {AGENT_RESOURCES_COPY.delete}
            </Button>
          </PermissionTooltip>
        </FactorySettingsCard>
      </FactorySettingsPageFrame>

      <AgentResourceConnectionDialog
        open={page.connectionOpen}
        organizationId={page.organizationId}
        resource={resource}
        isSaving={page.isSaving}
        onClose={page.closeEdit}
        onSave={page.saveConnection}
      />
      <MCPCatalogSetupDialog
        open={page.catalogSetupOpen}
        entry={page.catalogEntry}
        resource={resource}
        isSaving={page.isSaving}
        onClose={page.closeCatalogSetup}
        onSignIn={page.startCatalogOAuth}
        onSaveToken={page.saveCatalogToken}
      />
      <FactoryDeleteDialog
        open={page.pendingDelete}
        factoryName={page.displayName}
        title={`Delete "${page.displayName}"?`}
        description={AGENT_RESOURCES_COPY.deleteDescription}
        canDelete={page.canUpdate}
        isDeleting={page.isDeleting}
        onClose={() => page.setPendingDelete(false)}
        onConfirm={page.confirmDelete}
      />
    </>
  );
}
