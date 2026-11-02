import { useEffect, useRef, useState } from "react";

import { usePermissions } from "@/contexts/usePermissions";
import { getApiErrorMessage } from "@/lib/errors";
import { codexSubscriptionReady, useFactoryCodexConnection } from "../pages/onboarding/useFactoryCodexConnection";

type CodexConnection = ReturnType<typeof useFactoryCodexConnection>;

type SetupOptions = {
  organizationId: string;
  factoryId: string;
  open: boolean;
  onConnected?: () => void | Promise<void>;
  onClose: () => void;
};

function useConnectionCompletion({
  open,
  enabled,
  codex,
  onConnected,
}: {
  open: boolean;
  enabled: boolean;
  codex: CodexConnection;
  onConnected: SetupOptions["onConnected"];
}) {
  const [error, setError] = useState<string | null>(null);
  const started = useRef(false);
  const canceled = useRef(false);
  const version = useRef(0);
  const openRef = useRef(open);
  openRef.current = open;
  const connectedCallback = useRef(onConnected);
  connectedCallback.current = onConnected;

  useEffect(() => {
    openRef.current = open;
    canceled.current = false;
    return () => {
      openRef.current = false;
      canceled.current = true;
      version.current += 1;
      started.current = false;
    };
  }, [open]);

  useEffect(() => {
    if (!open) {
      version.current += 1;
      started.current = false;
      setError(null);
      return;
    }
    if (
      !enabled ||
      canceled.current ||
      !codexSubscriptionReady(codex.query.data) ||
      !codex.query.data?.availableModels?.length ||
      started.current
    )
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
  }, [open, enabled, codex]);

  return {
    error,
    cancel: () => {
      canceled.current = true;
      openRef.current = false;
      version.current += 1;
    },
    retry: () => {
      canceled.current = false;
      started.current = false;
      setError(null);
    },
  };
}

export function useCodexConnectionSetup({ organizationId, factoryId, open, onConnected, onClose }: SetupOptions) {
  const { isLoading: permissionsLoading } = usePermissions();
  const enabled = open;
  const codex = useFactoryCodexConnection(organizationId, factoryId, enabled);
  const completion = useConnectionCompletion({
    open,
    enabled: enabled && codex.query.data?.eligible === true,
    codex,
    onConnected,
  });
  const [closing, setClosing] = useState(false);
  const closingRef = useRef(false);
  useEffect(() => {
    if (open) return;
    closingRef.current = false;
    setClosing(false);
  }, [open]);
  const disabled = [
    !enabled,
    permissionsLoading,
    codex.query.isLoading,
    codex.query.data?.revocationPending,
    codex.query.data?.eligible !== true,
    closing,
  ].some(Boolean);
  const cancelSetup = async () => {
    if (!open || closingRef.current || codex.start.isPending || codex.cancel.isPending) return false;
    completion.cancel();
    closingRef.current = true;
    setClosing(true);
    try {
      if (codex.query.data?.status === "pending") await codex.cancel.mutateAsync();
      return true;
    } catch {
      completion.retry();
      closingRef.current = false;
      setClosing(false);
      return false;
    }
  };

  return {
    codex,
    disabled,
    completionError: completion.error,
    retryCompletion: completion.retry,
    busy: closing || codex.start.isPending || codex.cancel.isPending,
    cancelSetup,
    closeSetup: async () => {
      if (await cancelSetup()) onClose();
    },
  };
}
