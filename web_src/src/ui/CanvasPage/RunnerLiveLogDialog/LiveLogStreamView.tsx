import type { CommandSection } from "./types";
import { useLiveLogStream } from "./useLiveLogStream";

function formatDuration(durationMs: number): string {
  if (durationMs < 1000) {
    return `${(durationMs / 1000).toFixed(2)}s`;
  }
  if (durationMs >= 60_000) {
    const minutes = Math.floor(durationMs / 60_000);
    const seconds = Math.floor((durationMs % 60_000) / 1000);
    return `${minutes}m ${seconds.toString().padStart(2, "0")}s`;
  }
  return `${Math.floor(durationMs / 1000)}s`;
}

function CommandSectionView({
  section,
  onToggle,
}: {
  section: CommandSection;
  onToggle: (index: number) => void;
}) {
  const isCollapsed = section.status === "passed" && section.collapsed;

  return (
    <div className="border-t border-slate-950/20 first:border-t-0">
      <button
        type="button"
        className="flex w-full cursor-pointer items-center gap-2 bg-slate-800 px-4 py-2 font-mono text-left text-xs text-white"
        onClick={() => onToggle(section.index)}
      >
        <span>{isCollapsed ? "▶" : "▼"}</span>
        <span className="text-green-400">$</span>
        <span>{section.text}</span>
        {section.status === "running" ? <span className="ml-auto text-xs text-gray-300">Running...</span> : null}
        {section.status === "passed" && section.duration_ms !== null ? (
          <span className="ml-auto text-xs text-green-400">Passed in {formatDuration(section.duration_ms)}</span>
        ) : null}
        {section.status === "failed" && section.duration_ms !== null ? (
          <span className="ml-auto text-xs text-red-400">Failed in {formatDuration(section.duration_ms)}</span>
        ) : null}
      </button>
      {!isCollapsed ? (
        <pre className="overflow-auto bg-slate-900 px-4 py-2 text-left font-mono text-xs leading-relaxed whitespace-pre-wrap text-gray-300">
          {section.lines.join("")}
        </pre>
      ) : null}
    </div>
  );
}

export function LiveLogStreamView({ executionId }: { executionId: string }) {
  const { sections, orphanLines, error, isStreaming, toggleSection, scrollRef } = useLiveLogStream(executionId);
  const hasAnyLogs = orphanLines.length > 0 || sections.length > 0;

  return (
    <div className="flex min-h-[50vh] flex-col overflow-hidden bg-slate-50">
      <div ref={scrollRef} className="min-h-0 flex-1 overflow-auto">
        {error ? <div className="px-4 py-3 text-left text-destructive">{error}</div> : null}
        {!error && !hasAnyLogs && !isStreaming ? <div className="p-4 text-left text-muted-foreground">No log lines yet.</div> : null}
        {!error && !hasAnyLogs && isStreaming ? <div className="p-4 text-left text-muted-foreground">Connecting…</div> : null}
        {orphanLines.length > 0 ? (
          <pre className="overflow-auto bg-slate-900 px-4 py-2 text-left font-mono text-xs leading-relaxed whitespace-pre-wrap text-gray-300">
            {orphanLines.join("")}
          </pre>
        ) : null}
        {sections.map((section) => (
          <CommandSectionView key={`${section.index}-${section.text}`} section={section} onToggle={toggleSection} />
        ))}
      </div>
    </div>
  );
}
