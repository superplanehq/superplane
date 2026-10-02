import { cn, resolveIcon } from "@/lib/utils";
import { FileText, Pencil, Search, SquareTerminal, Sparkles, Wrench, type LucideIcon } from "lucide-react";

import { PhaseGlyph } from "../../linePhaseGlyph";
import type { SplitRunPhaseStatus } from "../splitRunMocks";

/** Small glyph components shared by the Automations tab redesign variants. */

export function StageStatusGlyph({ status, className }: { status: SplitRunPhaseStatus; className?: string }) {
  return <PhaseGlyph kind={status} className={className} />;
}

/**
 * Status without motion. A running run reads as a steady dot: the card
 * header glyph is the one spinner on the console.
 */
export function StaticStatusGlyph({ status, className }: { status: SplitRunPhaseStatus; className?: string }) {
  if (status === "running") {
    return (
      <span
        className={cn("size-2 shrink-0 rounded-full bg-[color:var(--status-running-dot)]", className)}
        aria-hidden
      />
    );
  }
  return <StageStatusGlyph status={status} className={className} />;
}

const TOOL_ICONS: Record<string, LucideIcon> = {
  bash: SquareTerminal,
  command_execution: SquareTerminal,
  read: FileText,
  edit: Pencil,
  write: Pencil,
  search: Search,
  grep: Search,
  glob: Search,
  prompt: Sparkles,
};

export function ToolKindIcon({ type, className }: { type: string; className?: string }) {
  const Icon = TOOL_ICONS[type.toLowerCase()] ?? Wrench;
  return <Icon className={cn("size-3.5 shrink-0 text-muted-foreground", className)} aria-hidden />;
}

export function NodeIcon({ iconSlug, className }: { iconSlug?: string; className?: string }) {
  const Icon = resolveIcon(iconSlug ?? "box");
  return <Icon className={cn("size-3.5 shrink-0 text-muted-foreground", className)} aria-hidden />;
}
