import { cn } from "@/lib/utils";
import type { ReactNode, SVGProps } from "react";

import { getWorkOrderDisplayStatusMeta, type WorkOrderDisplayStatus } from "../lib/workOrderProgress";

const DISK = "inline-flex size-3.5 shrink-0 items-center justify-center rounded-full";
const GLYPH = "size-[9px] text-white dark:text-zinc-950";

/**
 * Compact status mark next to a task title.
 *
 * Closed and waiting states use a filled disk and a light glyph, in the
 * Linear / Programa style. Draft stays a dashed ring. Running is a spinning
 * ring, not a filled disk.
 */
export function WorkOrderStatusIcon({
  status,
  title,
  className,
  "aria-label": ariaLabel,
  "aria-hidden": ariaHidden,
  "data-testid": testId,
}: {
  status: WorkOrderDisplayStatus;
  title?: string;
  className?: string;
  "aria-label"?: string;
  "aria-hidden"?: boolean;
  "data-testid"?: string;
}) {
  return (
    <span
      className={cn(markClassName(status), className)}
      title={title}
      aria-label={ariaLabel}
      aria-hidden={ariaHidden}
      data-testid={testId}
      data-status-mark={status}
    >
      {markGlyph(status)}
    </span>
  );
}

function markClassName(status: WorkOrderDisplayStatus): string {
  const tone = getWorkOrderDisplayStatusMeta(status).dotClassName;
  if (status === "draft") {
    return cn(DISK, "border border-dashed border-[color:var(--status-draft-dot)] bg-transparent");
  }
  if (status === "running") {
    return cn(
      DISK,
      "border-2 border-[color:var(--status-running-dot)] border-r-transparent bg-transparent animate-spin",
    );
  }
  return cn(DISK, tone);
}

function markGlyph(status: WorkOrderDisplayStatus): ReactNode {
  if (status === "draft" || status === "running") {
    return null;
  }
  if (status === "waiting") {
    return <ClockHands />;
  }
  if (status === "completed") {
    return <CheckGlyph />;
  }
  if (status === "failed") {
    return <XGlyph />;
  }
  if (status === "rejected") {
    return <BanGlyph />;
  }
  return <MinusGlyph />;
}

function Glyph({ children, ...props }: SVGProps<SVGSVGElement>) {
  return (
    <svg viewBox="0 0 16 16" className={GLYPH} fill="none" aria-hidden data-status-glyph {...props}>
      {children}
    </svg>
  );
}

function CheckGlyph() {
  return (
    <Glyph>
      <path
        d="M3.4 8.3 6.6 11.4 12.6 4.6"
        stroke="currentColor"
        strokeWidth="2.25"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
    </Glyph>
  );
}

function XGlyph() {
  return (
    <Glyph>
      <path d="M4.4 4.4 11.6 11.6" stroke="currentColor" strokeWidth="2.25" strokeLinecap="round" />
      <path d="M11.6 4.4 4.4 11.6" stroke="currentColor" strokeWidth="2.25" strokeLinecap="round" />
    </Glyph>
  );
}

function ClockHands() {
  return (
    <Glyph>
      <path d="M8 4.2 V8.2" stroke="currentColor" strokeWidth="2.25" strokeLinecap="round" />
      <path d="M8 8.2 H11.2" stroke="currentColor" strokeWidth="2.25" strokeLinecap="round" />
    </Glyph>
  );
}

function BanGlyph() {
  return (
    <Glyph>
      <path d="M4.2 11.8 11.8 4.2" stroke="currentColor" strokeWidth="2.25" strokeLinecap="round" />
    </Glyph>
  );
}

function MinusGlyph() {
  return (
    <Glyph>
      <path d="M4.2 8 H11.8" stroke="currentColor" strokeWidth="2.25" strokeLinecap="round" />
    </Glyph>
  );
}
