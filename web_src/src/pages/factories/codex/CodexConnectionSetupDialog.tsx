import { useEffect, useRef, useState } from "react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { usePermissions } from "@/contexts/usePermissions";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { FEATURE_CODEX_SUBSCRIPTIONS } from "@/lib/experimentalFeatures";
import { getApiErrorMessage } from "@/lib/errors";
import { codexSubscriptionReady, useFactoryCodexConnection } from "../pages/onboarding/useFactoryCodexConnection";
import { CodexDeviceSignIn } from "./CodexDeviceSignIn";

type CodexConnection = ReturnType<typeof useFactoryCodexConnection>;
type ConnectedAction = (() => void | Promise<void>) | undefined;

function useConnectionCompletion(open: boolean, codex: CodexConnection, onConnected: ConnectedAction) {
  const [error, setError] = useState<string | null>(null);
  const started = useRef(false);
  const version = useRef(0);
  const openRef = useRef(open);
  openRef.current = open;
  const connectedCallback = useRef(onConnected);
  connectedCallback.current = onConnected;

  useEffect(() => {
    if (!open) {
      version.current += 1;
      started.current = false;
      setError(null);
      return;
    }
    if (!codexSubscriptionReady(codex.query.data) || !codex.query.data?.availableModels?.length || started.current)
      return;
    started.current = true;
    const currentVersion = version.current;
    void codex
      .refresh()
      .then(() => {
        if (openRef.current && version.current === currentVersion) return connectedCallback.current?.();
      })
      .catch((completionError: unknown) => {
        if (!openRef.current || version.current !== currentVersion) return;
        setError(getApiErrorMessage(completionError, "Could not refresh your Codex connection. Try again."));
      });
  }, [open, codex]);

  return {
    error,
    cancel: () => {
      openRef.current = false;
      version.current += 1;
    },
    retry: () => {
      started.current = false;
      setError(null);
    },
  };
}

function CodexConnectionNotices({
  codex,
  featuresLoading,
  offered,
}: {
  codex: CodexConnection;
  featuresLoading: boolean;
  offered: boolean;
}) {
  const connection = codex.query.data;
  return (
    <>
      {featuresLoading || codex.query.isLoading ? <p role="status">Loading Codex connection...</p> : null}
      {codex.query.isError ? (
        <p role="alert">Could not load your Codex connection. Close this dialog and try again.</p>
      ) : null}
      {connection?.revocationPending ? (
        <p role="status">Stopping your previous tasks. You can reconnect when cleanup finishes.</p>
      ) : null}
      {!featuresLoading && !offered ? <p role="alert">Codex subscriptions are unavailable in this workspace.</p> : null}
      {connection?.eligible === false ? <p role="alert">Your account cannot connect Codex in this workspace.</p> : null}
      {codex.start.isError ? <p role="alert">Could not start Codex sign-in. Try again.</p> : null}
      {codex.cancel.isError ? <p role="alert">Could not cancel Codex sign-in. Try again.</p> : null}
      {codex.refreshCatalog.isError ? (
        <p role="alert">Could not load your Codex models. Close this dialog and try again.</p>
      ) : null}
    </>
  );
}

export function CodexConnectionSetupDialog({
  organizationId,
  factoryId,
  open,
  onOpenChange,
  onConnected,
}: {
  organizationId: string;
  factoryId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConnected?: () => void | Promise<void>;
}) {
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const features = useExperimentalFeature(organizationId);
  const offered = features.has(FEATURE_CODEX_SUBSCRIPTIONS);
  const enabled = [open, !features.isLoading, offered, canAct("factories", "read")].every(Boolean);
  const codex = useFactoryCodexConnection(organizationId, factoryId, enabled);
  const completion = useConnectionCompletion(enabled && codex.query.data?.eligible === true, codex, onConnected);
  const disabled = [
    !enabled,
    permissionsLoading,
    codex.query.isLoading,
    codex.query.data?.revocationPending,
    codex.query.data?.eligible !== true,
  ].some(Boolean);
  const close = () => {
    completion.cancel();
    if (codex.query.data?.status === "pending") codex.cancel.mutate();
    onOpenChange(false);
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => {
        if (!nextOpen) close();
      }}
    >
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Connect Codex</DialogTitle>
          <DialogDescription>
            Sign in with your Codex account to use your subscription in this workspace.
          </DialogDescription>
        </DialogHeader>
        <CodexConnectionNotices codex={codex} featuresLoading={features.isLoading} offered={offered} />
        <CodexDeviceSignIn
          selected
          disabled={disabled}
          connection={codex.query.data}
          starting={codex.start.isPending}
          canceling={codex.cancel.isPending}
          onSelect={() => {}}
          onStart={() => codex.start.mutate()}
          onCancel={() => codex.cancel.mutate()}
        />
        {completion.error ? (
          <div>
            <p role="alert">{completion.error}</p>
            <Button type="button" onClick={completion.retry}>
              Try again
            </Button>
          </div>
        ) : null}
        <DialogFooter>
          <Button type="button" variant="outline" onClick={close}>
            Cancel
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
