import type { CanvasesCanvasRun, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import { useEffect, useMemo, useState, type MouseEvent } from "react";
import { Timestamp } from "@/components/Timestamp";
import { appPath } from "@/lib/appPaths";
import { formatMinutesSecondsDuration } from "@/lib/duration";
import { cn } from "@/lib/utils";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIconMaps";
import { RunNodeIcon, RUN_NODE_ICON_SIZE } from "@/ui/Runs/RunNodeIcon";
import { RUN_STATUS_META, type RunStatusKey } from "@/ui/Runs/runPresentation";
import { Link as LinkIcon } from "lucide-react";
import { Link, useParams } from "react-router-dom";
import { toast } from "sonner";
import { isNormalClick } from "@/lib/linkHelpers";
import { RUNS_SIDEBAR_RUN_ROW_CLASS } from "./runsSidebarRowLayout";

interface RunRowProps {
  run: CanvasesCanvasRun;
  triggerName: string;
  title: string;
  status: RunStatusKey;
  triggerNode?: ComponentsNode;
  isSelected: boolean;
  componentIconMap: Record<string, string>;
  onSelectRun: (runId: string) => void;
}

export function RunRow({
  run,
  triggerName,
  title,
  status,
  triggerNode,
  isSelected,
  componentIconMap,
  onSelectRun,
}: RunRowProps) {
  const { organizationId, appId } = useParams<{ organizationId: string; appId: string }>();
  const iconSrc = getHeaderIconSrc(triggerNode?.component);
  const iconSlug = triggerNode?.component ? componentIconMap[triggerNode.component] : undefined;
  const currentTime = useRunningRunClock(status);
  const durationText = useMemo(() => getRunDurationText(run, status, currentTime), [run, status, currentTime]);
  const runHref = organizationId && appId && run.id ? appPath(organizationId, appId, `?run=${run.id}`) : "#";
  const selectRun = () => {
    if (run.id) onSelectRun(run.id);
  };
  const openRunInNewTab = () => {
    if (runHref === "#") return;
    window.open(runHref, "_blank", "noopener,noreferrer");
  };
  const handleTimestampClick = (event: MouseEvent<HTMLSpanElement>) => {
    if (isNormalClick(event)) {
      event.preventDefault();
      event.stopPropagation();
      selectRun();
      return;
    }

    event.stopPropagation();
    openRunInNewTab();
  };

  return (
    <div
      data-testid="runs-sidebar-row"
      className={cn(
        RUNS_SIDEBAR_RUN_ROW_CLASS,
        "group relative w-full transition-colors",
        isSelected ? "bg-sky-100 dark:bg-gray-800" : "hover:bg-slate-50 dark:hover:bg-gray-800",
      )}
    >
      <Link
        to={runHref}
        onClick={(e) => {
          if (isNormalClick(e)) {
            e.preventDefault();
            selectRun();
          }
        }}
        className="absolute inset-0 z-0"
        aria-label={title}
      />
      <RunRowTitleLine title={title} status={status} isSelected={isSelected} runHref={runHref} />
      <RunRowMetadataLine
        triggerName={triggerName}
        iconSrc={iconSrc}
        iconSlug={iconSlug}
        isSelected={isSelected}
        durationText={durationText}
        createdAt={run.createdAt}
        onTimestampClick={handleTimestampClick}
      />
    </div>
  );
}

function RunRowTitleLine({
  title,
  status,
  isSelected,
  runHref,
}: {
  title: string;
  status: RunStatusKey;
  isSelected: boolean;
  runHref: string;
}) {
  return (
    <div className="pointer-events-none relative z-10 flex min-w-0 items-center gap-2">
      <span
        className={cn(
          "min-w-0 flex-1 truncate text-[13px] leading-5",
          isSelected
            ? "font-semibold text-sky-950 dark:text-gray-50"
            : "font-semibold text-gray-900 dark:text-gray-100",
        )}
        title={title}
      >
        {title}
      </span>
      <CopyRunLinkButton runHref={runHref} />
      <RunStatusBadge status={status} />
    </div>
  );
}

function CopyRunLinkButton({ runHref }: { runHref: string }) {
  return (
    <button
      type="button"
      title="Copy link to run"
      className="pointer-events-auto hidden shrink-0 rounded p-0.5 text-gray-400 hover:bg-gray-200 hover:text-gray-600 group-hover:inline-flex dark:text-gray-500 dark:hover:bg-gray-700 dark:hover:text-gray-300"
      onClick={(event) => {
        event.stopPropagation();
        void copyRunLink(runHref);
      }}
    >
      <LinkIcon className="h-3 w-3" />
    </button>
  );
}

async function copyRunLink(runHref: string) {
  const copyUrl = new URL(runHref, window.location.origin);
  try {
    await navigator.clipboard.writeText(copyUrl.toString());
    toast.success("Run link copied");
  } catch {
    toast.error("Failed to copy run link");
  }
}

function RunStatusBadge({ status }: { status: RunStatusKey }) {
  const statusMeta = RUN_STATUS_META[status];
  const StatusIcon = statusMeta.icon;

  return (
    <span
      aria-label={statusMeta.label}
      title={statusMeta.label}
      className={cn(
        "inline-flex shrink-0 items-center gap-1 rounded px-1.5 py-0.5 text-[12px] font-medium leading-4 ring-1",
        statusMeta.badgeClassName,
      )}
    >
      <StatusIcon className="size-3.5" aria-hidden />
      <span>{statusMeta.label}</span>
    </span>
  );
}

function RunRowMetadataLine({
  triggerName,
  iconSrc,
  iconSlug,
  isSelected,
  durationText,
  createdAt,
  onTimestampClick,
}: {
  triggerName: string;
  iconSrc?: string;
  iconSlug?: string;
  isSelected: boolean;
  durationText: string | null;
  createdAt?: string;
  onTimestampClick: (event: MouseEvent<HTMLSpanElement>) => void;
}) {
  return (
    <div className="pointer-events-none relative z-10 flex min-w-0 items-center gap-2">
      <RunTriggerIdentity triggerName={triggerName} iconSrc={iconSrc} iconSlug={iconSlug} isSelected={isSelected} />
      <span className="flex min-w-0 shrink-0 items-center gap-2 text-[12px] leading-4 text-gray-500 dark:text-gray-400">
        {durationText ? <span className="shrink-0 tabular-nums">{durationText}</span> : null}
        {createdAt ? (
          <span
            className="pointer-events-auto shrink-0 tabular-nums"
            onClick={onTimestampClick}
            onAuxClick={onTimestampClick}
          >
            <Timestamp date={createdAt} display="relative" relativeStyle="abbreviated" includeAgo />
          </span>
        ) : null}
      </span>
    </div>
  );
}

function RunTriggerIdentity({
  triggerName,
  iconSrc,
  iconSlug,
  isSelected,
}: {
  triggerName: string;
  iconSrc?: string;
  iconSlug?: string;
  isSelected: boolean;
}) {
  return (
    <span className="pointer-events-none flex min-w-0 flex-1 items-center gap-1.5">
      <RunNodeIcon
        iconSrc={iconSrc}
        iconSlug={iconSlug}
        alt={triggerName}
        size={RUN_NODE_ICON_SIZE}
        className={cn(
          "h-3.5 w-3.5 shrink-0",
          isSelected ? "text-gray-800 dark:text-gray-100" : "text-gray-500 dark:text-gray-400",
        )}
      />
      <span className="min-w-0 truncate text-[12px] leading-4 text-gray-600 dark:text-gray-400" title={triggerName}>
        {triggerName}
      </span>
    </span>
  );
}

function useRunningRunClock(status: RunStatusKey) {
  const [currentTime, setCurrentTime] = useState(() => Date.now());

  useEffect(() => {
    if (status !== "running") return;

    setCurrentTime(Date.now());
    const interval = window.setInterval(() => {
      setCurrentTime(Date.now());
    }, 1000);

    return () => window.clearInterval(interval);
  }, [status]);

  return currentTime;
}

function getRunDurationText(run: CanvasesCanvasRun, status: RunStatusKey, currentTime: number) {
  const startedAt = parseTimestamp(run.createdAt);
  if (startedAt === null) return null;

  const endedAt = status === "running" ? currentTime : parseTimestamp(run.finishedAt);
  if (endedAt === null || endedAt < startedAt) return null;

  return formatMinutesSecondsDuration(endedAt - startedAt);
}

function parseTimestamp(value: string | undefined) {
  if (!value) return null;

  const timestamp = new Date(value).getTime();
  return Number.isFinite(timestamp) ? timestamp : null;
}
