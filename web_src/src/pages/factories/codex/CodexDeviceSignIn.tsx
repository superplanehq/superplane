import { LoadingButton } from "@/components/ui/loading-button";
import { showErrorToast } from "@/lib/toast";
import { CopyButton } from "@/ui/CopyButton";
import type { FactoriesFactoryCodexConnection } from "@/api-client";
import { useEffect, useState, type ReactNode } from "react";
import { ExternalLink, Loader2 } from "lucide-react";
import { Button } from "@/components/ui/button";

import { CODEX_CONNECTION_COPY } from "./codexConnectionCopy";

type CodexPhase = "device" | "pending" | "finishing" | "ready" | "start";

function codexPhase(connection?: FactoriesFactoryCodexConnection): CodexPhase {
  if (!connection) return "start";
  if (connection.status === "pending" && connection.verificationUrl && connection.verificationCode) return "device";
  if (connection.status === "pending") return "pending";
  if (connection.status === "finishing" || connection.status === "canceling" || connection.reservationHeld === true)
    return "finishing";
  if (connection.status === "ready") return "ready";
  return "start";
}

function DeviceSignIn({
  connection,
  cancelAction,
}: {
  connection: FactoriesFactoryCodexConnection;
  cancelAction: ReactNode;
}) {
  const copy = CODEX_CONNECTION_COPY;
  return (
    <div className="space-y-4 text-sm" data-testid="first-run-codex-device">
      <p>{copy.subscriptionDevice}</p>
      <div className="flex items-center justify-between gap-3 rounded-md border border-border bg-muted/40 px-4 py-3">
        <p className="break-all font-mono text-xl tracking-widest" data-testid="first-run-codex-code">
          {connection.verificationCode}
        </p>
        <CopyButton
          text={connection.verificationCode ?? ""}
          ariaLabel={copy.subscriptionCopyCode}
          copiedAriaLabel={copy.subscriptionCodeCopied}
          onCopyError={() => showErrorToast(copy.subscriptionCopyFailed)}
        />
      </div>
      <Button asChild className="w-full">
        <a href={connection.verificationUrl} target="_blank" rel="noreferrer">
          Open ChatGPT
          <ExternalLink className="size-4" aria-hidden />
        </a>
      </Button>
      <p className="text-muted-foreground">{copy.subscriptionDeviceSettings}</p>
      <div className="flex items-center gap-2 text-muted-foreground" role="status">
        <Loader2 className="size-4 shrink-0 animate-spin" aria-hidden />
        <p>{copy.subscriptionWaiting}</p>
      </div>
      {cancelAction}
    </div>
  );
}

function WaitingSignIn({
  phase,
  connection,
  cancelAction,
}: {
  phase: "pending" | "finishing";
  connection?: FactoriesFactoryCodexConnection;
  cancelAction: ReactNode;
}) {
  const copy = CODEX_CONNECTION_COPY;
  return (
    <div className="space-y-4 text-sm">
      <div className="flex items-start gap-2 text-muted-foreground" role="status">
        <Loader2 className="mt-0.5 size-4 shrink-0 animate-spin" aria-hidden />
        <div className="space-y-2">
          {phase === "pending" ? <p>{copy.subscriptionPending}</p> : <FinishingSignIn connection={connection} />}
        </div>
      </div>
      {phase === "pending" ? cancelAction : null}
    </div>
  );
}

function signInCleanupMessage(status?: string): string {
  const copy = CODEX_CONNECTION_COPY;
  if (status === "canceling") return copy.subscriptionCanceling;
  if (status === "finishing" || status === "ready") return copy.subscriptionFinishing;
  return copy.subscriptionStopping;
}

function FinishingSignIn({ connection }: { connection?: FactoriesFactoryCodexConnection }) {
  const [delayed, setDelayed] = useState(false);
  const copy = CODEX_CONNECTION_COPY;
  useEffect(() => {
    setDelayed(false);
    const timer = setTimeout(() => setDelayed(true), 60000);
    return () => clearTimeout(timer);
  }, [connection?.activeTaskId]);

  return (
    <>
      <p>{signInCleanupMessage(connection?.status)}</p>
      {delayed ? <p>{copy.subscriptionDelayed}</p> : null}
    </>
  );
}

function ReadySignIn({ connection }: { connection?: FactoriesFactoryCodexConnection }) {
  const copy = CODEX_CONNECTION_COPY;
  const label = connection?.accountLabel;
  return (
    <p className="text-sm" role="status">
      {label ? copy.subscriptionReadyAs(label) : copy.subscriptionReady}
    </p>
  );
}

export function CodexDeviceSignIn({
  disabled,
  connection,
  starting,
  canceling,
  onStart,
  onCancel,
  cancelAction,
}: {
  disabled: boolean;
  connection?: FactoriesFactoryCodexConnection;
  starting: boolean;
  canceling: boolean;
  onStart: () => void;
  onCancel: () => void;
  cancelAction?: ReactNode;
}) {
  const copy = CODEX_CONNECTION_COPY;
  const phase = codexPhase(connection);
  const signInCancelAction =
    cancelAction === undefined ? (
      <LoadingButton type="button" variant="outline" loading={canceling} onClick={onCancel}>
        {copy.subscriptionCancel}
      </LoadingButton>
    ) : (
      cancelAction
    );
  return (
    <fieldset disabled={disabled} className="m-0 min-w-0 border-0 p-0" data-testid="first-run-codex-subscription">
      {phase === "start" ? (
        <LoadingButton
          type="button"
          disabled={disabled || canceling}
          loading={starting}
          loadingText="Starting sign-in..."
          onClick={onStart}
        >
          Connect ChatGPT
        </LoadingButton>
      ) : null}
      {phase === "device" && connection ? (
        <DeviceSignIn connection={connection} cancelAction={signInCancelAction} />
      ) : null}
      {phase === "pending" || phase === "finishing" ? (
        <WaitingSignIn phase={phase} connection={connection} cancelAction={signInCancelAction} />
      ) : null}
      {phase === "ready" ? <ReadySignIn connection={connection} /> : null}
      {phase === "start" && connection?.status === "reconnect_required" ? (
        <p className="mt-3 text-[13px]">{copy.subscriptionReconnect}</p>
      ) : null}
    </fieldset>
  );
}
