import { cn } from "@/lib/utils";
import { useEffect, useRef, useState } from "react";

import type { AgentHarnessId, IssuesChoiceId, VcsHostId } from "./redesignFixtures";
import { vcsLabel } from "./redesignFixtures";

export type AnalysisProgress = {
  workspaceName: string;
  nameReady: boolean;
  selectedRepo: string | null;
  vcsHost: VcsHostId | null;
  repoReady: boolean;
  issuesChoice: IssuesChoiceId | null;
  agent: AgentHarnessId | null;
  agentReady: boolean;
};

type Milestone = "boot" | "name" | "repo" | "issues" | "agent";

function linesForMilestone(milestone: Milestone, progress: AnalysisProgress): string[] {
  switch (milestone) {
    case "boot":
      return ["setup worker online", "waiting for workspace configuration…"];
    case "name":
      return [`workspace registered: ${progress.workspaceName.trim()}`];
    case "repo": {
      const host = progress.vcsHost ? vcsLabel(progress.vcsHost) : "git";
      const repo = progress.selectedRepo ?? "repository";
      return [
        `${host} connected`,
        `queued analysis for ${repo}`,
        "cloning repository…",
        "indexing history in background…",
      ];
    }
    case "issues":
      if (progress.issuesChoice === "skip") {
        return ["issues source skipped - work orders will be created manually"];
      }
      if (progress.issuesChoice === "vcs" && progress.vcsHost) {
        return [`issues source: ${vcsLabel(progress.vcsHost)} Issues`, "scanning open issues…"];
      }
      if (progress.issuesChoice === "linear") {
        return ["issues source: Linear", "syncing Linear projects…"];
      }
      if (progress.issuesChoice === "jira") {
        return ["issues source: Jira", "syncing Jira projects…"];
      }
      return [];
    case "agent":
      return [
        `coding agent: ${progress.agent ?? "configured"}`,
        "credentials verified",
        "workspace ready for first work order",
      ];
  }
}

/**
 * Persistent setup.log side panel. Advances as the user completes onboarding steps.
 * Velocity / AI readiness / Knowledge results are intentionally omitted for now.
 */
export function AnalysisSidePanel({ progress }: { progress: AnalysisProgress }) {
  const [logLines, setLogLines] = useState<string[]>(() => linesForMilestone("boot", progress));
  const [pendingWrites, setPendingWrites] = useState(0);
  const seenRef = useRef<Set<Milestone>>(new Set(["boot"]));
  const terminalRef = useRef<HTMLOListElement>(null);

  const complete = progress.nameReady && progress.repoReady && progress.agentReady;
  const statusLabel =
    complete && pendingWrites === 0 ? "idle" : progress.nameReady || pendingWrites > 0 ? "running" : "idle";

  useEffect(() => {
    const next: Milestone[] = [];
    if (progress.nameReady && !seenRef.current.has("name")) next.push("name");
    if (progress.repoReady && !seenRef.current.has("repo")) next.push("repo");
    if (progress.issuesChoice !== null && !seenRef.current.has("issues")) next.push("issues");
    if (progress.agentReady && !seenRef.current.has("agent")) next.push("agent");
    if (next.length === 0) return;

    next.forEach((milestone) => seenRef.current.add(milestone));

    const timers: number[] = [];
    let delay = 0;
    let scheduled = 0;
    next.forEach((milestone) => {
      linesForMilestone(milestone, progress).forEach((line) => {
        scheduled += 1;
        delay += 350;
        timers.push(
          window.setTimeout(() => {
            setLogLines((current) => [...current, line]);
            setPendingWrites((count) => Math.max(0, count - 1));
          }, delay),
        );
      });
    });
    setPendingWrites((count) => count + scheduled);

    return () => {
      timers.forEach((id) => window.clearTimeout(id));
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps -- milestone fields only; full `progress` identity changes every render
  }, [
    progress.nameReady,
    progress.repoReady,
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
    ? "Workspace setup is complete."
    : progress.nameReady
      ? "SuperPlane prepares this workspace as you finish each section."
      : "Shows progress while you set up this workspace.";

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
