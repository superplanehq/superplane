import type { FactoriesFactoryCodexConnection } from "@/api-client";
import { LoadingButton } from "@/components/ui/loading-button";
import { usePermissions } from "@/contexts/usePermissions";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { FEATURE_CODEX_SUBSCRIPTIONS } from "@/lib/experimentalFeatures";
import { CodexDeviceSignIn } from "../../codex/CodexDeviceSignIn";
import type { useFactoryCodexConnection } from "../onboarding/useFactoryCodexConnection";
import { useFactorySettingsLayout } from "./factorySettingsLayoutContext";

type CodexConnection = ReturnType<typeof useFactoryCodexConnection>;

function useCodexConnectionState(codex: CodexConnection) {
  const { isLoading } = usePermissions();
  const { organizationId } = useFactorySettingsLayout();
  const features = useExperimentalFeature(organizationId);
  const connection = codex.query.data;
  const offered = [!features.isLoading, features.has(FEATURE_CODEX_SUBSCRIPTIONS), connection?.eligible === true].every(
    Boolean,
  );
  const disabled = [isLoading, codex.query.isLoading, connection?.revocationPending === true, !offered].some(Boolean);
  const ready = connection?.status === "ready";
  const signingIn =
    !ready &&
    (["pending", "finishing", "canceling"].includes(connection?.status ?? "") || connection?.reservationHeld === true);
  const refreshing = ready && (codex.refreshCatalog.isPending || connection?.reservationHeld === true);
  const reconnect = connection?.status === "reconnect_required";

  return {
    connection,
    disabled: disabled || refreshing,
    signingIn,
    ready,
    refreshing,
    reconnect,
    status: connectionStatus(features.isLoading || codex.query.isLoading, connection),
  };
}

function connectionStatus(loading: boolean, connection: FactoriesFactoryCodexConnection | undefined) {
  if (loading) return "Loading ChatGPT connection...";
  if (connection?.status === "ready") return "Connected";
  if (connection?.status === "reconnect_required") return "Reconnect required";
  return "Not connected";
}

export function FactoryCodexSubscriptionConnection({ codex }: { codex: CodexConnection }) {
  const { connection, disabled, signingIn, ready, refreshing, reconnect, status } = useCodexConnectionState(codex);

  if (signingIn)
    return (
      <CodexDeviceSignIn
        selected
        disabled={disabled}
        connection={connection}
        starting={codex.start.isPending}
        canceling={codex.cancel.isPending}
        onSelect={() => {}}
        onStart={() => codex.start.mutate()}
        onCancel={() => codex.cancel.mutate()}
      />
    );

  return (
    <div className="space-y-2">
      <div className="flex flex-col gap-3 rounded-md border border-border bg-muted/40 px-3 py-2 sm:flex-row sm:items-center sm:justify-between">
        <div className="min-w-0">
          <p className="break-words text-[13px] font-medium [overflow-wrap:anywhere]">
            {connection?.accountLabel || "Your ChatGPT subscription"}
          </p>
          <p role="status" className="mt-0.5 text-xs text-muted-foreground">
            {status}
          </p>
        </div>
        <div className="flex flex-wrap gap-2">
          {ready ? (
            <>
              <LoadingButton
                type="button"
                variant="outline"
                size="sm"
                loading={refreshing}
                aria-busy={refreshing}
                disabled={disabled}
                onClick={() => codex.refreshCatalog.mutate()}
              >
                Refresh usage and models
              </LoadingButton>
              <LoadingButton
                type="button"
                variant="outline"
                size="sm"
                loading={codex.disconnect.isPending}
                disabled={disabled}
                onClick={() => codex.disconnect.mutate()}
              >
                Disconnect ChatGPT
              </LoadingButton>
            </>
          ) : (
            <>
              <LoadingButton
                type="button"
                variant="outline"
                size="sm"
                loading={codex.start.isPending}
                disabled={disabled}
                onClick={() => codex.start.mutate()}
              >
                {reconnect ? "Reconnect ChatGPT" : "Connect ChatGPT"}
              </LoadingButton>
              {reconnect ? (
                <LoadingButton
                  type="button"
                  variant="outline"
                  size="sm"
                  loading={codex.disconnect.isPending}
                  disabled={disabled}
                  onClick={() => codex.disconnect.mutate()}
                >
                  Disconnect ChatGPT
                </LoadingButton>
              ) : null}
            </>
          )}
        </div>
      </div>
      {connection?.revocationPending ? (
        <p role="status" className="text-xs text-muted-foreground">
          Stopping your previous tasks. You can reconnect when cleanup finishes.
        </p>
      ) : null}
      {codex.query.isError ? (
        <p role="alert" className="text-xs text-muted-foreground">
          Unable to load your ChatGPT connection. Reload the page to try again.
        </p>
      ) : null}
    </div>
  );
}
