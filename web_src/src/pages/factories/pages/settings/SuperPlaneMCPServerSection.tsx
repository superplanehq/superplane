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
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty";
import { Input } from "@/components/ui/input";
import { CopyButton } from "@/ui/CopyButton";
import { Search } from "lucide-react";
import { useMemo, useState } from "react";

import { FactoryDeleteDialog } from "../../FactoryDeleteDialog";
import { FactorySettingsCard } from "./FactorySettingsCard";
import { SuperPlaneMCPClientsEmptyIllustration } from "./SuperPlaneMCPClientsEmptyIllustration";
import { SUPERPLANE_MCP_SERVER_COPY } from "./superplaneMCPServerCopy";

const MCP_CLIENT_TOOLS: WorkspaceMCPClientTool[] = ["cursor", "claudeCode", "vscode"];

export function SuperPlaneMCPServerSection({
  organizationId,
  factoryId,
  clients,
  isLoading,
  isError,
  canUpdate,
  cardTitle = SUPERPLANE_MCP_SERVER_COPY.title,
  showCardDescription = true,
  connectAction = "card",
  connectDialogOpen: connectDialogOpenProp,
  onConnectDialogOpenChange,
}: {
  organizationId: string;
  factoryId: string;
  clients: FactoriesFactoryMcpClient[];
  isLoading: boolean;
  isError: boolean;
  canUpdate: boolean;
  /** Omit on a dedicated Connect settings page that already sets the page title. */
  cardTitle?: string | null;
  showCardDescription?: boolean;
  /** Put Connect in the page header with `connectAction="none"` and controlled dialog state. */
  connectAction?: "card" | "none";
  connectDialogOpen?: boolean;
  onConnectDialogOpenChange?: (open: boolean) => void;
}) {
  const [pendingRevoke, setPendingRevoke] = useState<FactoriesFactoryMcpClient | undefined>();
  const [connectOpenInternal, setConnectOpenInternal] = useState(false);
  const connectOpen = connectDialogOpenProp ?? connectOpenInternal;
  const setConnectOpen = onConnectDialogOpenChange ?? setConnectOpenInternal;
  const revokeClient = useRevokeFactoryMCPClient(organizationId, factoryId);
  const pendingName = mcpClientDisplayName(pendingRevoke);
  const connectClientAction =
    connectAction === "card" ? (
      <Button type="button" size="sm" onClick={() => setConnectOpen(true)} data-testid="superplane-mcp-connect-client">
        {SUPERPLANE_MCP_SERVER_COPY.connectClient}
      </Button>
    ) : undefined;

  return (
    <>
      <FactorySettingsCard
        title={cardTitle === null ? undefined : (cardTitle ?? SUPERPLANE_MCP_SERVER_COPY.title)}
        description={showCardDescription ? SUPERPLANE_MCP_SERVER_COPY.sectionDescription : undefined}
        action={connectClientAction}
        attachedList={clients.length > 0}
        data-testid="superplane-mcp-server"
      >
        {isLoading ? (
          <p className="text-[13px] text-muted-foreground">{SUPERPLANE_MCP_SERVER_COPY.loading}</p>
        ) : isError ? (
          <p className="text-[13px] text-destructive">{SUPERPLANE_MCP_SERVER_COPY.loadError}</p>
        ) : clients.length === 0 ? (
          <SuperPlaneMCPServerEmptyState />
        ) : (
          <SuperPlaneMCPClientsTable
            clients={clients}
            canUpdate={canUpdate}
            pendingRevokeId={pendingRevoke?.id}
            isRevoking={revokeClient.isPending}
            onRevoke={setPendingRevoke}
          />
        )}
      </FactorySettingsCard>
      <Dialog open={connectOpen} onOpenChange={(open) => setConnectOpen(open)}>
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

function SuperPlaneMCPClientsTable({
  clients,
  canUpdate,
  pendingRevokeId,
  isRevoking,
  onRevoke,
}: {
  clients: FactoriesFactoryMcpClient[];
  canUpdate: boolean;
  pendingRevokeId?: string;
  isRevoking: boolean;
  onRevoke: (client: FactoriesFactoryMcpClient) => void;
}) {
  const [searchQuery, setSearchQuery] = useState("");
  const filteredClients = useMemo(() => filterMcpClients(clients, searchQuery), [clients, searchQuery]);

  return (
    <div className="space-y-3">
      <div className="relative">
        <Search
          className="pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-muted-foreground"
          aria-hidden
        />
        <Input
          value={searchQuery}
          onChange={(event) => setSearchQuery(event.target.value)}
          placeholder={SUPERPLANE_MCP_SERVER_COPY.clientsSearchPlaceholder}
          className="h-8 pl-8 text-[13px]"
          data-testid="superplane-mcp-clients-search"
        />
      </div>
      {filteredClients.length === 0 ? (
        <p
          className="py-8 text-center text-[13px] text-muted-foreground"
          data-testid="superplane-mcp-clients-search-empty"
        >
          {SUPERPLANE_MCP_SERVER_COPY.clientsSearchEmpty}
        </p>
      ) : (
        <table className="w-full text-left" data-testid="superplane-mcp-clients-list">
          <tbody>
            {filteredClients.map((client) => (
              <SuperPlaneMCPClientRow
                key={client.id}
                client={client}
                canUpdate={canUpdate}
                isRevoking={isRevoking && pendingRevokeId === client.id}
                onRevoke={() => onRevoke(client)}
              />
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}

function filterMcpClients(clients: FactoriesFactoryMcpClient[], query: string): FactoriesFactoryMcpClient[] {
  const normalized = query.trim().toLowerCase();
  if (!normalized) {
    return clients;
  }
  return clients.filter((client) => mcpClientSearchText(client).includes(normalized));
}

function mcpClientSearchText(client: FactoriesFactoryMcpClient): string {
  return [mcpClientUserName(client), client.userEmail?.trim(), mcpClientDisplayName(client)]
    .filter(Boolean)
    .join(" ")
    .toLowerCase();
}

function SuperPlaneMCPServerEmptyState() {
  return (
    <Empty className="flex-none gap-3 border-none p-6 md:p-8" data-testid="superplane-mcp-clients-empty">
      <EmptyHeader className="max-w-md gap-3">
        <EmptyMedia variant="default" className="mb-0">
          <SuperPlaneMCPClientsEmptyIllustration />
        </EmptyMedia>
        <EmptyTitle className="text-[15px] font-medium">{SUPERPLANE_MCP_SERVER_COPY.emptyTitle}</EmptyTitle>
        <EmptyDescription className="text-[13px]">{SUPERPLANE_MCP_SERVER_COPY.emptyDescription}</EmptyDescription>
      </EmptyHeader>
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
  const clientName = mcpClientDisplayName(client);
  const userName = mcpClientUserName(client);
  const when = formatRelativeTime(client.createdAt);
  const absoluteWhen = client.createdAt ? formatTimestampInUserTimezone(client.createdAt) : undefined;

  return (
    <tr className="border-b border-border last:border-b-0" data-testid={`superplane-mcp-client-${client.id}`}>
      <td className="max-w-[12rem] py-2.5 pr-3">
        <div className="flex min-w-0 items-center gap-2">
          <Avatar
            src={client.userAvatarUrl?.trim() || undefined}
            initials={getUserInitials(userName)}
            alt={userName}
            className="size-6 shrink-0 bg-muted text-[10px] text-muted-foreground"
          />
          <span className="truncate text-[13px] font-medium text-foreground">{userName}</span>
        </div>
      </td>
      <td className="py-2.5 pr-3 text-[13px] text-foreground">{clientName}</td>
      <td className="whitespace-nowrap py-2.5 pr-3 text-[13px] text-muted-foreground" title={absoluteWhen}>
        {when}
      </td>
      <td className="py-2.5 text-right">
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
      </td>
    </tr>
  );
}

function mcpClientDisplayName(client?: FactoriesFactoryMcpClient) {
  const name = client?.clientName?.trim();
  return name || SUPERPLANE_MCP_SERVER_COPY.unnamedClient;
}

function mcpClientUserName(client: FactoriesFactoryMcpClient) {
  return client.userName?.trim() || client.userEmail?.trim() || SUPERPLANE_MCP_SERVER_COPY.unnamedUser;
}
