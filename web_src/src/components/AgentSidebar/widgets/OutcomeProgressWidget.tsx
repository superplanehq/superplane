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

export type OutcomeCategory = {
  heading: string;
  /** Indices into the flat criteria array */
  criteriaIndices: number[];
};

export type OutcomeState = {
  title: string;
  criteria: OutcomeCriterion[];
  categories?: OutcomeCategory[];
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
  const [showAll, setShowAll] = useState(false);
  const VISIBLE_COUNT = 3;
  const visibleCriteria = showAll ? state.criteria : state.criteria.slice(0, VISIBLE_COUNT);
  const hiddenCount = state.criteria.length - VISIBLE_COUNT;

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
        {state.categories && state.categories.length > 0 ? (
          // Categorized view
          <>
            {state.categories.map((cat, ci) => {
              const catCriteria = cat.criteriaIndices.map((idx) => ({ ...state.criteria[idx], _idx: idx }));
              if (!showAll && ci > 0) return null; // Only show first category in preview
              const visible = !showAll ? catCriteria.slice(0, VISIBLE_COUNT) : catCriteria;
              return (
                <div key={ci}>
                  <div className="px-3 pt-2 pb-0.5">
                    <span className="text-[10px] font-semibold text-slate-500 uppercase tracking-wide">{cat.heading}</span>
                  </div>
                  {visible.map((item) => (
                    <CriterionRow
                      key={item._idx}
                      criterion={item}
                      index={item._idx}
                      expandedCriterion={expandedCriterion}
                      setExpandedCriterion={setExpandedCriterion}
                    />
                  ))}
                </div>
              );
            })}
          </>
        ) : (
          // Flat view
          visibleCriteria.map((criterion, i) => (
            <CriterionRow
              key={i}
              criterion={criterion}
              index={i}
              expandedCriterion={expandedCriterion}
              setExpandedCriterion={setExpandedCriterion}
            />
          ))
        )}
        {hiddenCount > 0 && !showAll && (
          <button
            type="button"
            onClick={() => setShowAll(true)}
            className="flex items-center gap-1 px-3 py-1.5 w-full text-left text-xs text-slate-500 hover:text-slate-700 hover:bg-slate-50"
          >
            <ChevronDown size={12} />+{hiddenCount} more
          </button>
        )}
        {showAll && hiddenCount > 0 && (
          <button
            type="button"
            onClick={() => setShowAll(false)}
            className="flex items-center gap-1 px-3 py-1.5 w-full text-left text-xs text-slate-500 hover:text-slate-700 hover:bg-slate-50"
          >
            <ChevronUp size={12} />
            Show less
          </button>
        )}
      </div>
    </div>
  );
}

function CriterionRow({
  criterion,
  index,
  expandedCriterion,
  setExpandedCriterion,
}: {
  criterion: OutcomeCriterion;
  index: number;
  expandedCriterion: number | null;
  setExpandedCriterion: (v: number | null) => void;
}) {
  const hasFeedback = criterion.status === "failed" && criterion.feedback;
  const isExpanded = expandedCriterion === index;
  return (
    <div>
      <button
        type="button"
        onClick={() => hasFeedback && setExpandedCriterion(isExpanded ? null : index)}
        className={cn(
          "flex items-center gap-2 px-3 py-1.5 w-full text-left",
          hasFeedback && "cursor-pointer hover:bg-slate-50",
          !hasFeedback && "cursor-default",
        )}
        disabled={!hasFeedback}
      >
        <StatusIcon status={criterion.status} />
        <span className="text-xs text-slate-700 flex-1">{criterion.text}</span>
        {hasFeedback &&
          (isExpanded ? (
            <ChevronUp size={12} className="text-slate-400 shrink-0" />
          ) : (
            <ChevronDown size={12} className="text-slate-400 shrink-0" />
          ))}
      </button>
      {hasFeedback && isExpanded && (
        <div className="px-3 pb-2 pl-8">
          <p className="text-xs text-red-600 bg-red-50 rounded px-2 py-1">{criterion.feedback}</p>
        </div>
      )}
    </div>
  );
}
