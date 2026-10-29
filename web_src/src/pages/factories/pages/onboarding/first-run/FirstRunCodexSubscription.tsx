import { LoadingButton } from "@/components/ui/loading-button";
import { showErrorToast } from "@/lib/toast";
import { CopyButton } from "@/ui/CopyButton";
import type { FactoriesFactoryCodexConnection } from "@/api-client";
import { useEffect, useState } from "react";

import { ConnectOptionRow, IntegrationChoiceIcon } from "../onboardingSteps";
import { FIRST_RUN_COPY } from "./firstRunCopy";

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
  canceling,
  onCancel,
}: {
  connection: FactoriesFactoryCodexConnection;
  canceling: boolean;
  onCancel: () => void;
}) {
  const copy = FIRST_RUN_COPY.agent;
  return (
    <div className="mt-3 space-y-2 text-[13px]" data-testid="first-run-codex-device">
      <p>{copy.subscriptionDevice}</p>
      <p>
        <a className="underline" href={connection.verificationUrl} target="_blank" rel="noreferrer">
          {connection.verificationUrl}
        </a>
      </p>
      <div className="flex items-center gap-2">
        <p className="font-mono text-base" data-testid="first-run-codex-code">
          {connection.verificationCode}
        </p>
        <CopyButton
          text={connection.verificationCode ?? ""}
          ariaLabel={copy.subscriptionCopyCode}
          copiedAriaLabel={copy.subscriptionCodeCopied}
          onCopyError={() => showErrorToast(copy.subscriptionCopyFailed)}
        />
      </div>
      <p className="text-muted-foreground">{copy.subscriptionDeviceSettings}</p>
      <LoadingButton type="button" variant="outline" loading={canceling} onClick={onCancel}>
        {copy.subscriptionCancel}
      </LoadingButton>
    </div>
  );
}

function WaitingSignIn({
  phase,
  connection,
  canceling,
  onCancel,
}: {
  phase: "pending" | "finishing";
  connection?: FactoriesFactoryCodexConnection;
  canceling: boolean;
  onCancel: () => void;
}) {
  const copy = FIRST_RUN_COPY.agent;
  return (
    <div className="mt-3 space-y-2 text-[13px]" role="status">
      {phase === "pending" ? <p>{copy.subscriptionPending}</p> : <FinishingSignIn connection={connection} />}
      {phase === "pending" ? (
        <LoadingButton type="button" variant="outline" loading={canceling} onClick={onCancel}>
          {copy.subscriptionCancel}
        </LoadingButton>
      ) : null}
    </div>
  );
}

function signInCleanupMessage(status?: string): string {
  const copy = FIRST_RUN_COPY.agent;
  if (status === "canceling") return copy.subscriptionCanceling;
  if (status === "finishing" || status === "ready") return copy.subscriptionFinishing;
  return copy.subscriptionStopping;
}

function FinishingSignIn({ connection }: { connection?: FactoriesFactoryCodexConnection }) {
  const [delayed, setDelayed] = useState(false);
  const copy = FIRST_RUN_COPY.agent;
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
  const copy = FIRST_RUN_COPY.agent;
  const label = connection?.accountLabel;
  return (
    <p className="mt-3 text-[13px]" role="status">
      {label ? copy.subscriptionReadyAs(label) : copy.subscriptionReady}
    </p>
  );
}

export function FirstRunCodexSubscription({
  selected,
  disabled,
  connection,
  starting,
  canceling,
  onSelect,
  onStart,
  onCancel,
}: {
  selected: boolean;
  disabled: boolean;
  connection?: FactoriesFactoryCodexConnection;
  starting: boolean;
  canceling: boolean;
  onSelect: () => void;
  onStart: () => void;
  onCancel: () => void;
}) {
  const copy = FIRST_RUN_COPY.agent;
  const phase = codexPhase(connection);
  return (
    <fieldset disabled={disabled} className="m-0 min-w-0 border-0 p-0" data-testid="first-run-codex-subscription">
      <ConnectOptionRow
        icon={<IntegrationChoiceIcon name="openai" />}
        title={copy.subscription}
        detail={copy.subscriptionHelper}
        selected={selected}
        connectLabel={phase === "start" || phase === "ready" ? "ChatGPT" : undefined}
        connected={phase === "ready"}
        disabled={disabled || starting || canceling}
        onSelect={onSelect}
        onConnect={onStart}
      />
      {selected ? (
        <>
          {phase === "device" && connection ? (
            <DeviceSignIn connection={connection} canceling={canceling} onCancel={onCancel} />
          ) : null}
          {phase === "pending" || phase === "finishing" ? (
            <WaitingSignIn phase={phase} connection={connection} canceling={canceling} onCancel={onCancel} />
          ) : null}
          {phase === "ready" ? <ReadySignIn connection={connection} /> : null}
        </>
      ) : null}
      {phase === "start" && connection?.status === "reconnect_required" ? (
        <p className="mt-3 text-[13px]">{copy.subscriptionReconnect}</p>
      ) : null}
    </fieldset>
  );
}
