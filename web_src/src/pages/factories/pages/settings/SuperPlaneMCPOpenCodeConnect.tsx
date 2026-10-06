import { useState } from "react";

import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { useCreateFactoryMCPAPIToken } from "@/hooks/useFactoryMCPClients";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast } from "@/lib/toast";
import { workspaceMCPClientSnippet, workspaceMCPServerURL } from "@/lib/workspaceMCPClientConfig";
import { CopyButton } from "@/ui/CopyButton";

import { SUPERPLANE_MCP_SERVER_COPY } from "./superplaneMCPServerCopy";

export function OpenCodeTokenConnect({
  origin,
  organizationId,
  factoryId,
  canUpdate,
  onSecret,
}: {
  origin: string;
  organizationId: string;
  factoryId: string;
  canUpdate: boolean;
  onSecret: (secret: string) => void;
}) {
  const [name, setName] = useState("");
  const createToken = useCreateFactoryMCPAPIToken(organizationId, factoryId);
  const guide = SUPERPLANE_MCP_SERVER_COPY.tools.opencode;
  const snippet = workspaceMCPClientSnippet("opencode", origin);

  return (
    <>
      <ol className="list-decimal space-y-1 pl-4 text-left text-[12px] leading-relaxed text-muted-foreground">
        {guide.steps.map((step) => (
          <li key={step}>{step}</li>
        ))}
      </ol>
      <form
        className="space-y-2"
        onSubmit={(event) => {
          event.preventDefault();
          const tokenName = name.trim();
          if (!tokenName || !canUpdate) {
            return;
          }
          void createToken
            .mutateAsync({ name: tokenName, resource: workspaceMCPServerURL(origin) })
            .then((created) => {
              if (!created?.plaintext) {
                showErrorToast(SUPERPLANE_MCP_SERVER_COPY.createTokenFailed);
                return;
              }
              setName("");
              onSecret(created.plaintext);
            })
            .catch((error: unknown) => {
              showErrorToast(getApiErrorMessage(error) || SUPERPLANE_MCP_SERVER_COPY.createTokenFailed);
            });
        }}
      >
        <Label htmlFor="superplane-mcp-opencode-name">{SUPERPLANE_MCP_SERVER_COPY.tokenNameLabel}</Label>
        <div className="flex items-center gap-2">
          <Input
            id="superplane-mcp-opencode-name"
            value={name}
            onChange={(event) => setName(event.target.value)}
            disabled={!canUpdate || createToken.isPending}
            data-testid="superplane-mcp-opencode-name"
          />
          <PermissionTooltip allowed={canUpdate} message={SUPERPLANE_MCP_SERVER_COPY.noUpdatePermission}>
            <LoadingButton
              type="submit"
              size="sm"
              disabled={!canUpdate || !name.trim()}
              loading={createToken.isPending}
              loadingText={SUPERPLANE_MCP_SERVER_COPY.creatingToken}
              data-testid="superplane-mcp-opencode-create"
            >
              {SUPERPLANE_MCP_SERVER_COPY.createToken}
            </LoadingButton>
          </PermissionTooltip>
        </div>
      </form>
      <MCPConfigSnippet
        text={snippet}
        ariaLabel={SUPERPLANE_MCP_SERVER_COPY.copyConfig}
        copiedAriaLabel={SUPERPLANE_MCP_SERVER_COPY.copiedConfig}
        testId="superplane-mcp-config-copy-opencode"
      />
    </>
  );
}

export function MCPAPITokenSecretDialog({ secret, onClose }: { secret: string | null; onClose: () => void }) {
  return (
    <Dialog
      open={Boolean(secret)}
      onOpenChange={(open) => {
        if (!open) {
          onClose();
        }
      }}
    >
      <DialogContent data-testid="superplane-mcp-opencode-secret-dialog">
        <DialogHeader>
          <DialogTitle>{SUPERPLANE_MCP_SERVER_COPY.secretTitle}</DialogTitle>
        </DialogHeader>
        <p className="text-[13px] text-muted-foreground">{SUPERPLANE_MCP_SERVER_COPY.secretDescription}</p>
        <div className="ph-no-capture flex items-start gap-2 rounded-md border border-border bg-muted/40 p-3">
          <code
            className="flex-1 break-all font-mono text-xs leading-5 text-foreground"
            data-testid="superplane-mcp-opencode-secret"
          >
            {secret}
          </code>
          <CopyButton
            variant="button"
            text={secret || ""}
            ariaLabel={SUPERPLANE_MCP_SERVER_COPY.copySecret}
            copiedAriaLabel={SUPERPLANE_MCP_SERVER_COPY.copiedSecret}
            className="shrink-0"
            data-testid="superplane-mcp-opencode-secret-copy"
          >
            Copy
          </CopyButton>
        </div>
        <p className="text-xs text-muted-foreground">{SUPERPLANE_MCP_SERVER_COPY.secretHint}</p>
        <Button type="button" onClick={onClose} data-testid="superplane-mcp-opencode-secret-done">
          {SUPERPLANE_MCP_SERVER_COPY.secretDone}
        </Button>
      </DialogContent>
    </Dialog>
  );
}

export function MCPConfigSnippet({
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
