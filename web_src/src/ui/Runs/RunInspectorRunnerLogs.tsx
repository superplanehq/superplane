import type { CanvasesCanvasNodeExecution } from "@/api-client";
import { buildExecutionInfo } from "@/pages/app/utils";
import { isExecutionInFlight } from "@/ui/CanvasPage/RunnerLiveLogDialog/types";
import {
  terminalCommandStatusForExecution,
  terminalTimeMsForExecution,
  useLiveLogStream,
} from "@/ui/CanvasPage/RunnerLiveLogDialog/useLiveLogStream";
import { EmptySectionText, TimelineAccordionCard } from "./RunInspectorTimelineCard";
import type { StatusPill } from "./RunInspectorTimelineTypes";
import type { RunInspectorNodeSection } from "./types";

export function RunnerLogsTimelineCard({ section, isOpen }: { section: RunInspectorNodeSection; isOpen: boolean }) {
  const execution = section.execution;

  if (!execution?.id) {
    return null;
  }

  return (
    <TimelineAccordionCard
      value="logs"
      status={runnerLogsStatus(section)}
      title="Logs"
      sourceName={section.nodeName}
      trailing={section.durationMs === undefined ? null : formatSeconds(section.durationMs)}
      actionPayload={undefined}
      jsonViewStyle={{}}
    >
      {isOpen ? (
        <RunnerLogsTerminal execution={execution} />
      ) : (
        <EmptySectionText>Open this section to load runner logs.</EmptySectionText>
      )}
    </TimelineAccordionCard>
  );
}

function RunnerLogsTerminal({ execution }: { execution: CanvasesCanvasNodeExecution }) {
  const executionInfo = buildExecutionInfo(execution);
  const executionInFlight = isExecutionInFlight(executionInfo);
  const { sections, orphanLines, error, isStreaming, scrollRef } = useLiveLogStream(
    executionInfo.id,
    executionInFlight,
    terminalCommandStatusForExecution(executionInfo),
    terminalTimeMsForExecution(executionInfo),
  );
  const lines = runnerLogLines(orphanLines, sections);
  const isWaiting = lines.length === 0 && !error && (executionInFlight || isStreaming);
  const hasError = Boolean(error) && !executionInFlight;

  if (hasError) {
    return <EmptySectionText>Something went wrong while fetching logs. Please try again later.</EmptySectionText>;
  }

  if (isWaiting) {
    return <EmptySectionText>Waiting for logs...</EmptySectionText>;
  }

  if (lines.length === 0) {
    return <EmptySectionText>No log lines yet.</EmptySectionText>;
  }

  return (
    <div
      ref={scrollRef}
      data-testid="run-inspector-runner-logs-terminal"
      className="max-h-80 overflow-y-auto rounded-sm bg-slate-950 px-4 py-3 text-slate-100 shadow-inner ring-1 ring-slate-900/10 dark:bg-black dark:ring-gray-800"
    >
      <pre className="whitespace-pre-wrap font-mono text-[12px] leading-relaxed">{lines.join("\n")}</pre>
    </div>
  );
}

function runnerLogLines(orphanLines: string[], sections: Array<{ text: string; lines: string[] }>): string[] {
  return [
    ...orphanLines.filter((line) => line.trim() !== ""),
    ...sections.flatMap((section) => [`$ ${section.text}`, ...section.lines].filter((line) => line.trim() !== "")),
  ];
}

function runnerLogsStatus(section: RunInspectorNodeSection): StatusPill {
  return {
    dotClassName: section.badge?.badgeColor ?? "bg-slate-400",
    label: section.badge?.label ?? "Logs",
    tone: section.errorMessage ? "error" : "default",
  };
}

function formatSeconds(durationMs: number): string {
  return `${Math.max(0, Math.round(durationMs / 1000))}s`;
}
