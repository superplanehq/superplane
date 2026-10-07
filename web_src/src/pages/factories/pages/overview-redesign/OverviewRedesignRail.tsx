import { BookOpen, FlaskConical, Loader2, Sparkles, Workflow } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

import { CardEmptyState, OverviewCard } from "./overviewRedesignCardParts";
import type {
  ImprovementCategory,
  ImprovementProposal,
  SuggestionsState,
  WorkspaceReadiness,
} from "./overviewRedesignMocks";

/* ------------------------- Suggested tasks ------------------------- */

function confidenceChipClassName(confidencePct: number) {
  if (confidencePct >= 80) {
    return "border-emerald-500/30 bg-emerald-500/10 text-emerald-700 dark:text-emerald-400";
  }
  if (confidencePct >= 60) {
    return "border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-400";
  }
  return "border-slate-500/30 bg-slate-500/10 text-slate-700 dark:text-slate-400";
}

export function SuggestionsCard({ suggestions }: { suggestions: SuggestionsState }) {
  return (
    <OverviewCard
      title="Suggested tasks"
      subtitle="Candidates from repository analysis."
      preview
      testId="overview-suggestions-card"
    >
      <SuggestionsCardBody suggestions={suggestions} />
    </OverviewCard>
  );
}

function SuggestionsCardBody({ suggestions }: { suggestions: SuggestionsState }) {
  if (suggestions.scanning) {
    return (
      <CardEmptyState
        icon={<Loader2 className="size-5 animate-spin text-muted-foreground" aria-hidden />}
        title="Repository scan in progress"
        hint={
          suggestions.scanTarget
            ? `Analyzing ${suggestions.scanTarget}. Suggestions appear when the scan completes.`
            : "Suggestions appear when the scan completes."
        }
      />
    );
  }

  if (suggestions.candidates.length === 0) {
    return (
      <CardEmptyState
        icon={<Sparkles className="size-5 text-muted-foreground" aria-hidden />}
        title="No suggestions right now"
        hint="SuperPlane analyzes connected repositories and proposes work here."
      />
    );
  }

  return (
    <ul>
      {suggestions.candidates.slice(0, 3).map((candidate) => (
        <li
          key={candidate.id}
          className="border-b border-border/60 px-4 py-3 last:border-b-0"
          data-testid={`overview-candidate-row-${candidate.id}`}
        >
          <p className="text-[13px] font-medium leading-snug text-foreground">{candidate.title}</p>
          <div className="mt-1.5 flex items-center gap-2">
            <span className="min-w-0 flex-1 truncate text-[12px] text-muted-foreground">{candidate.repository}</span>
            <Badge
              variant="outline"
              className={cn(
                "shrink-0 px-1.5 py-0 text-[11px] font-medium tabular-nums",
                confidenceChipClassName(candidate.confidencePct),
              )}
            >
              {candidate.confidencePct}%
            </Badge>
            {/* Mock-only: the candidate review flow does not exist yet. */}
            <Button
              size="xs"
              variant="outline"
              className="shrink-0"
              onClick={() => console.warn("mock action: review task candidate", candidate.id)}
            >
              Review
            </Button>
          </div>
        </li>
      ))}
    </ul>
  );
}

/* -------------------------- Workspace improvements -------------------------- */

const IMPROVEMENT_META: Record<ImprovementCategory, { label: string; icon: typeof FlaskConical }> = {
  tests: { label: "Tests", icon: FlaskConical },
  context: { label: "Agent context", icon: BookOpen },
  line: { label: "Line setup", icon: Workflow },
};

function readinessBarClassName(score: number) {
  return score < 50 ? "bg-amber-500" : "bg-emerald-500";
}

/**
 * Left column of the improvements card: the overall readiness score with
 * per-category mini bars, so the number is explainable at a glance.
 */
function ReadinessSummary({ readiness }: { readiness: WorkspaceReadiness }) {
  return (
    <div className="border-b border-border/60 px-5 py-4 md:w-80 md:shrink-0 md:border-b-0 md:border-r">
      <p className="text-[12px] text-muted-foreground">AI readiness</p>
      <p className="mt-0.5 flex items-baseline gap-1">
        <span className="text-[26px] font-semibold tabular-nums leading-tight text-foreground">
          {readiness.overall}
        </span>
        <span className="text-[12px] text-muted-foreground">/ 100</span>
      </p>
      <dl className="mt-4 space-y-3">
        {readiness.categories.map((category) => (
          <div key={category.id}>
            <div className="flex items-baseline justify-between gap-3">
              <dt className="text-[12px] text-muted-foreground">{category.label}</dt>
              <dd className="text-[12px] font-medium tabular-nums text-foreground">{category.score}</dd>
            </div>
            <div className="mt-1 h-1 overflow-hidden rounded-full bg-muted" aria-hidden>
              <div
                className={cn("h-full rounded-full", readinessBarClassName(category.score))}
                style={{ width: `${category.score}%` }}
              />
            </div>
          </div>
        ))}
      </dl>
    </div>
  );
}

export function ImprovementsCard({
  proposals,
  readiness,
}: {
  proposals: ImprovementProposal[];
  readiness?: WorkspaceReadiness;
}) {
  return (
    <OverviewCard
      title="Workspace improvements"
      subtitle="Proposals from repository and line analysis."
      preview
      testId="overview-improvements-card"
    >
      {!readiness && proposals.length === 0 ? (
        <CardEmptyState title="No proposals yet" hint="Proposals appear after the first repository scan." />
      ) : (
        <div className="md:flex">
          {readiness ? <ReadinessSummary readiness={readiness} /> : null}
          <ul className="min-w-0 flex-1">
            {proposals.slice(0, 3).map((proposal) => {
              const meta = IMPROVEMENT_META[proposal.category];
              const Icon = meta.icon;
              return (
                <li
                  key={proposal.id}
                  className="flex items-center gap-3 border-b border-border/60 px-4 py-3 last:border-b-0"
                  data-testid={`overview-improvement-row-${proposal.id}`}
                >
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="inline-flex items-center gap-1 text-[11px] font-medium text-muted-foreground">
                        <Icon className="size-3" aria-hidden />
                        {meta.label}
                      </span>
                      <Badge
                        variant="outline"
                        className="border-emerald-500/30 bg-emerald-500/10 px-1.5 py-0 text-[11px] font-medium tabular-nums text-emerald-700 dark:text-emerald-400"
                      >
                        {proposal.impactLabel}
                      </Badge>
                    </div>
                    <p className="mt-1 text-[13px] font-medium leading-snug text-foreground">{proposal.title}</p>
                    <p className="mt-0.5 text-[12px] leading-snug text-muted-foreground">{proposal.description}</p>
                  </div>
                  {/* Mock-only: the proposal flows (update line, create automation…) do not exist yet. */}
                  <Button
                    size="xs"
                    variant="outline"
                    className="shrink-0"
                    onClick={() => console.warn("mock action: apply improvement proposal", proposal.id)}
                  >
                    {proposal.actionLabel}
                  </Button>
                </li>
              );
            })}
          </ul>
        </div>
      )}
    </OverviewCard>
  );
}
