import { Loader2 } from "lucide-react";

import { cn } from "@/lib/utils";
import { factoryCardClassName } from "@/pages/factories/pages/factoryPageLayoutStyles";

import type { CheckResult, Finding, FindingSeverity, VerificationRun } from "./types";
import { CheckKindLabel, CheckOutcomeChip } from "./CheckOutcomeChip";
import { FindingItem } from "./FindingItem";
import { VerificationStatusBadge } from "./VerificationStatusBadge";

const SEVERITY_ORDER: FindingSeverity[] = ["high", "medium", "low"];

const SEVERITY_GROUP_LABELS: Record<FindingSeverity, string> = {
  high: "High severity",
  medium: "Medium severity",
  low: "Low severity",
};

interface WorkOrderVerificationPanelProps {
  run: VerificationRun;
}

/**
 * Verification results on the work order detail: run status, per-check
 * outcomes, and findings grouped by severity. Command check results are
 * shown apart from agent review results because commands are authoritative.
 */
export function WorkOrderVerificationPanel({ run }: WorkOrderVerificationPanelProps) {
  const blockingOpen = run.findings.filter(
    (finding) => finding.enforcement === "blocking" && finding.status === "open",
  ).length;

  return (
    <section className={cn(factoryCardClassName, "flex flex-col")} aria-label="Verification">
      <header className="flex flex-wrap items-center justify-between gap-2 border-b border-border px-4 py-3">
        <div className="flex items-center gap-2">
          <h3 className="workspace-section-title text-foreground">Verification</h3>
          <VerificationStatusBadge status={run.status} />
        </div>
        <p className="text-[12px] text-muted-foreground">
          Suite {run.suiteName} · Rule set {run.ruleSetName}
        </p>
      </header>

      <div className="flex flex-col gap-4 px-4 py-4">
        {run.status === "failed" ? (
          <p className="workspace-body-text text-foreground">
            {blockingOpen} blocking {blockingOpen === 1 ? "finding stops" : "findings stop"} this line. Fix or accept
            the blocking findings, then dispatch the work order again.
          </p>
        ) : null}
        {run.status === "running" ? (
          <p className="flex items-center gap-2 workspace-body-text text-muted-foreground">
            <Loader2 className="size-4 animate-spin" aria-hidden />
            Checks run in parallel. Results appear as each check finishes.
          </p>
        ) : null}
        {run.status === "passed" ? (
          <p className="workspace-body-text text-foreground">
            All checks passed. Advisory findings do not stop the line.
          </p>
        ) : null}

        <ChecksList results={run.checks} />
        <FindingsBySeverity findings={run.findings} />
      </div>
    </section>
  );
}

function ChecksList({ results }: { results: CheckResult[] }) {
  const commandResults = results.filter((result) => result.check.kind === "command");
  const agentResults = results.filter((result) => result.check.kind === "agent");
  return (
    <div className="grid gap-4 sm:grid-cols-2">
      <CheckGroup
        title="Command checks"
        helper="Deterministic results. These decide the gate."
        results={commandResults}
      />
      <CheckGroup title="Agent reviews" helper="Findings from AI review of changed files." results={agentResults} />
    </div>
  );
}

function CheckGroup({ title, helper, results }: { title: string; helper: string; results: CheckResult[] }) {
  return (
    <div className="flex flex-col gap-1 rounded-md border border-border bg-background p-3">
      <p className="workspace-section-label text-muted-foreground">{title}</p>
      <p className="text-[12px] text-muted-foreground">{helper}</p>
      <ul className="mt-2 flex flex-col divide-y divide-border">
        {results.map((result) => (
          <li key={result.check.id} className="flex flex-col gap-1 py-2">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <span className="text-[13px] font-medium text-foreground">{result.check.name}</span>
              <CheckOutcomeChip outcome={result.outcome} />
            </div>
            <div className="flex flex-wrap items-center gap-3">
              <CheckKindLabel kind={result.check.kind} />
              {result.command ? (
                <code className="font-mono text-[12px] text-muted-foreground">{result.command}</code>
              ) : null}
            </div>
            {result.summary ? <p className="text-[12px] text-muted-foreground">{result.summary}</p> : null}
          </li>
        ))}
      </ul>
    </div>
  );
}

function FindingsBySeverity({ findings }: { findings: Finding[] }) {
  if (findings.length === 0) {
    return (
      <div className="rounded-md border border-dashed border-border bg-background px-4 py-6 text-center">
        <p className="workspace-body-text text-muted-foreground">No findings for this run.</p>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-3">
      {SEVERITY_ORDER.map((severity) => {
        const group = findings.filter((finding) => finding.severity === severity);
        if (group.length === 0) return null;
        return (
          <div key={severity} className="rounded-md border border-border bg-background px-4 py-2">
            <p className="workspace-section-label pt-2 text-muted-foreground">
              {SEVERITY_GROUP_LABELS[severity]} ({group.length})
            </p>
            <div className="divide-y divide-border">
              {group.map((finding) => (
                <FindingItem key={finding.id} finding={finding} />
              ))}
            </div>
          </div>
        );
      })}
    </div>
  );
}
