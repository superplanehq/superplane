import type { FactoriesFactoryMcpClient } from "@/api-client";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useRevokeFactoryMCPClient } from "@/hooks/useFactoryMCPClients";
import { getApiErrorMessage } from "@/lib/errors";
import { formatRelativeTime, formatTimestampInUserTimezone } from "@/lib/timezone";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import {
  type WorkspaceMCPClientTool,
  workspaceMCPClientSnippet,
  workspaceMCPServerURL,
} from "@/lib/workspaceMCPClientConfig";
import { CopyButton } from "@/ui/CopyButton";
import { Cable } from "lucide-react";
import { useState } from "react";

import { FactoryDeleteDialog } from "../../FactoryDeleteDialog";
import { FactorySettingsCard } from "./FactorySettingsCard";
import { SUPERPLANE_MCP_SERVER_COPY } from "./superplaneMCPServerCopy";

const MCP_CLIENT_TOOLS: WorkspaceMCPClientTool[] = ["cursor", "claudeCode", "vscode"];

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
          <div>
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
            <div className="mt-5 border-t border-border pt-5" data-testid="superplane-mcp-client-setup">
              <p className="mb-3 text-[13px] font-medium text-foreground">{SUPERPLANE_MCP_SERVER_COPY.connectTitle}</p>
              <SuperPlaneMCPClientSetup origin={window.location.origin} />
            </div>
          </div>
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
  return (
    <Empty className="border-0 p-6 md:p-10" data-testid="superplane-mcp-clients-empty">
      <EmptyHeader>
        <EmptyMedia variant="icon">
          <Cable />
        </EmptyMedia>
        <EmptyTitle>{SUPERPLANE_MCP_SERVER_COPY.emptyTitle}</EmptyTitle>
        <EmptyDescription>{SUPERPLANE_MCP_SERVER_COPY.emptyBody}</EmptyDescription>
      </EmptyHeader>
      <EmptyContent className="max-w-full items-stretch text-left">
        <SuperPlaneMCPClientSetup origin={window.location.origin} />
      </EmptyContent>
    </Empty>
  );
}

function SuperPlaneMCPClientSetup({ origin }: { origin: string }) {
  return (
    <div className="space-y-3">
      <MCPServerURLRow url={workspaceMCPServerURL(origin)} />
      <MCPClientSetupTabs origin={origin} />
    </div>
  );
}

function MCPServerURLRow({ url }: { url: string }) {
  return (
    <div>
      <p className="mb-1.5 text-[11px] font-medium uppercase tracking-[0.04em] text-muted-foreground">
        {SUPERPLANE_MCP_SERVER_COPY.serverUrlLabel}
      </p>
      <div className="flex items-center gap-2 rounded-md border border-border bg-muted/40 px-3 py-2">
        <code
          className="min-w-0 flex-1 truncate text-left text-[12px] text-foreground"
          data-testid="superplane-mcp-server-url"
        >
          {url}
        </code>
        <CopyButton
          text={url}
          ariaLabel={SUPERPLANE_MCP_SERVER_COPY.copyUrl}
          copiedAriaLabel={SUPERPLANE_MCP_SERVER_COPY.copiedUrl}
          data-testid="superplane-mcp-url-copy"
        />
      </div>
    </div>
  );
}

function MCPClientSetupTabs({ origin }: { origin: string }) {
  return (
    <Tabs defaultValue="cursor">
      <TabsList className="h-auto w-full flex-wrap justify-start" data-testid="superplane-mcp-client-tools">
        {MCP_CLIENT_TOOLS.map((tool) => (
          <TabsTrigger key={tool} value={tool} data-testid={`superplane-mcp-client-tool-${tool}`}>
            {SUPERPLANE_MCP_SERVER_COPY.tools[tool].label}
          </TabsTrigger>
        ))}
      </TabsList>
      {MCP_CLIENT_TOOLS.map((tool) => (
        <TabsContent key={tool} value={tool} className="mt-3 space-y-3">
          <MCPClientToolGuide tool={tool} origin={origin} />
        </TabsContent>
      ))}
    </Tabs>
  );
}

function MCPClientToolGuide({ tool, origin }: { tool: WorkspaceMCPClientTool; origin: string }) {
  const guide = SUPERPLANE_MCP_SERVER_COPY.tools[tool];
  const snippet = workspaceMCPClientSnippet(tool, origin);
  const isCommand = tool === "claudeCode";
  return (
    <>
      <ol className="list-decimal space-y-1 pl-4 text-left text-[12px] leading-relaxed text-muted-foreground">
        {guide.steps.map((step) => (
          <li key={step}>{step}</li>
        ))}
      </ol>
      <MCPConfigSnippet
        text={snippet}
        ariaLabel={isCommand ? SUPERPLANE_MCP_SERVER_COPY.copyCommand : SUPERPLANE_MCP_SERVER_COPY.copyConfig}
        copiedAriaLabel={isCommand ? SUPERPLANE_MCP_SERVER_COPY.copiedCommand : SUPERPLANE_MCP_SERVER_COPY.copiedConfig}
        testId={`superplane-mcp-config-copy-${tool}`}
      />
    </>
  );
}

function MCPConfigSnippet({
  text,
  ariaLabel,
  copiedAriaLabel,
  testId,
}: {
  text: string;
  ariaLabel: string;
  copiedAriaLabel: string;
  testId: string;
}) {
  return (
    <div className="relative w-full overflow-hidden rounded-md border border-border bg-muted/40 text-left">
      <div className="absolute right-2 top-2 z-10">
        <CopyButton text={text} ariaLabel={ariaLabel} copiedAriaLabel={copiedAriaLabel} data-testid={testId} />
      </div>
      <pre className="overflow-x-auto px-4 py-3 pr-12 text-[12px] leading-relaxed text-foreground">{text}</pre>
    </div>
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
