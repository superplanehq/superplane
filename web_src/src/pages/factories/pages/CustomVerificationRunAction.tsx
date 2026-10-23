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
      setStatus({ type: "success", message: `${verification.name} started.` });
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

  const pending = run.isPending;
  const one = verifications[0];

  return (
    <div
      className={placement === "logs" ? "relative z-20 ml-auto flex items-center gap-2" : "mt-1.5 flex flex-col gap-1"}
      data-testid={`custom-verification-action-${workOrderId}`}
    >
      {verifications.length === 1 && one ? (
        <Button
          type="button"
          size="sm"
          variant="outline"
          className={placement === "logs" ? "h-7 px-2.5 text-xs" : "h-7 w-full justify-center text-xs"}
          disabled={pending}
          onClick={() => void start(one)}
        >
          {pending ? (
            <Loader2 className="size-3.5 animate-spin" aria-hidden />
          ) : (
            <Play className="size-3.5" aria-hidden />
          )}
          {pending ? "Starting…" : placement === "logs" ? "Run" : "Run automation"}
        </Button>
      ) : (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              type="button"
              size="sm"
              variant="outline"
              className={placement === "logs" ? "h-7 px-2.5 text-xs" : "h-7 w-full justify-center text-xs"}
              disabled={pending}
            >
              {pending ? (
                <Loader2 className="size-3.5 animate-spin" aria-hidden />
              ) : (
                <Play className="size-3.5" aria-hidden />
              )}
              {pending ? "Starting…" : placement === "logs" ? "Run" : "Run automation"}
              {!pending ? <ChevronDown className="size-3.5" aria-hidden /> : null}
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="start" className="w-56">
            {verifications.map((verification) => (
              <DropdownMenuItem key={verification.id} onClick={() => void start(verification)}>
                {verification.name}
              </DropdownMenuItem>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>
      )}
      {status && placement === "board" ? (
        <div
          className={
            status.type === "error"
              ? "flex items-start gap-1 text-[11px] leading-4 text-destructive"
              : "flex items-center gap-1 text-[11px] leading-4 text-emerald-600 dark:text-emerald-400"
          }
          role={status.type === "error" ? "alert" : "status"}
        >
          {status.type === "error" ? (
            <AlertCircle className="mt-0.5 size-3 shrink-0" aria-hidden />
          ) : (
            <CheckCircle2 className="size-3 shrink-0" aria-hidden />
          )}
          <span>{status.message}</span>
          {status.type === "error" && lastAutomationId ? (
            <Button
              type="button"
              variant="link"
              size="sm"
              className="h-auto shrink-0 p-0 text-[11px] text-destructive"
              onClick={() => {
                const verification = verifications.find((entry) => entry.id === lastAutomationId);
                if (verification) {
                  void start(verification);
                }
              }}
            >
              Try again
            </Button>
          ) : null}
        </div>
      ) : null}
    </div>
  );
}
