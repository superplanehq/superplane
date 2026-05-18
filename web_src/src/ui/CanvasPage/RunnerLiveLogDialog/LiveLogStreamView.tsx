import { ChevronDown, ChevronRight } from "lucide-react";
import { cn } from "../../../lib/utils";
import type { CommandSection } from "./types";
import { useLiveLogStream } from "./useLiveLogStream";

export function LiveLogStreamView({ executionId }: { executionId: string }) {
  const { sections, orphanLines, error, toggleSection, scrollRef } = useLiveLogStream(executionId);
  const hasAnyLogs = orphanLines.length > 0 || sections.length > 0;

  return (
    <div className="flex flex-col overflow-hidden bg-store-50">
      <div ref={scrollRef} className="min-h-0 flex-1 overflow-auto">
        {error ? <ErrorMessage /> : null}
        {!error && !hasAnyLogs ? <NoLogsMessage /> : null}

        {sections.map((section) => (
          <CommandSectionView key={`${section.index}-${section.text}`} section={section} onToggle={toggleSection} />
        ))}
      </div>
    </div>
  );
}

function NoLogsMessage() {
  return <div className="px-4 py-3 text-left text-muted-foreground">No log lines yet.</div>;
}

function ErrorMessage() {
  return (
    <div className="px-4 py-3 text-left text-destructive">
      Something went wrong while fetching logs. Please try again later.
    </div>
  );
}

function CommandSectionView({ section, onToggle }: { section: CommandSection; onToggle: (index: number) => void }) {
  return (
    <div className="border-b border-slate-200">
      <CommandSectionHeader section={section} onToggle={onToggle} />
      <CommandSectionContent section={section} />
    </div>
  );
}

function CommandSectionHeader({ section, onToggle }: { section: CommandSection; onToggle: (index: number) => void }) {
  const isCollapsed = section.status === "passed" && section.collapsed;

  return (
    <button
      type="button"
      className="flex w-full cursor-pointer items-center justify-between gap-2 px-4 py-2 font-mono text-left text-xs"
      onClick={() => onToggle(section.index)}
    >
      <div className="flex items-center gap-2">
        <span>{isCollapsed ? <ChevronRight className="size-4" /> : <ChevronDown className="size-4" />}</span>
        <span className="font-semibold">{section.text}</span>
      </div>

      <div className="flex items-center gap-2">
        <Duration duration={section.duration_ms ?? 0} />
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
    <pre className="overflow-auto px-4 py-2 text-left font-mono text-xs leading-relaxed whitespace-pre-wrap text-gray-800 bg-white border-t border-slate-200">
      {section.lines.filter((line) => line.trim() !== "").join("\n")}
    </pre>
  );
}

function Duration({ duration }: { duration: number }) {
  return <div className="flex items-center gap-1">{formatDuration(duration)}</div>;
}

function StatusBadge({ status }: { status: CommandSection["status"] }) {
  const klass = cn({
    "uppercase text-[11px] py-[1.5px] px-[5px] font-semibold rounded flex items-center tracking-wide justify-center text-white": true,
    "bg-emerald-500": status === "passed",
    "bg-red-500": status === "failed",
    "bg-blue-500": status === "running",
  });

  const label = status === "passed" ? "Passed" : "Failed";

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
