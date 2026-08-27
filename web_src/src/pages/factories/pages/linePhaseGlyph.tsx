import { cn } from "@/lib/utils";
import { Check, Circle, CircleDashed, Clock, LoaderCircle, X } from "lucide-react";
import type { ReactNode } from "react";

import type { PhaseGlyphKind } from "../lib/linePhaseRuns";

const DISK = "inline-flex items-center justify-center rounded-full text-white dark:text-zinc-950";
const MARK = "size-[0.68em] stroke-[3]";

export function PhaseGlyph({ kind, className }: { kind: PhaseGlyphKind; className?: string }) {
  const shared = cn("size-3.5 shrink-0", className);

  if (kind === "running") {
    return <LoaderCircle className={cn(shared, "animate-spin text-[color:var(--status-running-dot)]")} aria-hidden />;
  }
  if (kind === "waiting") {
    return (
      <StatusDisk className={shared} tone="bg-[color:var(--status-waiting-dot)]">
        <Clock className={cn(MARK, "[&_circle]:hidden")} />
      </StatusDisk>
    );
  }
  if (kind === "failed") {
    return (
      <StatusDisk className={shared} tone="bg-[color:var(--status-failed-dot)]">
        <X className={MARK} />
      </StatusDisk>
    );
  }
  if (kind === "passed") {
    return (
      <StatusDisk className={shared} tone="bg-[color:var(--status-completed-dot)]">
        <Check className={MARK} />
      </StatusDisk>
    );
  }
  if (kind === "queued") {
    return <CircleDashed className={cn(shared, "text-muted-foreground")} aria-hidden />;
  }
  return <Circle className={cn(shared, "text-muted-foreground/40")} aria-hidden />;
}

function StatusDisk({ className, tone, children }: { className?: string; tone: string; children: ReactNode }) {
  return (
    <span className={cn(DISK, tone, className)} aria-hidden>
      {children}
    </span>
  );
}
