import { cn } from "@/lib/utils";
import { CopyButton } from "@/ui/CopyButton";
import { Sheet, SheetContent, SheetDescription, SheetHeader, SheetTitle } from "@/ui/sheet";

import type { AutomationStage } from "./automationsViewModel";
import { MONO_LOG_CLASSNAME } from "./redesignFormat";

/** The full agent log for one stage, as plain monospace text. */
export function RawLogPre({ log, className }: { log: string; className?: string }) {
  if (!log) {
    return <p className="text-[13px] text-muted-foreground">This stage has no agent log.</p>;
  }
  return (
    <pre
      className={cn(
        "max-h-full overflow-auto rounded-md border border-border bg-muted/40 p-3",
        MONO_LOG_CLASSNAME,
        className,
      )}
      data-testid="redesign-raw-log"
    >
      {log}
    </pre>
  );
}

export function RawLogSheet({
  stage,
  open,
  onOpenChange,
}: {
  stage: AutomationStage | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent
        side="right"
        className="flex w-full flex-col gap-3 sm:max-w-3xl"
        data-testid="redesign-raw-log-sheet"
      >
        <SheetHeader className="pr-8">
          <div className="flex items-center justify-between gap-3">
            <SheetTitle className="text-[15px]">{stage ? `${stage.name} log` : "Log"}</SheetTitle>
            {stage?.rawLog ? (
              <CopyButton text={stage.rawLog} variant="button" buttonVariant="outline" ariaLabel="Copy log">
                Copy log
              </CopyButton>
            ) : null}
          </div>
          <SheetDescription>
            {stage
              ? `${stage.componentName}${stage.model ? ` · ${stage.model}` : ""} · ${stage.duration}`
              : "Select a stage to read its log."}
          </SheetDescription>
        </SheetHeader>
        <div className="min-h-0 flex-1 overflow-hidden">
          <RawLogPre log={stage?.rawLog ?? ""} className="h-full" />
        </div>
      </SheetContent>
    </Sheet>
  );
}
