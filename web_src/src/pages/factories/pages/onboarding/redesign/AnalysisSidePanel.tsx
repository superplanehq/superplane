import { cn } from "@/lib/utils";
import { Check, Circle, Loader2, Minus } from "lucide-react";
import { useEffect, useRef, useState, type Dispatch, type MutableRefObject, type SetStateAction } from "react";

import type { AgentHarnessId, IssuesChoiceId, VcsHostId } from "./redesignFixtures";
import { vcsLabel } from "./redesignFixtures";

export type AnalysisProgress = {
  workspaceName: string;
  nameReady: boolean;
  selectedRepo: string | null;
  vcsHost: VcsHostId | null;
  /** Repository analysis starts only after Continue to issues. */
  repoCommitted: boolean;
  issuesChoice: IssuesChoiceId | null;
  /** Backlog analysis starts only after Continue to coding agent. */
  issuesCommitted: boolean;
  agent: AgentHarnessId | null;
  agentReady: boolean;
};

type SetupTaskId = "wiki" | "velocity" | "ai-readiness" | "issues" | "lines";
type TaskStatus = "pending" | "running" | "completed" | "skipped";
type SetupGroupId = "boot" | "name" | "repo" | "issues" | "lines";

type SetupTask = {
  id: SetupTaskId;
  label: string;
  optional?: boolean;
};

const SETUP_TASKS: SetupTask[] = [
  { id: "wiki", label: "Generate wiki from the app repository" },
  { id: "velocity", label: "Get initial velocity metrics" },
  { id: "ai-readiness", label: "Analyze AI readiness" },
  { id: "issues", label: "Review issues for candidate work orders", optional: true },
  { id: "lines", label: "Set up lines and automations" },
];

const EMPTY_TASK_STATUS: Record<SetupTaskId, TaskStatus> = {
  wiki: "pending",
  velocity: "pending",
  "ai-readiness": "pending",
  issues: "pending",
  lines: "pending",
};

type LogBatch = {
  taskId: SetupTaskId;
  lines: string[];
  finishStatus?: "completed" | "skipped";
  /** Delay between log lines for this batch (longer = stays running across later steps). */
  lineIntervalMs?: number;
};

function repoPreface(vcsHost: VcsHostId | null, selectedRepo: string | null): string[] {
  const host = vcsHost ? vcsLabel(vcsHost) : "git";
  const repo = selectedRepo ?? "app repository";
  return [`${host} connected`, `analyzing app repository ${repo}`];
}

/**
 * Wiki, velocity, and AI readiness start together after Continue to issues.
 * Durations differ on purpose: wiki stays running while the user can finish
 * Issues and start later setup work.
 */
function repoBatches(): LogBatch[] {
  return [
    {
      taskId: "wiki",
      lineIntervalMs: 2200,
      lines: [
        "building workspace wiki…",
        "indexing docs and architecture…",
        "mapping components and dependencies…",
        "writing wiki draft…",
        "wiki draft ready",
      ],
    },
    {
      taskId: "velocity",
      lineIntervalMs: 1100,
      lines: ["collecting delivery signals…", "computing initial velocity metrics…", "velocity baseline ready"],
    },
    {
      taskId: "ai-readiness",
      lineIntervalMs: 900,
      lines: ["scoring repository for agent work…", "AI readiness summary ready"],
    },
  ];
}

function issuesBatch(issuesChoice: IssuesChoiceId | null, vcsHost: VcsHostId | null): LogBatch {
  if (issuesChoice === "skip") {
    return {
      taskId: "issues",
      finishStatus: "skipped",
      lines: ["backlog import skipped - create work orders manually"],
    };
  }
  if (issuesChoice === "vcs" && vcsHost) {
    return {
      taskId: "issues",
      lines: [
        `backlog: ${vcsLabel(vcsHost)} Issues`,
        "scoring open issues for agent work…",
        "candidate work orders drafted",
      ],
    };
  }
  if (issuesChoice === "linear") {
    return {
      taskId: "issues",
      lines: ["backlog: Linear", "scoring Linear issues for agent work…", "candidate work orders drafted"],
    };
  }
  if (issuesChoice === "jira") {
    return {
      taskId: "issues",
      lines: ["backlog: Jira", "scoring Jira issues for agent work…", "candidate work orders drafted"],
    };
  }
  return { taskId: "issues", lines: ["reviewing backlog for candidate work orders…"] };
}

function linesBatch(agent: AgentHarnessId | null): LogBatch {
  return {
    taskId: "lines",
    lines: [
      `coding agent: ${agent ?? "configured"}`,
      "credentials verified",
      "configuring lines and automations…",
      "ready to hand off the first work order",
    ],
  };
}

function panelSubtitle(complete: boolean, nameReady: boolean): string {
  if (complete) {
    return "Ready for the first work order.";
  }
  if (nameReady) {
    return "SuperPlane runs these tasks as you finish each section.";
  }
  return "Shows setup tasks while you configure this workspace.";
}

function TaskStatusIcon({ status }: { status: TaskStatus }) {
  if (status === "completed") {
    return (
      <span className="flex size-3.5 items-center justify-center rounded-full bg-emerald-500/20 text-emerald-400">
        <Check className="size-2.5" strokeWidth={3} aria-hidden />
      </span>
    );
  }
  if (status === "skipped") {
    return (
      <span className="flex size-3.5 items-center justify-center rounded-full bg-zinc-800 text-zinc-500">
        <Minus className="size-2.5" strokeWidth={3} aria-hidden />
      </span>
    );
  }
  if (status === "running") {
    return <Loader2 className="size-3.5 animate-spin text-emerald-400" aria-hidden />;
  }
  return <Circle className="size-3.5 text-zinc-600" strokeWidth={1.75} aria-hidden />;
}

type ScheduleGroupArgs = {
  groupId: SetupGroupId;
  ready: boolean;
  preface?: string[];
  batches: LogBatch[];
  parallel?: boolean;
  completedGroupsRef: MutableRefObject<Set<SetupGroupId>>;
  inFlightGroupsRef: MutableRefObject<Set<SetupGroupId>>;
  setLogLines: Dispatch<SetStateAction<string[]>>;
  setTaskStatus: Dispatch<SetStateAction<Record<SetupTaskId, TaskStatus>>>;
  setPendingWrites: Dispatch<SetStateAction<number>>;
};

/**
 * Schedule one setup group. Kept in its own effect so later Continues do not
 * cancel or restart earlier groups that are still running.
 */
function scheduleSetupGroup({
  groupId,
  ready,
  preface = [],
  batches,
  parallel = false,
  completedGroupsRef,
  inFlightGroupsRef,
  setLogLines,
  setTaskStatus,
  setPendingWrites,
}: ScheduleGroupArgs): (() => void) | undefined {
  if (!ready) return undefined;
  if (completedGroupsRef.current.has(groupId) || inFlightGroupsRef.current.has(groupId)) {
    return undefined;
  }

  inFlightGroupsRef.current.add(groupId);

  const timers: number[] = [];
  let delay = 0;
  let scheduled = 0;

  const appendLine = (line: string, at: number, onDone?: () => void) => {
    scheduled += 1;
    timers.push(
      window.setTimeout(() => {
        setLogLines((current) => [...current, line]);
        setPendingWrites((count) => Math.max(0, count - 1));
        onDone?.();
      }, at),
    );
  };

  preface.forEach((line) => {
    delay += 280;
    appendLine(line, delay);
  });

  if (parallel && batches.length > 0) {
    const parallelStart = delay + 40;
    timers.push(
      window.setTimeout(() => {
        setTaskStatus((current) => {
          const nextStatus = { ...current };
          batches.forEach((batch) => {
            nextStatus[batch.taskId] = "running";
          });
          return nextStatus;
        });
      }, parallelStart),
    );

    let groupEnd = parallelStart;
    batches.forEach((batch, batchIndex) => {
      const interval = batch.lineIntervalMs ?? 300;
      let batchAt = parallelStart + 80 + batchIndex * 70;
      batch.lines.forEach((line, lineIndex) => {
        batchAt += interval;
        const isLastLine = lineIndex === batch.lines.length - 1;
        appendLine(line, batchAt, () => {
          if (isLastLine) {
            setTaskStatus((current) => ({
              ...current,
              [batch.taskId]: batch.finishStatus ?? "completed",
            }));
          }
        });
      });
      groupEnd = Math.max(groupEnd, batchAt);
    });

    timers.push(
      window.setTimeout(() => {
        inFlightGroupsRef.current.delete(groupId);
        completedGroupsRef.current.add(groupId);
      }, groupEnd + 10),
    );
  } else {
    batches.forEach((batch) => {
      if (batch.finishStatus !== "skipped") {
        timers.push(
          window.setTimeout(() => {
            setTaskStatus((current) => ({ ...current, [batch.taskId]: "running" }));
          }, delay + 40),
        );
      }

      batch.lines.forEach((line, lineIndex) => {
        delay += 320;
        const isLastLine = lineIndex === batch.lines.length - 1;
        appendLine(line, delay, () => {
          if (isLastLine) {
            setTaskStatus((current) => ({
              ...current,
              [batch.taskId]: batch.finishStatus ?? "completed",
            }));
          }
        });
      });
    });

    timers.push(
      window.setTimeout(() => {
        inFlightGroupsRef.current.delete(groupId);
        completedGroupsRef.current.add(groupId);
      }, delay + 10),
    );
  }

  if (scheduled > 0) {
    setPendingWrites((count) => count + scheduled);
  }

  return () => {
    timers.forEach((id) => window.clearTimeout(id));
    if (scheduled > 0) {
      setPendingWrites((count) => Math.max(0, count - scheduled));
    }
    // Strict Mode: allow this group to re-schedule. Do not mark completed.
    inFlightGroupsRef.current.delete(groupId);
  };
}

/**
 * Setup side panel: task checklist (pending / running / completed / skipped)
 * above a streaming setup.log.
 *
 * Each onboarding Continue schedules its own group in a separate effect so later
 * steps do not restart wiki / velocity / AI readiness that are still running.
 */
export function AnalysisSidePanel({ progress }: { progress: AnalysisProgress }) {
  const [logLines, setLogLines] = useState<string[]>(["waiting for workspace setup…"]);
  const [taskStatus, setTaskStatus] = useState<Record<SetupTaskId, TaskStatus>>(EMPTY_TASK_STATUS);
  const [pendingWrites, setPendingWrites] = useState(0);
  const completedGroupsRef = useRef<Set<SetupGroupId>>(new Set(["boot"]));
  const inFlightGroupsRef = useRef<Set<SetupGroupId>>(new Set());
  const terminalRef = useRef<HTMLOListElement>(null);

  const complete = progress.nameReady && progress.repoCommitted && progress.agentReady && pendingWrites === 0;
  const statusLabel = complete ? "idle" : progress.nameReady || pendingWrites > 0 ? "running" : "idle";

  useEffect(() => {
    if (!progress.repoCommitted) {
      completedGroupsRef.current.delete("repo");
      inFlightGroupsRef.current.delete("repo");
      setTaskStatus((current) => ({
        ...current,
        wiki: "pending",
        velocity: "pending",
        "ai-readiness": "pending",
      }));
    }
  }, [progress.repoCommitted, progress.selectedRepo]);

  useEffect(() => {
    if (!progress.issuesCommitted) {
      completedGroupsRef.current.delete("issues");
      inFlightGroupsRef.current.delete("issues");
      setTaskStatus((current) => ({ ...current, issues: "pending" }));
    }
  }, [progress.issuesCommitted, progress.issuesChoice]);

  useEffect(() => {
    if (!progress.agentReady) {
      completedGroupsRef.current.delete("lines");
      inFlightGroupsRef.current.delete("lines");
      setTaskStatus((current) => ({ ...current, lines: "pending" }));
    }
  }, [progress.agentReady, progress.agent]);

  useEffect(() => {
    return scheduleSetupGroup({
      groupId: "name",
      ready: progress.nameReady,
      preface: [`workspace registered: ${progress.workspaceName.trim()}`],
      batches: [],
      completedGroupsRef,
      inFlightGroupsRef,
      setLogLines,
      setTaskStatus,
      setPendingWrites,
    });
  }, [progress.nameReady, progress.workspaceName]);

  useEffect(() => {
    return scheduleSetupGroup({
      groupId: "repo",
      ready: progress.repoCommitted && progress.selectedRepo !== null,
      preface: repoPreface(progress.vcsHost, progress.selectedRepo),
      batches: repoBatches(),
      parallel: true,
      completedGroupsRef,
      inFlightGroupsRef,
      setLogLines,
      setTaskStatus,
      setPendingWrites,
    });
  }, [progress.repoCommitted, progress.selectedRepo, progress.vcsHost]);

  useEffect(() => {
    return scheduleSetupGroup({
      groupId: "issues",
      ready: progress.issuesCommitted && progress.issuesChoice !== null,
      batches: [issuesBatch(progress.issuesChoice, progress.vcsHost)],
      completedGroupsRef,
      inFlightGroupsRef,
      setLogLines,
      setTaskStatus,
      setPendingWrites,
    });
  }, [progress.issuesCommitted, progress.issuesChoice, progress.vcsHost]);

  useEffect(() => {
    return scheduleSetupGroup({
      groupId: "lines",
      ready: progress.agentReady,
      batches: [linesBatch(progress.agent)],
      completedGroupsRef,
      inFlightGroupsRef,
      setLogLines,
      setTaskStatus,
      setPendingWrites,
    });
  }, [progress.agentReady, progress.agent]);

  useEffect(() => {
    const el = terminalRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [logLines, statusLabel]);

  const subtitle = panelSubtitle(complete, progress.nameReady);

  return (
    <aside
      className="flex h-[min(720px,calc(100vh-5.5rem))] flex-col overflow-hidden rounded-lg border border-border bg-background"
      data-testid="onboarding-analysis-panel"
    >
      <div className="shrink-0 border-b border-border px-4 py-3">
        <div className="text-[13px] font-medium tracking-[-0.02em]">Setup log</div>
        <p className="mt-0.5 text-[12px] text-muted-foreground">{subtitle}</p>
      </div>

      <div className="flex min-h-0 flex-1 flex-col bg-zinc-950 text-zinc-100">
        <header className="flex shrink-0 items-center gap-2 border-b border-zinc-800 px-3 py-2 text-[11px]">
          <span className="flex gap-1" aria-hidden>
            <i className="size-2 rounded-full bg-zinc-600" />
            <i className="size-2 rounded-full bg-zinc-600" />
            <i className="size-2 rounded-full bg-zinc-600" />
          </span>
          <span className="font-mono text-zinc-300">setup.log</span>
          <span className={cn("ml-auto font-mono", statusLabel === "running" ? "text-emerald-400" : "text-zinc-500")}>
            {statusLabel}
          </span>
        </header>

        <ol className="shrink-0 space-y-1.5 border-b border-zinc-800 px-3 py-3" aria-label="Setup tasks">
          {SETUP_TASKS.map((task) => {
            const status = taskStatus[task.id];
            return (
              <li
                key={task.id}
                className={cn(
                  "flex items-start gap-2 font-mono text-[11px] leading-4",
                  status === "pending" && "text-zinc-500",
                  status === "running" && "text-zinc-100",
                  status === "completed" && "text-zinc-300",
                  status === "skipped" && "text-zinc-600",
                )}
                data-status={status}
              >
                <span className="mt-0.5 shrink-0">
                  <TaskStatusIcon status={status} />
                </span>
                <span className={cn(status === "skipped" && "line-through")}>
                  {task.label}
                  {task.optional && status === "pending" ? (
                    <span className="text-zinc-600 no-underline"> · optional</span>
                  ) : null}
                </span>
              </li>
            );
          })}
        </ol>

        <ol
          ref={terminalRef}
          className="min-h-0 flex-1 space-y-1 overflow-y-auto overscroll-contain px-3 py-3 font-mono text-[11px] leading-5"
          aria-label="Setup log"
        >
          {logLines.map((line, index) => (
            <li key={`${index}-${line}`}>
              <span className="text-emerald-500/80" aria-hidden>
                ›{" "}
              </span>
              {line}
            </li>
          ))}
          {statusLabel === "running" ? (
            <li className="text-emerald-500/80" aria-hidden>
              › <span className="inline-block h-3 w-1.5 animate-pulse bg-emerald-400 align-middle" />
            </li>
          ) : null}
        </ol>
      </div>
    </aside>
  );
}
