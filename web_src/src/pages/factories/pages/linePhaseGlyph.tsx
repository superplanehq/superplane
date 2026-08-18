import { cn } from "@/lib/utils";
import { Circle, CircleAlert, CircleCheck, CircleDashed, Clock, LoaderCircle } from "lucide-react";
import type { PhaseGlyphKind } from "../lib/linePhaseRuns";

export function PhaseGlyph({ kind, className }: { kind: PhaseGlyphKind; className?: string }) {
  const shared = cn("size-3.5 shrink-0", className);

  if (kind === "running") {
    return <LoaderCircle className={cn(shared, "animate-spin text-[#3b82f6]")} aria-hidden />;
  }
  if (kind === "waiting") {
    return <Clock className={cn(shared, "text-[#f59e0b]")} aria-hidden />;
  }
  if (kind === "failed") {
    return <CircleAlert className={cn(shared, "text-[#ef4444]")} aria-hidden />;
  }
  if (kind === "passed") {
    return <CircleCheck className={cn(shared, "text-[#10b981]")} aria-hidden />;
  }
  if (kind === "queued") {
    return <CircleDashed className={cn(shared, "text-[#a3a3a3]")} aria-hidden />;
  }
  return <Circle className={cn(shared, "text-muted-foreground/40")} aria-hidden />;
}
