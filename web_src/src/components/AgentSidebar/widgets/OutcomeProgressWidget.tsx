import { ClipboardList, Loader2, X } from "lucide-react";
import { useMemo, useState, type ReactNode } from "react";
import { cn } from "@/lib/utils";
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

  const explanationEntry = useMemo(() => {
    if (explanationOpen === null) return null;

    const entry = state.log?.[explanationOpen];
    if (!entry || !("explanation" in entry) || !entry.explanation) {
      return null;
    }

    return entry;
  }, [explanationOpen, state.log]);

  const dismissible = onDismiss && (state.phase === "passed" || state.phase === "exhausted");
  const sectionCount = state.categories?.length ?? 0;

  return (
    <>
      <div
        className="overflow-hidden rounded-lg border border-slate-200 bg-white dark:border-gray-700 dark:bg-gray-800"
        data-testid="outcome-progress"
      >
        <OutcomeHeader
          title={state.title}
          phase={state.phase}
          dismissible={Boolean(dismissible)}
          onDismiss={onDismiss}
        />

        <RubricSummary
          sectionCount={sectionCount}
          criteriaCount={state.criteria.length}
          onOpenRubric={() => setRubricOpen(true)}
        />

        <OutcomeLog entries={state.log} onOpenExplanation={setExplanationOpen} />
      </div>

      <RubricModal open={rubricOpen} state={state} onClose={() => setRubricOpen(false)} />

      <ExplanationModal
        explanation={explanationEntry?.explanation}
        open={explanationEntry != null}
        onClose={() => setExplanationOpen(null)}
      />
    </>
  );
}

function OutcomeHeader({
  title,
  phase,
  dismissible,
  onDismiss,
}: {
  title: string;
  phase: OutcomePhase;
  dismissible: boolean;
  onDismiss?: () => void;
}) {
  return (
    <div className="flex items-center justify-between border-b border-slate-200 bg-slate-50 px-3 py-2 dark:border-gray-700 dark:bg-gray-900/60">
      <div className="flex min-w-0 items-center gap-2">
        <span className="truncate text-sm font-medium">🎯 {title}</span>
      </div>
      <div className="flex shrink-0 items-center gap-2">
        {dismissible ? (
          <button
            type="button"
            onClick={onDismiss}
            className="text-slate-400 transition-colors hover:text-slate-600 dark:text-gray-500 dark:hover:text-gray-300"
            title="Dismiss"
          >
            <X size={14} />
          </button>
        ) : null}
        <span className={cn("text-xs font-medium", phaseColor[phase])}>
          {showPhaseSpinner(phase) ? <Loader2 size={10} className="mr-1 inline animate-spin" /> : null}
          {phaseLabel[phase]}
        </span>
      </div>
    </div>
  );
}

function RubricSummary({
  sectionCount,
  criteriaCount,
  onOpenRubric,
}: {
  sectionCount: number;
  criteriaCount: number;
  onOpenRubric: () => void;
}) {
  return (
    <div className="flex items-center justify-between border-b border-slate-100 px-3 py-1.5 dark:border-gray-700">
      <span className="text-[10px] text-slate-500 dark:text-gray-400">
        {sectionCount > 0 ? `${sectionCount} sections · ` : ""}
        {criteriaCount} criteria
      </span>
      <button
        type="button"
        onClick={onOpenRubric}
        className="text-[10px] font-medium text-slate-600 hover:text-slate-800 dark:text-gray-300 dark:hover:text-gray-100"
      >
        View rubric
      </button>
    </div>
  );
}

function OutcomeLog({
  entries,
  onOpenExplanation,
}: {
  entries: OutcomeState["log"];
  onOpenExplanation: (index: number) => void;
}) {
  if (!entries.length) {
    return (
      <div className="px-3 py-2">
        <div className="flex items-center gap-2 py-0.5">
          <Loader2 size={12} className="shrink-0 animate-spin text-blue-500" />
          <span className="text-xs text-slate-500 dark:text-gray-400">Starting…</span>
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-1 px-3 py-2">
      {entries.map((entry, index) =>
        isIterationEntry(entry) ? (
          <IterationLogRow key={index} entry={entry} iteration={Math.floor(index / 2) + 1} />
        ) : (
          <GradingLogRow key={index} entry={entry} onOpenExplanation={() => onOpenExplanation(index)} />
        ),
      )}
    </div>
  );
}

function IterationLogRow({ entry, iteration }: { entry: IterationEntry; iteration: number }) {
  return (
    <div className="flex items-center gap-2 py-0.5">
      {entry.phase === "building" ? (
        <Loader2 size={12} className="shrink-0 animate-spin text-blue-500" />
      ) : (
        <span className="shrink-0 text-xs">✓</span>
      )}
      <span className="text-xs text-slate-700 dark:text-gray-300">
        Iteration {iteration} — {entry.phase === "building" ? "building…" : "finished"}
      </span>
    </div>
  );
}

function GradingLogRow({ entry, onOpenExplanation }: { entry: GradingEntry; onOpenExplanation: () => void }) {
  return (
    <div className="flex items-center gap-2 py-0.5">
      {gradingIcon(entry.phase)}
      <span className={cn("text-xs", gradingTextColor(entry.phase))}>{gradingLabel(entry.phase)}</span>
      {entry.explanation ? (
        <button
          type="button"
          onClick={onOpenExplanation}
          className="ml-auto shrink-0 text-[10px] text-slate-500 underline hover:text-slate-700 dark:text-gray-400 dark:hover:text-gray-200"
        >
          see result
        </button>
      ) : null}
    </div>
  );
}

function RubricModal({ open, state, onClose }: { open: boolean; state: OutcomeState; onClose: () => void }) {
  if (!open) {
    return null;
  }

  return (
    <CenteredModal
      title={state.title}
      onClose={onClose}
      titleIcon={<ClipboardList size={16} className="text-slate-600 dark:text-gray-300" />}
    >
      {state.categories && state.categories.length > 0 ? (
        <CategorizedCriteriaList categories={state.categories} />
      ) : (
        <FlatCriteriaList criteria={state.criteria} />
      )}
    </CenteredModal>
  );
}

function ExplanationModal({
  explanation,
  open,
  onClose,
}: {
  explanation?: string;
  open: boolean;
  onClose: () => void;
}) {
  if (!open || !explanation) {
    return null;
  }

  return (
    <CenteredModal title="Grading Result" onClose={onClose}>
      <p className="whitespace-pre-wrap text-sm text-slate-700 dark:text-gray-300">{explanation}</p>
    </CenteredModal>
  );
}

function CenteredModal({
  title,
  titleIcon,
  children,
  onClose,
}: {
  title: string;
  titleIcon?: ReactNode;
  children: ReactNode;
  onClose: () => void;
}) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40">
      <div className="mx-4 flex max-h-[80vh] w-full max-w-lg flex-col rounded-lg bg-white shadow-xl dark:bg-gray-800">
        <div className="flex items-center justify-between border-b border-slate-200 px-4 py-3 dark:border-gray-700">
          <div className="flex items-center gap-2">
            {titleIcon}
            <h2 className="text-sm font-semibold text-slate-900 dark:text-gray-100">{title}</h2>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="text-slate-400 hover:text-slate-600 dark:text-gray-500 dark:hover:text-gray-300"
          >
            <X size={16} />
          </button>
        </div>
        <div className="flex-1 overflow-y-auto p-4">{children}</div>
        <div className="flex justify-end border-t border-slate-200 px-4 py-3 dark:border-gray-700">
          <button
            type="button"
            onClick={onClose}
            className="text-xs text-slate-500 hover:text-slate-700 dark:text-gray-400 dark:hover:text-gray-200"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
}

function CategorizedCriteriaList({ categories }: { categories: RubricCategory[] }) {
  return (
    <div className="space-y-3">
      {categories.map((category, categoryIndex) => (
        <div key={categoryIndex}>
          <p className="mb-1 text-[10px] font-semibold uppercase tracking-wide text-slate-500 dark:text-gray-400">
            {category.heading}
          </p>
          {category.criteria.map((criterion, criterionIndex) => (
            <CriteriaRow
              key={criterionIndex}
              prefix={<span className="mt-0.5 shrink-0 text-xs text-slate-400 dark:text-gray-500">✦</span>}
            >
              <span className="text-sm text-slate-700 dark:text-gray-300">{criterion.text}</span>
            </CriteriaRow>
          ))}
        </div>
      ))}
    </div>
  );
}

function FlatCriteriaList({ criteria }: { criteria: OutcomeState["criteria"] }) {
  return (
    <>
      {criteria.map((criterion, index) => (
        <CriteriaRow
          key={index}
          bordered
          prefix={
            <span className="mt-0.5 shrink-0 text-sm font-medium text-slate-500 dark:text-gray-400">{index + 1}.</span>
          }
        >
          <span className="text-sm text-slate-700 dark:text-gray-300">{criterion.text}</span>
        </CriteriaRow>
      ))}
    </>
  );
}

function CriteriaRow({
  children,
  prefix,
  bordered = false,
}: {
  children: ReactNode;
  prefix: ReactNode;
  bordered?: boolean;
}) {
  return (
    <div
      className={cn(
        "flex items-start gap-2",
        bordered ? "border-b border-slate-50 py-1.5 last:border-0 dark:border-gray-700" : "py-1",
      )}
    >
      {prefix}
      {children}
    </div>
  );
}

function isIterationEntry(entry: IterationEntry | GradingEntry): entry is IterationEntry {
  return entry.phase === "building" || entry.phase === "finished";
}

function showPhaseSpinner(phase: OutcomePhase): boolean {
  return phase === "building" || phase === "grading";
}

function gradingIcon(phase: GradingEntry["phase"]) {
  if (phase === "grading") {
    return <Loader2 size={12} className="shrink-0 animate-spin text-amber-500" />;
  }
  if (phase === "satisfied") {
    return <span className="shrink-0 text-xs">✅</span>;
  }
  return <span className="shrink-0 text-xs">❌</span>;
}

function gradingLabel(phase: GradingEntry["phase"]): string {
  switch (phase) {
    case "grading":
      return "Grading…";
    case "satisfied":
      return "Satisfied";
    default:
      return "Needs revision";
  }
}

function gradingTextColor(phase: GradingEntry["phase"]): string {
  switch (phase) {
    case "grading":
      return "text-amber-700 dark:text-amber-300";
    case "satisfied":
      return "text-emerald-700 dark:text-emerald-300";
    default:
      return "text-red-700 dark:text-red-300";
  }
}
