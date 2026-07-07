import * as AccordionPrimitive from "@radix-ui/react-accordion";
import { ChevronRight } from "lucide-react";
import { useEffect, useRef } from "react";
import { formatDuration } from "@/lib/duration";
import { withEventStatusBadgeClasses } from "@/lib/eventStatusBadge";
import { cn } from "@/lib/utils";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIconMaps";
import { AccordionContent, AccordionItem } from "@/ui/accordion";
import { RunNodeIcon, RUN_NODE_ICON_SIZE } from "./RunNodeIcon";
import { RunInspectorStepTimeline } from "./RunInspectorStepTimeline";
import type { RunInspectorNodeSection } from "./runNodeDetailModel";

export function RunInspectorNodeAccordion({
  section,
  componentIconMap,
  isOpen,
  onRerun,
  rerunPending,
}: {
  section: RunInspectorNodeSection;
  componentIconMap: Record<string, string>;
  isOpen: boolean;
  onRerun: () => void;
  rerunPending: boolean;
}) {
  const iconSrc = getHeaderIconSrc(section.workflowNode?.component);
  const iconSlug = section.workflowNode?.component ? componentIconMap[section.workflowNode.component] : undefined;
  const itemRef = useRef<HTMLDivElement>(null);
  const wasOpenRef = useRef(false);

  useEffect(() => {
    if (!isOpen) {
      wasOpenRef.current = false;
      return;
    }

    if (wasOpenRef.current) {
      return;
    }

    wasOpenRef.current = true;
    const frame = window.requestAnimationFrame(() => {
      itemRef.current?.scrollIntoView?.({ block: "start", behavior: "smooth" });
    });

    return () => window.cancelAnimationFrame(frame);
  }, [isOpen]);

  return (
    <AccordionItem
      ref={itemRef}
      value={section.nodeId}
      className="scroll-mt-8 border-slate-950/10 dark:border-gray-800"
    >
      <AccordionPrimitive.Header
        className={cn(
          "flex items-center bg-white transition-colors hover:bg-slate-50 dark:bg-gray-950 dark:hover:bg-gray-900",
          isOpen &&
            "sticky top-8 z-20 bg-[#e1f5ff] text-slate-950 shadow-[0_1px_0_rgba(15,23,42,0.08)] dark:bg-sky-950 dark:text-gray-100 dark:shadow-[0_1px_0_rgba(31,41,55,0.8)]",
        )}
      >
        <AccordionPrimitive.Trigger className="flex min-w-0 flex-1 items-center gap-3 px-4 py-3 text-left hover:no-underline">
          <ChevronRight
            className={cn(
              "h-4 w-4 shrink-0 text-slate-400 transition-transform duration-200",
              isOpen && "rotate-90 text-slate-600 dark:text-gray-300",
            )}
          />
          <RunNodeIcon
            iconSrc={iconSrc}
            iconSlug={iconSlug}
            alt={section.nodeName}
            size={RUN_NODE_ICON_SIZE}
            className="text-slate-500 dark:text-gray-400"
          />
          <span className="min-w-0 flex-1 truncate text-sm font-medium text-slate-900 dark:text-gray-100">
            {section.nodeName}
          </span>
        </AccordionPrimitive.Trigger>
        <NodeMetadata section={section} onRerun={onRerun} rerunPending={rerunPending} />
      </AccordionPrimitive.Header>
      <AccordionContent className="bg-slate-50 px-3 pb-3 pt-3 dark:bg-gray-950">
        <RunInspectorStepTimeline section={section} componentIconMap={componentIconMap} />
      </AccordionContent>
    </AccordionItem>
  );
}

function NodeMetadata({
  section,
  onRerun,
  rerunPending,
}: {
  section: RunInspectorNodeSection;
  onRerun: () => void;
  rerunPending: boolean;
}) {
  return (
    <div className="ml-auto flex shrink-0 items-center gap-3 px-4 text-xs text-slate-500 dark:text-gray-400">
      {section.isTrigger ? (
        <button
          type="button"
          disabled={rerunPending}
          className="inline-flex h-6 items-center rounded-sm border border-slate-200 bg-white px-2 text-xs font-medium text-slate-700 transition-colors hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60 dark:border-gray-700 dark:bg-gray-950 dark:text-gray-200 dark:hover:bg-gray-800 dark:hover:text-gray-100"
          onClick={(event) => {
            event.stopPropagation();
            onRerun();
          }}
        >
          {rerunPending ? "Rerun..." : "Rerun"}
        </button>
      ) : null}
      {section.isTrigger && section.createdAt ? (
        <span>{formatEventTimestamp(section.createdAt)}</span>
      ) : section.durationMs !== undefined ? (
        <span>{formatStepDuration(section.durationMs)}</span>
      ) : null}
      {section.badge ? <EventStatusBadge badgeColor={section.badge.badgeColor} label={section.badge.label} /> : null}
    </div>
  );
}

function formatStepDuration(durationMs: number): string {
  if (durationMs > 0 && durationMs < 1000) return "<1s";
  return formatDuration(durationMs);
}

function formatEventTimestamp(timestamp: string): string {
  const date = new Date(timestamp);
  if (Number.isNaN(date.getTime())) return "";

  const pad = (value: number) => String(value).padStart(2, "0");
  const months = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];
  return `${pad(date.getHours())}:${pad(date.getMinutes())} - ${date.getDate()}.${months[date.getMonth()]}`;
}

function EventStatusBadge({ badgeColor, label }: { badgeColor: string; label: string }) {
  return (
    <span
      className={cn(
        "inline-flex shrink-0 items-center justify-center rounded px-[5px] py-[1.5px] text-[10px] font-semibold uppercase tracking-wide text-white",
        withEventStatusBadgeClasses(badgeColor),
      )}
    >
      {label}
    </span>
  );
}
