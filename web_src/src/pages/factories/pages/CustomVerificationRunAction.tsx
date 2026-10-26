import { Button } from "@/components/ui/button";
import { useRunFactoryCustomVerification } from "@/hooks/useFactoryCustomVerificationData";
import { getApiErrorMessage } from "@/lib/errors";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { AlertCircle, CheckCircle2, ChevronDown, Loader2, Play } from "lucide-react";
import { useState } from "react";

import type { ManualCustomVerification } from "../lib/customVerification";

export function CustomVerificationRunAction({
  organizationId,
  factoryId,
  workOrderId,
  verifications,
  placement = "board",
}: {
  organizationId: string;
  factoryId: string;
  workOrderId: string;
  verifications: readonly ManualCustomVerification[];
  placement?: "board" | "logs";
}) {
  const run = useRunFactoryCustomVerification(organizationId, factoryId);
  const [status, setStatus] = useState<{ type: "success" | "error"; message: string } | null>(null);
  const [lastAutomationId, setLastAutomationId] = useState("");

  if (verifications.length === 0) {
    return null;
  }

  const start = async (verification: ManualCustomVerification) => {
    setStatus(null);
    setLastAutomationId(verification.id);
    try {
      await run.mutateAsync({ automationId: verification.id, workOrderId });
      if (placement === "board") {
        setStatus({ type: "success", message: `${verification.name} started.` });
      }
    } catch (error) {
      setStatus({
        type: "error",
        message: getApiErrorMessage(
          error,
          "SuperPlane could not start the automation. Confirm that the task has an open pull request, then try again.",
        ),
      });
    }
  };

  return (
    <div
      className={
        placement === "logs"
          ? "relative z-20 flex max-w-[16rem] flex-col items-end gap-1"
          : "mt-1.5 flex flex-col gap-1"
      }
      data-testid={`custom-verification-action-${workOrderId}`}
    >
      <RunActionControls placement={placement} pending={run.isPending} verifications={verifications} onStart={start} />
      <RunActionStatus
        status={status}
        lastAutomationId={lastAutomationId}
        verifications={verifications}
        onRetry={start}
      />
    </div>
  );
}

function RunActionControls({
  placement,
  pending,
  verifications,
  onStart,
}: {
  placement: "board" | "logs";
  pending: boolean;
  verifications: readonly ManualCustomVerification[];
  onStart: (verification: ManualCustomVerification) => Promise<void>;
}) {
  const one = verifications[0];
  const compact = placement === "logs";
  const label = pending ? "Starting…" : compact ? "Run" : "Run automation";
  const className = compact ? "h-7 px-2.5 text-xs" : "h-7 w-full justify-center text-xs";
  if (verifications.length === 1 && one) {
    return (
      <Button
        type="button"
        size="sm"
        variant="outline"
        className={className}
        disabled={pending}
        onClick={() => void onStart(one)}
      >
        {pending ? (
          <Loader2 className="size-3.5 animate-spin" aria-hidden />
        ) : (
          <Play className="size-3.5" aria-hidden />
        )}
        {label}
      </Button>
    );
  }
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button type="button" size="sm" variant="outline" className={className} disabled={pending}>
          {pending ? (
            <Loader2 className="size-3.5 animate-spin" aria-hidden />
          ) : (
            <Play className="size-3.5" aria-hidden />
          )}
          {label}
          {!pending ? <ChevronDown className="size-3.5" aria-hidden /> : null}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-56">
        {verifications.map((verification) => (
          <DropdownMenuItem key={verification.id} onClick={() => void onStart(verification)}>
            {verification.name}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function RunActionStatus({
  status,
  lastAutomationId,
  verifications,
  onRetry,
}: {
  status: { type: "success" | "error"; message: string } | null;
  lastAutomationId: string;
  verifications: readonly ManualCustomVerification[];
  onRetry: (verification: ManualCustomVerification) => Promise<void>;
}) {
  if (!status) {
    return null;
  }
  const failed = status.type === "error";
  return (
    <div
      className={
        failed
          ? "flex items-start gap-1 text-[11px] leading-4 text-destructive"
          : "flex items-center gap-1 text-[11px] leading-4 text-emerald-600 dark:text-emerald-400"
      }
      role={failed ? "alert" : "status"}
    >
      {failed ? (
        <AlertCircle className="mt-0.5 size-3 shrink-0" aria-hidden />
      ) : (
        <CheckCircle2 className="size-3 shrink-0" aria-hidden />
      )}
      <span>{status.message}</span>
      {failed && lastAutomationId ? (
        <Button
          type="button"
          variant="link"
          size="sm"
          className="h-auto shrink-0 p-0 text-[11px] text-destructive"
          onClick={() => {
            const verification = verifications.find((entry) => entry.id === lastAutomationId);
            if (verification) {
              void onRetry(verification);
            }
          }}
        >
          Try again
        </Button>
      ) : null}
    </div>
  );
}
