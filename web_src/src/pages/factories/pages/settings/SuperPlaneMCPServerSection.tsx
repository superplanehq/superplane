import type { FactoriesFactoryMcpClient } from "@/api-client";
import { Avatar } from "@/components/Avatar/avatar";
import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useRevokeFactoryMCPClient } from "@/hooks/useFactoryMCPClients";
import { getApiErrorMessage } from "@/lib/errors";
import { getUserInitials } from "@/lib/orgUserDisplay";
import { formatRelativeTime, formatTimestampInUserTimezone } from "@/lib/timezone";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import {
  type WorkspaceMCPClientTool,
  workspaceMCPClientSnippet,
  workspaceMCPServerURL,
} from "@/lib/workspaceMCPClientConfig";
import { CopyButton } from "@/ui/CopyButton";
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
  const [connectOpen, setConnectOpen] = useState(false);
  const revokeClient = useRevokeFactoryMCPClient(organizationId, factoryId);
  const pendingName = mcpClientDisplayName(pendingRevoke);
  const connectClientAction = (
    <Button type="button" size="sm" onClick={() => setConnectOpen(true)} data-testid="superplane-mcp-connect-client">
      {SUPERPLANE_MCP_SERVER_COPY.connectClient}
    </Button>
  );

  return (
    <>
      <FactorySettingsCard
        title={SUPERPLANE_MCP_SERVER_COPY.title}
        description={SUPERPLANE_MCP_SERVER_COPY.sectionDescription}
        action={connectClientAction}
        attachedList
        data-testid="superplane-mcp-server"
      >
        {isLoading ? (
          <p className="text-[13px] text-muted-foreground">{SUPERPLANE_MCP_SERVER_COPY.loading}</p>
        ) : isError ? (
          <p className="text-[13px] text-destructive">{SUPERPLANE_MCP_SERVER_COPY.loadError}</p>
        ) : clients.length === 0 ? (
          <SuperPlaneMCPServerEmptyState />
        ) : (
          <ul
            className="mx-4 divide-y divide-border border-t border-border"
            data-testid="superplane-mcp-clients-list"
          >
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
      <Dialog open={connectOpen} onOpenChange={setConnectOpen}>
        <DialogContent
          className="flex max-h-[min(42rem,85vh)] max-w-lg flex-col gap-3 overflow-y-auto sm:max-w-2xl"
          data-testid="superplane-mcp-connect-dialog"
        >
          <DialogHeader>
            <DialogTitle>{SUPERPLANE_MCP_SERVER_COPY.connectTitle}</DialogTitle>
          </DialogHeader>
          <SuperPlaneMCPClientSetup origin={window.location.origin} />
        </DialogContent>
      </Dialog>
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
    <div className="mx-4 border-t border-border py-5 text-center" data-testid="superplane-mcp-clients-empty">
      <p className="text-[13px] text-muted-foreground">{SUPERPLANE_MCP_SERVER_COPY.emptyMessage}</p>
    </div>
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
  const clientName = mcpClientDisplayName(client);
  const userName = mcpClientUserName(client);
  const when = formatRelativeTime(client.createdAt);
  const absoluteWhen = client.createdAt ? formatTimestampInUserTimezone(client.createdAt) : undefined;

  return (
    <li
      className="flex min-w-0 items-center gap-3 py-2"
      data-testid={`superplane-mcp-client-${client.id}`}
      title={absoluteWhen}
    >
      <div className="flex min-w-0 flex-1 items-center gap-2 overflow-hidden">
        <Avatar
          src={client.userAvatarUrl?.trim() || undefined}
          initials={getUserInitials(userName)}
          alt={userName}
          className="size-6 shrink-0 bg-muted text-[10px] text-muted-foreground"
        />
        <p className="min-w-0 truncate text-[13px] text-foreground">
          <span className="font-medium">{userName}</span>
          <span className="font-normal text-muted-foreground"> - </span>
          <span className="font-medium">{clientName}</span>
        </p>
      </div>
      <div className="flex shrink-0 items-center gap-1">
        <span className="whitespace-nowrap text-[12px] text-muted-foreground">{when}</span>
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
      </div>
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
