import { CheckCircle2, XCircle, Circle, Loader2, ChevronDown, ChevronUp } from "lucide-react";
import { cn } from "@/lib/utils";
import { useState } from "react";

export type CriterionStatus = "pending" | "evaluating" | "passed" | "failed";

export type OutcomeCriterion = {
  text: string;
  status: CriterionStatus;
  feedback?: string;
};

export type OutcomePhase = "building" | "grading" | "passed" | "failed" | "exhausted";

export type OutcomeState = {
  title: string;
  criteria: OutcomeCriterion[];
  iteration: number;
  maxIterations: number;
  phase: OutcomePhase;
};

const phaseLabel: Record<OutcomePhase, string> = {
  building: "Building…",
  grading: "Grading…",
  passed: "Complete",
  failed: "Fixing…",
  exhausted: "Max iterations reached",
};

const phaseColor: Record<OutcomePhase, string> = {
  building: "text-blue-600",
  grading: "text-amber-600",
  passed: "text-emerald-600",
  failed: "text-red-600",
  exhausted: "text-red-600",
};

function StatusIcon({ status }: { status: CriterionStatus }) {
  switch (status) {
    case "passed":
      return <CheckCircle2 size={14} className="text-emerald-500 shrink-0" />;
    case "failed":
      return <XCircle size={14} className="text-red-500 shrink-0" />;
    case "evaluating":
      return <Loader2 size={14} className="text-amber-500 animate-spin shrink-0" />;
    default:
      return <Circle size={14} className="text-slate-300 shrink-0" />;
  }
}

export function OutcomeProgressWidget({ state }: { state: OutcomeState }) {
  const [expandedCriterion, setExpandedCriterion] = useState<number | null>(null);

  return (
    <div
      className="border border-slate-200 rounded-lg bg-white shadow-sm overflow-hidden"
      data-testid="outcome-progress"
    >
      {/* Header */}
      <div className="px-3 py-2 bg-slate-50 border-b border-slate-200 flex items-center justify-between">
        <div className="flex items-center gap-2 min-w-0">
          <span className="text-sm font-medium truncate">🎯 {state.title}</span>
        </div>
        <div className="flex items-center gap-2 shrink-0">
          <span className={cn("text-xs font-medium", phaseColor[state.phase])}>
            {state.phase === "grading" && <Loader2 size={10} className="inline animate-spin mr-1" />}
            {phaseLabel[state.phase]}
          </span>
          <span className="text-xs text-slate-400">
            {state.iteration}/{state.maxIterations}
          </span>
        </div>
      </div>

      {/* Criteria list */}
      <div className="divide-y divide-slate-100">
        {state.criteria.map((criterion, i) => {
          const hasFeedback = criterion.status === "failed" && criterion.feedback;
          const isExpanded = expandedCriterion === i;
          return (
            <div key={i}>
              <button
                type="button"
                onClick={() => hasFeedback && setExpandedCriterion(isExpanded ? null : i)}
                className={cn(
                  "flex items-center gap-2 px-3 py-1.5 w-full text-left",
                  hasFeedback && "cursor-pointer hover:bg-slate-50",
                  !hasFeedback && "cursor-default",
                )}
                disabled={!hasFeedback}
              >
                <StatusIcon status={criterion.status} />
                <span className="text-xs text-slate-700 flex-1">{criterion.text}</span>
                {hasFeedback && (
                  isExpanded
                    ? <ChevronUp size={12} className="text-slate-400 shrink-0" />
                    : <ChevronDown size={12} className="text-slate-400 shrink-0" />
                )}
              </button>
              {hasFeedback && isExpanded && (
                <div className="px-3 pb-2 pl-8">
                  <p className="text-xs text-red-600 bg-red-50 rounded px-2 py-1">{criterion.feedback}</p>
                </div>
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
