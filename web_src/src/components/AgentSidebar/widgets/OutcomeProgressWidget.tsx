import { Loader2, ClipboardList, X } from "lucide-react";
import { cn } from "@/lib/utils";
import { useState } from "react";
import type { RubricCategory } from "./parser";

export type OutcomePhase = "building" | "grading" | "passed" | "failed" | "exhausted";

export type IterationEntry = {
  phase: "building" | "finished";
};

export type GradingEntry = {
  phase: "grading" | "needs_revision" | "satisfied";
  explanation?: string;
};

export type OutcomeState = {
  title: string;
  criteria: { text: string }[];
  categories?: RubricCategory[];
  iteration: number;
  maxIterations: number;
  phase: OutcomePhase;
  /** Log of iteration + grading results */
  log: (IterationEntry | GradingEntry)[];
};

// Keep these exports for backward compat with index.tsx
export type OutcomeCategory = {
  heading: string;
  criteriaIndices: number[];
};

const phaseLabel: Record<OutcomePhase, string> = {
  building: "Building…",
  grading: "Grading…",
  passed: "Complete ✅",
  failed: "Needs revision",
  exhausted: "Max iterations reached",
};

const phaseColor: Record<OutcomePhase, string> = {
  building: "text-blue-600",
  grading: "text-amber-600",
  passed: "text-emerald-600",
  failed: "text-red-600",
  exhausted: "text-red-600",
};

export function OutcomeProgressWidget({ state, onDismiss }: { state: OutcomeState; onDismiss?: () => void }) {
  const [rubricOpen, setRubricOpen] = useState(false);
  const [explanationOpen, setExplanationOpen] = useState<number | null>(null);

  const hasCategories = state.categories && state.categories.length > 0;
  const sectionCount = hasCategories ? state.categories!.length : 0;
  const criteriaCount = state.criteria.length;

  return (
    <>
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
            {onDismiss && (state.phase === "passed" || state.phase === "exhausted") && (
              <button
                type="button"
                onClick={onDismiss}
                className="text-slate-400 hover:text-slate-600 transition-colors"
                title="Dismiss"
              >
                <X size={14} />
              </button>
            )}
            <span className={cn("text-xs font-medium", phaseColor[state.phase])}>
              {(state.phase === "building" || state.phase === "grading") && (
                <Loader2 size={10} className="inline animate-spin mr-1" />
              )}
              {phaseLabel[state.phase]}
            </span>
          </div>
        </div>

        {/* Sub-header: rubric meta */}
        <div className="px-3 py-1.5 border-b border-slate-100 flex items-center justify-between">
          <span className="text-[10px] text-slate-500">
            {sectionCount > 0 ? `${sectionCount} sections · ` : ""}
            {criteriaCount} criteria
          </span>
          <button
            type="button"
            onClick={() => setRubricOpen(true)}
            className="text-[10px] text-violet-600 hover:text-violet-800 font-medium"
          >
            View rubric
          </button>
        </div>

        {/* Iteration log */}
        <div className="px-3 py-2 space-y-1">
          {(state.log ?? []).map((entry, i) => {
            if ("phase" in entry && (entry.phase === "building" || entry.phase === "finished")) {
              const iterEntry = entry as IterationEntry;
              const iterNum = Math.floor(i / 2) + 1;
              return (
                <div key={i} className="flex items-center gap-2 py-0.5">
                  {iterEntry.phase === "building" ? (
                    <Loader2 size={12} className="text-blue-500 animate-spin shrink-0" />
                  ) : (
                    <span className="text-xs shrink-0">✓</span>
                  )}
                  <span className="text-xs text-slate-700">
                    Iteration {iterNum} — {iterEntry.phase === "building" ? "building…" : "finished"}
                  </span>
                </div>
              );
            }

            const gradEntry = entry as GradingEntry;
            return (
              <div key={i} className="flex items-center gap-2 py-0.5">
                {gradEntry.phase === "grading" ? (
                  <Loader2 size={12} className="text-amber-500 animate-spin shrink-0" />
                ) : gradEntry.phase === "satisfied" ? (
                  <span className="text-xs shrink-0">✅</span>
                ) : (
                  <span className="text-xs shrink-0">❌</span>
                )}
                <span
                  className={cn(
                    "text-xs",
                    gradEntry.phase === "satisfied"
                      ? "text-emerald-700"
                      : gradEntry.phase === "needs_revision"
                        ? "text-red-700"
                        : "text-amber-700",
                  )}
                >
                  {gradEntry.phase === "grading"
                    ? "Grading…"
                    : gradEntry.phase === "satisfied"
                      ? "Satisfied"
                      : "Needs revision"}
                </span>
                {gradEntry.explanation && (
                  <button
                    type="button"
                    onClick={() => setExplanationOpen(i)}
                    className="text-[10px] text-slate-500 hover:text-slate-700 underline ml-auto shrink-0"
                  >
                    see result
                  </button>
                )}
              </div>
            );
          })}

          {(!state.log || state.log.length === 0) && (
            <div className="flex items-center gap-2 py-0.5">
              <Loader2 size={12} className="text-blue-500 animate-spin shrink-0" />
              <span className="text-xs text-slate-500">Starting…</span>
            </div>
          )}
        </div>
      </div>

      {/* Rubric modal */}
      {rubricOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
          <div className="bg-white rounded-lg shadow-xl w-full max-w-lg max-h-[80vh] flex flex-col mx-4">
            <div className="flex items-center justify-between px-4 py-3 border-b border-slate-200">
              <div className="flex items-center gap-2">
                <ClipboardList size={16} className="text-violet-600" />
                <h2 className="text-sm font-semibold text-slate-900">{state.title}</h2>
              </div>
              <button
                type="button"
                onClick={() => setRubricOpen(false)}
                className="text-slate-400 hover:text-slate-600"
              >
                <X size={16} />
              </button>
            </div>
            <div className="overflow-y-auto p-4 flex-1">
              {hasCategories ? (
                <div className="space-y-3">
                  {state.categories!.map((cat, ci) => (
                    <div key={ci}>
                      <p className="text-[10px] font-semibold text-violet-600 uppercase tracking-wide mb-1">
                        {cat.heading}
                      </p>
                      {cat.criteria.map((c, j) => (
                        <div key={j} className="flex items-start gap-2 py-1 border-b border-slate-50 last:border-0">
                          <span className="text-violet-400 text-xs mt-0.5 shrink-0">✦</span>
                          <span className="text-sm text-slate-700">{c.text}</span>
                        </div>
                      ))}
                    </div>
                  ))}
                </div>
              ) : (
                state.criteria.map((c, i) => (
                  <div key={i} className="flex items-start gap-2 py-1.5 border-b border-slate-50 last:border-0">
                    <span className="text-violet-500 text-sm mt-0.5 shrink-0 font-medium">{i + 1}.</span>
                    <span className="text-sm text-slate-700">{c.text}</span>
                  </div>
                ))
              )}
            </div>
            <div className="px-4 py-3 border-t border-slate-200 flex justify-end">
              <button
                type="button"
                onClick={() => setRubricOpen(false)}
                className="text-xs text-slate-500 hover:text-slate-700"
              >
                Close
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Explanation modal */}
      {explanationOpen !== null &&
        (() => {
          const entry = state.log?.[explanationOpen] as GradingEntry;
          if (!entry?.explanation) return null;
          return (
            <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
              <div className="bg-white rounded-lg shadow-xl w-full max-w-lg max-h-[80vh] flex flex-col mx-4">
                <div className="flex items-center justify-between px-4 py-3 border-b border-slate-200">
                  <h2 className="text-sm font-semibold text-slate-900">Grading Result</h2>
                  <button
                    type="button"
                    onClick={() => setExplanationOpen(null)}
                    className="text-slate-400 hover:text-slate-600"
                  >
                    <X size={16} />
                  </button>
                </div>
                <div className="overflow-y-auto p-4 flex-1">
                  <p className="text-sm text-slate-700 whitespace-pre-wrap">{entry.explanation}</p>
                </div>
                <div className="px-4 py-3 border-t border-slate-200 flex justify-end">
                  <button
                    type="button"
                    onClick={() => setExplanationOpen(null)}
                    className="text-xs text-slate-500 hover:text-slate-700"
                  >
                    Close
                  </button>
                </div>
              </div>
            </div>
          );
        })()}
    </>
  );
}
