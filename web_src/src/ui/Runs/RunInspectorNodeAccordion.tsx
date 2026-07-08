import * as AccordionPrimitive from "@radix-ui/react-accordion";
import { ChevronRight } from "lucide-react";
import { useEffect, useRef } from "react";
import { Button } from "@/components/ui/button";
import { formatMinutesSecondsDuration } from "@/lib/duration";
import { withEventStatusBadgeClasses } from "@/lib/eventStatusBadge";
import { cn } from "@/lib/utils";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIconMaps";
import { AccordionContent, AccordionItem } from "@/ui/accordion";
import { RunNodeIcon, RUN_NODE_ICON_SIZE } from "./RunNodeIcon";
import { RunInspectorStepTimeline } from "./RunInspectorStepTimeline";
import type {
  RunInspectorApprovalRecord,
  RunInspectorCurrentUser,
  RunInspectorNodeSection,
} from "./runNodeDetailModel";
import type { useRunInspectorActions } from "./useRunInspectorActions";

export function RunInspectorNodeAccordion({
  section,
  componentIconMap,
  organizationId,
  isOpen,
  onRerun,
  rerunPending,
  actions,
  currentUser,
}: {
  section: RunInspectorNodeSection;
  componentIconMap: Record<string, string>;
  organizationId?: string;
  isOpen: boolean;
  onRerun: () => void;
  rerunPending: boolean;
  actions: ReturnType<typeof useRunInspectorActions>;
  currentUser?: RunInspectorCurrentUser;
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
        <NodeActions section={section} actions={actions} currentUser={currentUser} />
        <NodeMetadata section={section} onRerun={onRerun} rerunPending={rerunPending} />
      </AccordionPrimitive.Header>
      <AccordionContent className="bg-slate-50 px-3 pb-3 pt-3 dark:bg-gray-950">
        <RunInspectorStepTimeline
          section={section}
          componentIconMap={componentIconMap}
          organizationId={organizationId}
        />
      </AccordionContent>
    </AccordionItem>
  );
}

function NodeActions({
  section,
  actions,
  currentUser,
}: {
  section: RunInspectorNodeSection;
  actions: ReturnType<typeof useRunInspectorActions>;
  currentUser?: RunInspectorCurrentUser;
}) {
  const actionableApproval = findActionableApprovalRecord(section.actions.approvalRecords, currentUser ?? null);
  const hasActions = section.actions.canStop || section.actions.canPushThrough || actionableApproval;

  if (!hasActions) return null;

  return (
    <div className="flex shrink-0 items-center gap-2 pl-3">
      {actionableApproval ? (
        <>
          <NodeActionButton
            label="Approve"
            tone="success"
            disabled={actions.nodeHookPending}
            onClick={() => actions.invokeNodeHook(section, "approve", { index: actionableApproval.index, comment: "" })}
          />
          <NodeActionButton
            label="Reject"
            tone="danger"
            disabled={actions.nodeHookPending}
            onClick={() => {
              const reason = window.prompt("Reason for rejection");
              if (!reason?.trim()) return;
              actions.invokeNodeHook(section, "reject", { index: actionableApproval.index, reason: reason.trim() });
            }}
          />
        </>
      ) : null}
      {section.actions.canPushThrough ? (
        <NodeActionButton
          label="Push through"
          tone="neutral"
          disabled={actions.nodeHookPending}
          onClick={() => actions.invokeNodeHook(section, "pushThrough", null)}
        />
      ) : null}
      {section.actions.canStop ? (
        <NodeActionButton
          label="Stop"
          tone="danger"
          disabled={actions.stopNodePending}
          onClick={() => actions.stopNode(section)}
        />
      ) : null}
    </div>
  );
}

function NodeActionButton({
  label,
  tone,
  disabled,
  onClick,
}: {
  label: string;
  tone: "success" | "danger" | "neutral";
  disabled: boolean;
  onClick: () => void;
}) {
  return (
    <Button
      type="button"
      variant="outline"
      size="xs"
      disabled={disabled}
      className={cn(
        "h-7 rounded-sm bg-white px-2.5 text-xs font-medium shadow-none disabled:cursor-not-allowed disabled:opacity-60 dark:bg-gray-950",
        tone === "success" &&
          "border-emerald-300 text-emerald-700 hover:bg-emerald-50 dark:border-emerald-800 dark:text-emerald-300 dark:hover:bg-emerald-950/50",
        tone === "danger" &&
          "border-red-300 text-red-600 hover:bg-red-50 dark:border-red-900 dark:text-red-300 dark:hover:bg-red-950/50",
        tone === "neutral" &&
          "border-slate-200 text-slate-700 hover:bg-slate-50 dark:border-gray-700 dark:text-gray-200 dark:hover:bg-gray-800",
      )}
      onClick={(event) => {
        event.stopPropagation();
        onClick();
      }}
    >
      {label}
    </Button>
  );
}

function findActionableApprovalRecord(
  records: RunInspectorApprovalRecord[],
  account: RunInspectorCurrentUser | null,
): RunInspectorApprovalRecord | null {
  if (!account || hasCurrentUserActed(records, account)) return null;

  return records.find((record) => record.state === "pending" && canCurrentUserActOnRecord(record, account)) ?? null;
}

function hasCurrentUserActed(records: RunInspectorApprovalRecord[], account: RunInspectorCurrentUser): boolean {
  return records.some(
    (record) => record.state !== "pending" && (record.user?.id === account.id || record.user?.email === account.email),
  );
}

function canCurrentUserActOnRecord(record: RunInspectorApprovalRecord, account: RunInspectorCurrentUser): boolean {
  if (record.type === "anyone") return true;
  if (record.type === "user") return record.user?.id === account.id || record.user?.email === account.email;
  if (record.type === "role") return !!record.roleRef?.name && (account.roles ?? []).includes(record.roleRef.name);
  if (record.type === "group") return !!record.groupRef?.name && (account.groups ?? []).includes(record.groupRef.name);
  return false;
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
  return formatMinutesSecondsDuration(durationMs);
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
