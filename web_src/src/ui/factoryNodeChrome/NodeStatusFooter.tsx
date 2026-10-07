import type { ReactNode } from "react";
import { cn } from "@/lib/utils";
import { FactoryNodeStatusGlyph } from "./StatusGlyph";
import { factoryNodeStatusLabel, factoryNodeStatusStripClass } from "./status";
import type { FactoryNodeStatus } from "./types";

export function NodeStatusFooter({
  status,
  metrics,
  label,
}: {
  status: FactoryNodeStatus;
  metrics: ReactNode | null;
  /** Override status word (e.g. edit-mode "No run"). */
  label?: string;
}) {
  return (
    <div
      className={cn(
        "flex items-center justify-between gap-2 border-t px-3.5 py-2",
        factoryNodeStatusStripClass(status),
      )}
    >
      <span className="inline-flex items-center gap-1.5 text-[11px] font-medium">
        <FactoryNodeStatusGlyph status={status} />
        {label ?? factoryNodeStatusLabel(status)}
      </span>
      <span className="min-w-0 max-w-[60%] truncate text-right text-[11px] font-medium tabular-nums opacity-80">
        {metrics ?? "—"}
      </span>
    </div>
  );
}
