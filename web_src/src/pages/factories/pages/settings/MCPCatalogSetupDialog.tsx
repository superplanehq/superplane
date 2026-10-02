import { ExternalLink } from "lucide-react";
import { useEffect, useState } from "react";

import type { FactoriesFactoryAgentResource } from "@/api-client";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";

import { AGENT_RESOURCES_COPY, type AgentResourceInstruction } from "./agentResourceCopy";
import { DATADOG_MCP_DEFAULT_SITE_ID, DATADOG_MCP_SITES, datadogMCPSiteForURL, datadogMCPURL } from "./datadogMcpSites";
import type { MCPCatalogEntry } from "./mcpCatalog";

export function MCPCatalogSetupDialog({
  open,
  entry,
  resource,
  isSaving,
  onClose,
  onSignIn,
  onSaveToken,
}: {
  open: boolean;
  entry?: MCPCatalogEntry;
  resource?: FactoriesFactoryAgentResource;
  isSaving: boolean;
  onClose: () => void;
  onSignIn: (entry: MCPCatalogEntry) => Promise<void>;
  onSaveToken: (entry: MCPCatalogEntry, token: string) => Promise<void>;
}) {
  const [token, setToken] = useState("");
  const [tokenError, setTokenError] = useState("");
  const [siteId, setSiteId] = useState(DATADOG_MCP_DEFAULT_SITE_ID);

  useEffect(() => {
    if (!open) {
      return;
    }
    setToken("");
    setTokenError("");
    setSiteId(datadogMCPSiteForURL(resource?.url)?.id ?? DATADOG_MCP_DEFAULT_SITE_ID);
  }, [open, entry?.id, resource?.url]);

  if (!entry) {
    return null;
  }

  const isHeaderAuth = entry.auth === "AUTH_HEADERS";
  const isDatadog = entry.id === "datadog";
  const title = resource ? entry.label : AGENT_RESOURCES_COPY.addServerTitle(entry.label);
  const signInEntry = isDatadog ? { ...entry, url: datadogMCPURL(siteId) } : entry;

  const handleSignIn = async () => {
    await onSignIn(signInEntry);
  };

  const handleSaveToken = async () => {
    const trimmed = token.trim();
    if (!trimmed) {
      setTokenError(AGENT_RESOURCES_COPY.tokenRequired);
      return;
    }
    setTokenError("");
    await onSaveToken(entry, trimmed);
  };

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && !isSaving && onClose()}>
      <DialogContent data-testid="mcp-catalog-setup-dialog">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <span data-testid="mcp-catalog-setup-icon">
              <IntegrationIcon integrationName={entry.icon} iconSlug={entry.icon} className="size-5" />
            </span>
            {title}
          </DialogTitle>
          <DialogDescription data-testid="mcp-catalog-setup-instruction">
            <CatalogInstruction instruction={entry.instruction} />
          </DialogDescription>
        </DialogHeader>
        {isHeaderAuth ? (
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="mcp-catalog-setup-token">{AGENT_RESOURCES_COPY.tokenLabel}</Label>
            <Input
              id="mcp-catalog-setup-token"
              type="password"
              value={token}
              onChange={(event) => {
                setToken(event.target.value);
                setTokenError("");
              }}
              autoComplete="off"
              className="ph-no-capture"
              data-testid="mcp-catalog-setup-token"
              autoFocus
            />
            {tokenError ? <p className="text-[12px] text-destructive">{tokenError}</p> : null}
          </div>
        ) : null}
        {isDatadog ? (
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="mcp-catalog-setup-site">{AGENT_RESOURCES_COPY.datadogSiteLabel}</Label>
            <Select value={siteId} onValueChange={setSiteId}>
              <SelectTrigger id="mcp-catalog-setup-site" data-testid="mcp-catalog-setup-site" className="w-full">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {DATADOG_MCP_SITES.map((site) => (
                  <SelectItem key={site.id} value={site.id}>
                    {site.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <p className="text-[12px] text-muted-foreground">{AGENT_RESOURCES_COPY.datadogSiteHelper}</p>
          </div>
        ) : null}
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose} disabled={isSaving}>
            {AGENT_RESOURCES_COPY.cancel}
          </Button>
          {isHeaderAuth ? (
            <LoadingButton
              type="button"
              onClick={() => void handleSaveToken()}
              loading={isSaving}
              data-testid="mcp-catalog-setup-save"
            >
              {AGENT_RESOURCES_COPY.addConnection}
            </LoadingButton>
          ) : (
            <LoadingButton
              type="button"
              onClick={() => void handleSignIn()}
              loading={isSaving}
              data-testid="mcp-catalog-setup-sign-in"
            >
              {AGENT_RESOURCES_COPY.signIn}
              <ExternalLink className="size-3.5" aria-hidden />
            </LoadingButton>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function CatalogInstruction({ instruction }: { instruction: AgentResourceInstruction }) {
  if (typeof instruction === "string") {
    return instruction;
  }
  return (
    <>
      {instruction.before}
      <a
        href={instruction.href}
        target="_blank"
        rel="external noopener noreferrer"
        className="inline-flex items-center gap-0.5 text-foreground underline underline-offset-2"
      >
        {instruction.label}
        <ExternalLink className="size-3.5" aria-hidden />
      </a>
      {instruction.after}
    </>
  );
}
