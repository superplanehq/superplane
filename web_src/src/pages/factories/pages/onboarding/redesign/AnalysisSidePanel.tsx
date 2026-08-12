import { cn } from "@/lib/utils";
import { useEffect, useRef, useState } from "react";

import type { OnboardingNavAnalyzingKey } from "../onboardingStorybookContextValue";
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

type Milestone = "boot" | "name" | "repo" | "issues" | "agent";

function linesForMilestone(milestone: Milestone, progress: AnalysisProgress): string[] {
  switch (milestone) {
    case "boot":
      return ["waiting for workspace setup…"];
    case "name":
      return [`workspace registered: ${progress.workspaceName.trim()}`];
    case "repo": {
      const host = progress.vcsHost ? vcsLabel(progress.vcsHost) : "git";
      const repo = progress.selectedRepo ?? "app repository";
      return [
        `${host} connected`,
        `analyzing app repository ${repo}`,
        "cloning repository…",
        "indexing codebase in background…",
      ];
    }
    case "issues":
      if (progress.issuesChoice === "skip") {
        return ["backlog import skipped - create work orders manually"];
      }
      if (progress.issuesChoice === "vcs" && progress.vcsHost) {
        return [`backlog: ${vcsLabel(progress.vcsHost)} Issues`, "scoring open issues for agent work…"];
      }
      if (progress.issuesChoice === "linear") {
        return ["backlog: Linear", "scoring Linear issues for agent work…"];
      }
      if (progress.issuesChoice === "jira") {
        return ["backlog: Jira", "scoring Jira issues for agent work…"];
      }
      return [];
    case "agent":
      return [
        `coding agent: ${progress.agent ?? "configured"}`,
        "credentials verified",
        "ready to hand off the first work order",
      ];
  }
}

function asNavAnalyzingKey(milestone: Milestone): OnboardingNavAnalyzingKey | null {
  if (milestone === "repo" || milestone === "issues" || milestone === "agent") return milestone;
  return null;
}

/**
 * Persistent setup.log side panel. Advances as the user completes onboarding steps.
 * Velocity / AI readiness / Knowledge results are intentionally omitted for now.
 */
export function AnalysisSidePanel({
  progress,
  onNavAnalyzing,
}: {
  progress: AnalysisProgress;
  onNavAnalyzing?: (key: OnboardingNavAnalyzingKey, analyzing: boolean) => void;
}) {
  const [logLines, setLogLines] = useState<string[]>(() => linesForMilestone("boot", progress));
  const [pendingWrites, setPendingWrites] = useState(0);
  /** Milestones whose log lines have fully written. */
  const completedRef = useRef<Set<Milestone>>(new Set(["boot"]));
  /** Milestones currently scheduled (may be cancelled by effect cleanup / Strict Mode). */
  const inFlightRef = useRef<Set<Milestone>>(new Set());
  const terminalRef = useRef<HTMLOListElement>(null);
  const onNavAnalyzingRef = useRef(onNavAnalyzing);
  onNavAnalyzingRef.current = onNavAnalyzing;

  const complete = progress.nameReady && progress.repoCommitted && progress.agentReady;
  const statusLabel =
    complete && pendingWrites === 0 ? "idle" : progress.nameReady || pendingWrites > 0 ? "running" : "idle";

  // Allow re-analysis after the user changes a selection and Continues again.
  useEffect(() => {
    if (!progress.repoCommitted) {
      completedRef.current.delete("repo");
      inFlightRef.current.delete("repo");
    }
  }, [progress.repoCommitted, progress.selectedRepo]);

  useEffect(() => {
    if (!progress.issuesCommitted) {
      completedRef.current.delete("issues");
      inFlightRef.current.delete("issues");
    }
  }, [progress.issuesCommitted, progress.issuesChoice]);

  useEffect(() => {
    const next: Milestone[] = [];
    const consider = (milestone: Milestone, ready: boolean) => {
      if (!ready) return;
      if (completedRef.current.has(milestone) || inFlightRef.current.has(milestone)) return;
      next.push(milestone);
    };
    consider("name", progress.nameReady);
    consider("repo", progress.repoCommitted && progress.selectedRepo !== null);
    consider("issues", progress.issuesCommitted && progress.issuesChoice !== null);
    consider("agent", progress.agentReady);
    if (next.length === 0) return;

    next.forEach((milestone) => inFlightRef.current.add(milestone));

    const timers: number[] = [];
    let delay = 0;
    let scheduled = 0;
    next.forEach((milestone) => {
      const lines = linesForMilestone(milestone, progress);
      const navKey = asNavAnalyzingKey(milestone);
      if (lines.length === 0) {
        inFlightRef.current.delete(milestone);
        completedRef.current.add(milestone);
        if (navKey) onNavAnalyzingRef.current?.(navKey, false);
        return;
      }
      lines.forEach((line, lineIndex) => {
        scheduled += 1;
        delay += 350;
        const isLast = lineIndex === lines.length - 1;
        timers.push(
          window.setTimeout(() => {
            setLogLines((current) => [...current, line]);
            setPendingWrites((count) => Math.max(0, count - 1));
            if (isLast) {
              inFlightRef.current.delete(milestone);
              completedRef.current.add(milestone);
              if (navKey) onNavAnalyzingRef.current?.(navKey, false);
            }
          }, delay),
        );
      });
    });
    if (scheduled > 0) {
      setPendingWrites((count) => count + scheduled);
    }

    return () => {
      timers.forEach((id) => window.clearTimeout(id));
      if (scheduled > 0) {
        setPendingWrites((count) => Math.max(0, count - scheduled));
      }
      // Allow re-schedule after Strict Mode / dep cleanup. Do not mark completed.
      next.forEach((milestone) => {
        inFlightRef.current.delete(milestone);
      });
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps -- milestone fields only; full `progress` identity changes every render
  }, [
    progress.nameReady,
    progress.repoCommitted,
    progress.issuesCommitted,
    progress.issuesChoice,
    progress.agentReady,
    progress.workspaceName,
    progress.selectedRepo,
    progress.vcsHost,
    progress.agent,
  ]);

  useEffect(() => {
    const el = terminalRef.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, [logLines, statusLabel]);

  const subtitle = complete
    ? "Ready for the first work order."
    : progress.nameReady
      ? "SuperPlane analyzes the app and backlog as you finish each section."
      : "Shows analysis progress while you set up this workspace.";

  return (
    <aside
      className="flex h-full min-h-[520px] flex-col overflow-hidden rounded-lg border border-border bg-background"
      data-testid="onboarding-analysis-panel"
    >
      <div className="border-b border-border px-4 py-3">
        <div className="text-[13px] font-medium tracking-[-0.02em]">Setup log</div>
        <p className="mt-0.5 text-[12px] text-muted-foreground">{subtitle}</p>
      </div>

      <div className="flex min-h-0 flex-1 flex-col bg-zinc-950 text-zinc-100">
        <header className="flex items-center gap-2 border-b border-zinc-800 px-3 py-2 text-[11px]">
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
        <ol
          ref={terminalRef}
          className="min-h-0 flex-1 space-y-1 overflow-y-auto px-3 py-3 font-mono text-[11px] leading-5"
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
