import { cn } from "@/lib/utils";
import { FactoryNodeStatusGlyph } from "./StatusGlyph";
import { factoryNodeStatusLabel, factoryNodeStatusStripClass } from "./status";
import type { FactoryNodeStatus } from "./types";

export function NodeStatusFooter({ status, metrics }: { status: FactoryNodeStatus; metrics: string | null }) {
  return (
    <div className={cn("flex items-center justify-between gap-2 border-t px-3.5 py-2", factoryNodeStatusStripClass(status))}>
      <span className="inline-flex items-center gap-1.5 text-[11px] font-medium">
        <FactoryNodeStatusGlyph status={status} />
        {factoryNodeStatusLabel(status)}
      </span>
      <span className="font-mono text-[11px] tabular-nums opacity-80">{metrics ?? "—"}</span>
    </div>
  );
}
