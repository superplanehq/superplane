import { Dialog, DialogClose, DialogContent, DialogTitle } from "@/components/ui/dialog";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import type {
  SplitRunPhase,
  SplitRunPhaseStatus,
  SplitRunStreamLine,
} from "@/pages/factories/pages/work-order-split-run/splitRunMocks";
import { ArrowLeft, Logs } from "lucide-react";
import { lazy, Suspense, useCallback, useState } from "react";
import type { ExecutionInfo } from "../../../pages/app/mappers/types";
import { isExecutionInFlight, type RunnerLiveLogDialogProps } from "./types";

const PhaseLogCard = lazy(async () => {
  const mod = await import("@/pages/factories/pages/work-order-split-run/PhaseLogCard");
  return { default: mod.PhaseLogCard };
});

const SEE_LOGS_LABEL = "See logs";
const GO_BACK_LABEL = "Go back";

export function RunnerLiveLogDialog({
  title,
  canvasMode,
  execution,
  component,
  iconSlug,
  session,
}: RunnerLiveLogDialogProps) {
  const [open, setOpen] = useState(false);

  const handleOpen = useCallback((e: React.MouseEvent<HTMLButtonElement>) => {
    e.preventDefault();
    e.stopPropagation();
    setOpen(true);
  }, []);

  const stopCanvasInteraction = useCallback((e: React.MouseEvent<HTMLButtonElement>) => {
    e.stopPropagation();
  }, []);

  if (!execution) {
    return null;
  }

  if (canvasMode !== "live") {
    return null;
  }

  const phase = phaseForRunnerExecution({ title, component, iconSlug, execution });

  return (
    <>
      <Tooltip>
        <TooltipTrigger asChild>
          <button
            type="button"
            aria-label={SEE_LOGS_LABEL}
            className="flex size-6 shrink-0 items-center justify-center rounded-md text-muted-foreground hover:bg-muted hover:text-foreground"
            onClick={handleOpen}
            onMouseDown={stopCanvasInteraction}
            onPointerDown={stopCanvasInteraction}
          >
            <Logs className="size-3.5" aria-hidden />
          </button>
        </TooltipTrigger>
        <TooltipContent side="top">{SEE_LOGS_LABEL}</TooltipContent>
      </Tooltip>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent
          size="90vw"
          showCloseButton={false}
          className="flex flex-col gap-0 overflow-hidden bg-background p-0"
          onClick={(e) => e.stopPropagation()}
        >
          <div className="flex shrink-0 items-center gap-2 border-b border-border px-3 py-2">
            <DialogClose
              aria-label={GO_BACK_LABEL}
              className="flex size-7 shrink-0 items-center justify-center rounded-md text-muted-foreground hover:bg-muted hover:text-foreground"
            >
              <ArrowLeft className="size-3.5" aria-hidden />
            </DialogClose>
            <DialogTitle className="text-sm font-medium">{title}</DialogTitle>
          </div>
          <div className="min-h-0 flex-1 overflow-y-auto p-2">
            <Suspense fallback={null}>
              <PhaseLogCard
                phase={phase}
                expanded
                collapsible={false}
                organizationId={session?.organizationId}
                canvasId={session?.canvasId}
              />
            </Suspense>
          </div>
        </DialogContent>
      </Dialog>
    </>
  );
}

function phaseForRunnerExecution({
  title,
  component,
  iconSlug,
  execution,
}: {
  title: string;
  component?: string;
  iconSlug?: string;
  execution: ExecutionInfo;
}): SplitRunPhase {
  const status = statusForRunnerExecution(execution);
  const streamLine: SplitRunStreamLine = {
    id: execution.id,
    at: execution.createdAt,
    componentName: title,
    status,
    component: component,
    executionId: execution.id,
    iconSlug,
  };

  return {
    id: execution.id,
    name: title,
    status,
    duration: "",
    componentName: title,
    artifacts: [],
    stream: [streamLine],
    canvasSteps: [],
  };
}

function statusForRunnerExecution(execution: ExecutionInfo): SplitRunPhaseStatus {
  if (isExecutionInFlight(execution)) {
    return "running";
  }
  if (execution.result === "RESULT_FAILED") {
    return "failed";
  }
  if (execution.result === "RESULT_CANCELLED") {
    return "cancelled";
  }
  if (execution.result === "RESULT_PASSED") {
    return "passed";
  }
  return "pending";
}
