import React from "react";
import { getDraftDiffOutlineClassName, type DraftDiffStatus } from "@/lib/draftDiff";
import { resolveNodeIconColorClass } from "@/lib/colors";
import { FACTORY_NODE_CARD_WIDTH, FACTORY_NODE_STEP_CARD_WIDTH } from "@/lib/factoryCanvasChrome";
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
  metrics: React.ReactNode | null;
  draftDiffStatus?: DraftDiffStatus;
  dimBodyBelowHeader?: boolean;
  isCompactView?: boolean;
  showStatusFooter?: boolean;
  statusLabel?: string;
  body?: React.ReactNode;
};

/** Monochrome logos (github / SuperPlane) need invert on dark card chrome. */
function shouldInvertMonoFactoryIcon(iconSrc: string | undefined): boolean {
  if (!iconSrc) return false;
  const lower = iconSrc.toLowerCase();
  return lower.includes("github") || lower.includes("superplane");
}

function factoryNodeWidth(body: React.ReactNode, isCompactView: boolean | undefined): number {
  return body && !isCompactView ? FACTORY_NODE_STEP_CARD_WIDTH : FACTORY_NODE_CARD_WIDTH;
}

function FactoryNodeBody({ body, isCompactView }: { body: React.ReactNode; isCompactView?: boolean }) {
  return body && !isCompactView ? body : null;
}

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
  showStatusFooter = true,
  statusLabel,
  body,
}: FactoryNodeCardShellProps) {
  const Icon = React.useMemo(() => resolveIcon(iconSlug), [iconSlug]);
  const invertMonoIcon = shouldInvertMonoFactoryIcon(iconSrc);

  return (
    <div
      data-testid={toTestId(`factory-node-${title}`)}
      style={{ width: factoryNodeWidth(body, isCompactView) }}
      className={cn(
        // No border box — transparent border left a 1px canvas gap that read as white rim.
        "canvas-node-drag-handle cursor-pointer overflow-hidden rounded-2xl border-0 bg-card text-left shadow-[0_1px_3px_rgba(15,23,42,0.06),0_2px_6px_rgba(15,23,42,0.04)] dark:shadow-[0_1px_3px_rgba(0,0,0,0.28)]",
        draftDiffStatus
          ? getDraftDiffOutlineClassName(draftDiffStatus)
          : "outline-1 outline-black/[0.04] dark:outline-white/[0.06]",
        selected &&
          "outline-ring/40 shadow-[0_0_0_2px_rgba(15,23,42,0.05)] dark:shadow-[0_0_0_2px_rgba(163,163,163,0.2)]",
        dimBodyBelowHeader && "opacity-70",
      )}
    >
      <div className="px-3.5 pt-3.5 pb-3 text-left">
        <div className="flex items-start justify-start gap-2.5">
          <span className="mt-0.5 flex size-7 shrink-0 items-center justify-center rounded-md border border-black/[0.06] dark:border-white/[0.08] bg-muted">
            {iconSrc ? (
              <img
                src={iconSrc}
                alt=""
                className={cn("size-4 object-contain opacity-90", invertMonoIcon && "dark:invert")}
              />
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
      <FactoryNodeBody body={body} isCompactView={isCompactView} />
      {showStatusFooter ? <NodeStatusFooter status={status} metrics={metrics} label={statusLabel} /> : null}
    </div>
  );
}
