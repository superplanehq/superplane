import { ChevronDown, ChevronRight } from "lucide-react";
import { useEffect, useState } from "react";
import { cn } from "../../../lib/utils";
import type { ExecutionInfo } from "../../../pages/app/mappers/types";
import { isExecutionInFlight, type CommandSection } from "./types";
import { useLiveLogStream } from "./useLiveLogStream";

export function LiveLogStreamView({ execution }: { execution: ExecutionInfo }) {
  const executionInFlight = isExecutionInFlight(execution);
  const { sections, orphanLines, error, toggleSection, scrollRef } = useLiveLogStream(execution.id, executionInFlight);
  const hasAnyLogs = orphanLines.length > 0 || sections.length > 0;
  const lastSectionIndex = sections.length - 1;
  const waitingForLogs = executionInFlight && !hasAnyLogs && !error;
  const showError = Boolean(error) && !executionInFlight;

  return (
    <div ref={scrollRef} className="h-full min-h-0 overflow-y-auto bg-slate-50">
      {showError ? <ErrorMessage /> : null}
      {waitingForLogs ? <WaitingForLogsMessage /> : null}
      {!showError && !waitingForLogs && !hasAnyLogs ? <NoLogsMessage /> : null}

      {sections.map((section, index) => (
        <CommandSectionView
          key={`${section.index}-${section.text}`}
          section={section}
          onToggle={toggleSection}
          isLast={index === lastSectionIndex}
        />
      ))}
    </div>
  );
}

function NoLogsMessage() {
  return <div className="px-4 py-3 text-left text-muted-foreground">No log lines yet.</div>;
}

function WaitingForLogsMessage() {
  return <div className="px-4 py-3 text-left text-muted-foreground">Waiting for logs…</div>;
}

function ErrorMessage() {
  return (
    <div className="px-4 py-3 text-left text-destructive">
      Something went wrong while fetching logs. Please try again later.
    </div>
  );
}

function CommandSectionView({
  section,
  onToggle,
  isLast,
}: {
  section: CommandSection;
  onToggle: (index: number) => void;
  isLast: boolean;
}) {
  return (
    <div className="border-b border-slate-200">
      <CommandSectionHeader section={section} onToggle={onToggle} isLast={isLast} />
      <CommandSectionContent section={section} />
    </div>
  );
}

function CommandSectionHeader({
  section,
  onToggle,
  isLast,
}: {
  section: CommandSection;
  onToggle: (index: number) => void;
  isLast: boolean;
}) {
  const isCollapsed = section.status === "passed" && section.collapsed;

  const openChevron = <ChevronRight className="size-4" />;
  const closedChevron = <ChevronDown className="size-4" />;

  return (
    <button
      type="button"
      className={cn(
        "flex w-full cursor-pointer items-center justify-between gap-2 px-4 py-2 font-mono text-left text-xs",
        isLast && "sticky top-0 z-10 border-b border-slate-200 bg-slate-50",
      )}
      onClick={() => onToggle(section.index)}
    >
      <div className="flex items-center gap-2">
        <span>{isCollapsed ? openChevron : closedChevron}</span>
        <span className="font-medium text-gray-900">{section.text}</span>
      </div>

      <div className="flex items-center gap-2">
        <Duration status={section.status} durationMs={section.duration_ms} startedAt={section.started_at} />
        <StatusBadge status={section.status} />
      </div>
    </button>
  );
}

function CommandSectionContent({ section }: { section: CommandSection }) {
  if (section.collapsed) {
    return null;
  }

  return (
    <pre className="px-4 py-2 text-left font-mono text-xs leading-relaxed whitespace-pre-wrap text-gray-800 bg-white border-t border-slate-200">
      {section.lines.filter((line) => line.trim() !== "").join("\n")}
    </pre>
  );
}

function Duration({
  status,
  durationMs,
  startedAt,
}: {
  status: CommandSection["status"];
  durationMs: number | null;
  startedAt: number | null;
}) {
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    if (status !== "running" || startedAt === null) {
      return;
    }

    setNow(Date.now());
    const interval = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(interval);
  }, [status, startedAt]);

  const duration = status === "running" && startedAt !== null ? Math.max(0, now - startedAt) : (durationMs ?? 0);

  return <div className="flex items-center gap-1">{formatDuration(duration)}</div>;
}

function StatusBadge({ status }: { status: CommandSection["status"] }) {
  const klass = cn({
    "uppercase text-[11px] py-[1.5px] px-[5px] font-semibold rounded flex items-center tracking-wide justify-center text-white": true,
    "bg-emerald-500": status === "passed",
    "bg-red-500": status === "failed",
    "bg-blue-500": status === "running",
  });

  let label = "";
  switch (status) {
    case "passed":
      label = "Passed";
      break;
    case "failed":
      label = "Failed";
      break;
    default:
      label = "Running";
      break;
  }

  return (
    <div className={klass}>
      <span>{label}</span>
    </div>
  );
}

function formatDuration(durationMs: number): string {
  // Less than 1 second
  if (durationMs < 1000) {
    return `00:00`;
  }

  // Less than 1 minute
  if (durationMs < 60_000) {
    const seconds = Math.floor(durationMs / 1000);
    return `00:${seconds.toString().padStart(2, "0")}`;
  }

  // Less than 1 hour
  if (durationMs < 3_600_000) {
    const minutes = Math.floor(durationMs / 60_000);
    const seconds = Math.floor((durationMs % 60_000) / 1000);
    return `${minutes.toString().padStart(2, "0")}:${seconds.toString().padStart(2, "0")}`;
  }

  // More than 1 hour
  const hours = Math.floor(durationMs / 3_600_000);
  const minutes = Math.floor((durationMs % 3_600_000) / 60_000);
  const seconds = Math.floor((durationMs % 60_000) / 1000);
  return `${hours.toString().padStart(2, "0")}:${minutes.toString().padStart(2, "0")}:${seconds.toString().padStart(2, "0")}`;
}
