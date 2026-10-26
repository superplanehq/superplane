import { Button } from "@/components/ui/button";
import { useRunFactoryCustomAutomation } from "@/hooks/useFactoryCustomAutomationData";
import { getApiErrorMessage } from "@/lib/errors";
import { DropdownMenu, DropdownMenuContent, DropdownMenuItem, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { AlertCircle, CheckCircle2, ChevronDown, Loader2, Play } from "lucide-react";
import { useState } from "react";

import type { ManualCustomAutomation } from "../lib/customAutomation";

export function CustomAutomationRunAction({
  organizationId,
  factoryId,
  workOrderId,
  automations,
  placement = "board",
}: {
  organizationId: string;
  factoryId: string;
  workOrderId: string;
  automations: readonly ManualCustomAutomation[];
  placement?: "board" | "logs";
}) {
  const run = useRunFactoryCustomAutomation(organizationId, factoryId);
  const [status, setStatus] = useState<{ type: "success" | "error"; message: string } | null>(null);
  const [lastAutomationId, setLastAutomationId] = useState("");

  if (automations.length === 0) {
    return null;
  }

  const start = async (automation: ManualCustomAutomation) => {
    setStatus(null);
    setLastAutomationId(automation.id);
    try {
      await run.mutateAsync({ automationId: automation.id, workOrderId });
      if (placement === "board") {
        setStatus({ type: "success", message: `${automation.name} started.` });
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
      data-testid={`custom-automation-action-${workOrderId}`}
    >
      <RunActionControls placement={placement} pending={run.isPending} automations={automations} onStart={start} />
      <RunActionStatus status={status} lastAutomationId={lastAutomationId} automations={automations} onRetry={start} />
    </div>
  );
}

function RunActionControls({
  placement,
  pending,
  automations,
  onStart,
}: {
  placement: "board" | "logs";
  pending: boolean;
  automations: readonly ManualCustomAutomation[];
  onStart: (automation: ManualCustomAutomation) => Promise<void>;
}) {
  const one = automations[0];
  const compact = placement === "logs";
  const label = pending ? "Starting…" : compact ? "Run" : "Run automation";
  const className = compact ? "h-7 px-2.5 text-xs" : "h-7 w-full justify-center text-xs";
  if (automations.length === 1 && one) {
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
        {automations.map((automation) => (
          <DropdownMenuItem key={automation.id} onClick={() => void onStart(automation)}>
            {automation.name}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function RunActionStatus({
  status,
  lastAutomationId,
  automations,
  onRetry,
}: {
  status: { type: "success" | "error"; message: string } | null;
  lastAutomationId: string;
  automations: readonly ManualCustomAutomation[];
  onRetry: (automation: ManualCustomAutomation) => Promise<void>;
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
            const automation = automations.find((entry) => entry.id === lastAutomationId);
            if (automation) {
              void onRetry(automation);
            }
          }}
        >
          Try again
        </Button>
      ) : null}
    </div>
  );
}
