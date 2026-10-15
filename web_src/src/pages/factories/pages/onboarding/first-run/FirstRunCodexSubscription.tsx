import { LoadingButton } from "@/components/ui/loading-button";
import type { FactoriesFactoryCodexConnection } from "@/api-client";

import { FIRST_RUN_COPY } from "./firstRunCopy";

type CodexPhase = "device" | "finishing" | "ready" | "start" | "idle";

function codexPhase(selected: boolean, connection?: FactoriesFactoryCodexConnection): CodexPhase {
  if (!selected) return "idle";
  if (!connection) return "start";
  if (connection.status === "pending" && connection.verificationUrl && connection.verificationCode) return "device";
  if (connection.status === "finishing" || connection.reservationHeld === true) return "finishing";
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
      <p className="font-mono text-base" data-testid="first-run-codex-code">
        {connection.verificationCode}
      </p>
      <p className="text-muted-foreground">{copy.subscriptionDeviceSettings}</p>
      <LoadingButton type="button" variant="outline" loading={canceling} onClick={onCancel}>
        {copy.subscriptionCancel}
      </LoadingButton>
    </div>
  );
}

function FinishingSignIn({ canceling, onCancel }: { canceling: boolean; onCancel: () => void }) {
  const copy = FIRST_RUN_COPY.agent;
  return (
    <div className="mt-3 space-y-2 text-[13px]" role="status">
      <p>{copy.subscriptionFinishing}</p>
      <LoadingButton type="button" variant="outline" loading={canceling} onClick={onCancel}>
        {copy.subscriptionCancel}
      </LoadingButton>
    </div>
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

function StartSignIn({ status, starting, onStart }: { status?: string; starting: boolean; onStart: () => void }) {
  const copy = FIRST_RUN_COPY.agent;
  const reconnect = status === "reconnect_required";
  return (
    <div className="mt-3 space-y-2">
      {reconnect ? <p className="text-[13px]">{copy.subscriptionReconnect}</p> : null}
      <LoadingButton type="button" loading={starting} loadingText={copy.subscriptionStart} onClick={onStart}>
        {reconnect ? copy.subscriptionReconnect : copy.subscriptionStart}
      </LoadingButton>
    </div>
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
  const phase = codexPhase(selected, connection);
  return (
    <fieldset disabled={disabled} className="m-0 min-w-0 border-0 p-0" data-testid="first-run-codex-subscription">
      <button
        type="button"
        className="w-full rounded-md border px-3 py-3 text-left disabled:opacity-50"
        aria-pressed={selected}
        onClick={onSelect}
      >
        <span className="block text-[13px] font-medium">{copy.subscription}</span>
        <span className="mt-1 block text-[13px] text-muted-foreground">{copy.subscriptionHelper}</span>
      </button>
      {phase === "device" && connection ? (
        <DeviceSignIn connection={connection} canceling={canceling} onCancel={onCancel} />
      ) : null}
      {phase === "finishing" ? <FinishingSignIn canceling={canceling} onCancel={onCancel} /> : null}
      {phase === "ready" ? <ReadySignIn connection={connection} /> : null}
      {phase === "start" ? <StartSignIn status={connection?.status} starting={starting} onStart={onStart} /> : null}
    </fieldset>
  );
}
