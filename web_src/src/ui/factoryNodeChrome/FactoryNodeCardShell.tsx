import React from "react";
import { getDraftDiffOutlineClassName, type DraftDiffStatus } from "@/lib/draftDiff";
import { resolveNodeIconColorClass } from "@/lib/colors";
import { cn, resolveIcon } from "@/lib/utils";
import { toTestId } from "@/lib/testID";
import { NodeStatusFooter } from "./NodeStatusFooter";
import type { FactoryNodeStatus } from "./types";

type FactoryNodeCardShellProps = {
  title: string;
  iconSrc?: string;
  iconSlug?: string;
  iconColor?: string;
  selected?: boolean;
  subtitle: string | null;
  status: FactoryNodeStatus;
  metrics: string | null;
  draftDiffStatus?: DraftDiffStatus;
  dimBodyBelowHeader?: boolean;
  isCompactView?: boolean;
};

export function FactoryNodeCardShell({
  title,
  iconSrc,
  iconSlug,
  iconColor,
  selected = false,
  subtitle,
  status,
  metrics,
  draftDiffStatus,
  dimBodyBelowHeader = false,
  isCompactView,
}: FactoryNodeCardShellProps) {
  const Icon = React.useMemo(() => resolveIcon(iconSlug), [iconSlug]);

  return (
    <div
      data-testid={toTestId(`factory-node-${title}`)}
      className={cn(
        "canvas-node-drag-handle cursor-pointer overflow-hidden rounded-2xl border border-border bg-card text-left shadow-[0_1px_2px_rgba(15,23,42,0.04),0_4px_12px_rgba(15,23,42,0.04)] dark:shadow-[0_1px_2px_rgba(0,0,0,0.35)] w-[280px]",
        getDraftDiffOutlineClassName(draftDiffStatus),
        selected && "border-ring shadow-[0_0_0_3px_rgba(15,23,42,0.06)] dark:shadow-[0_0_0_3px_rgba(163,163,163,0.25)]",
        dimBodyBelowHeader && "opacity-70",
      )}
    >
      <div className="px-3.5 pt-3.5 pb-3 text-left">
        <div className="flex items-start justify-start gap-2.5">
          <span className="mt-0.5 flex size-7 shrink-0 items-center justify-center rounded-md border border-border bg-muted">
            {iconSrc ? (
              <img src={iconSrc} alt="" className="size-4 object-contain opacity-90" />
            ) : (
              <Icon size={16} className={resolveNodeIconColorClass(iconColor)} />
            )}
          </span>
          <div className="min-w-0 flex-1 text-left">
            <div className="text-left text-[14px] leading-snug font-semibold tracking-[-0.015em] text-card-foreground">
              {title}
            </div>
            {subtitle && !isCompactView ? (
              <p className="mt-0.5 text-left text-[12px] leading-snug text-muted-foreground">{subtitle}</p>
            ) : null}
          </div>
        </div>
      </div>
      {!isCompactView ? <NodeStatusFooter status={status} metrics={metrics} /> : null}
    </div>
  );
}
