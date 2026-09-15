import {
  factoryContentBodyClassName,
  factoryContentHeaderClassName,
  factoryPageSubtitleClassName,
  factoryPageTitleClassName,
} from "@/pages/factories/pages/factoryPageLayoutStyles";
import type {
  Achievement,
  HealthSnapshot,
  RecurringPattern,
  RunSummaryReport,
  Streak,
} from "@/pages/factories/verification/types";

import { AchievementsGrid } from "./AchievementsGrid";
import { HealthScoreCard } from "./HealthScoreCard";
import { RecurringPatternCard } from "./RecurringPatternCard";
import { RunSummaryReportCard } from "./RunSummaryReportCard";
import { StreakIndicator } from "./StreakIndicator";

interface FactoryHealthPageProps {
  factoryName: string;
  snapshot: HealthSnapshot;
  streaks: Streak[];
  patterns: RecurringPattern[];
  achievements: Achievement[];
  latestReport: RunSummaryReport;
  onViewSuggestions: (patternId: string) => void;
}

/**
 * The Health tab for a factory: score, streaks, recurring patterns,
 * achievements, and the latest run summary, composed in the workspace page
 * layout.
 */
export function FactoryHealthPage({
  factoryName,
  snapshot,
  streaks,
  patterns,
  achievements,
  latestReport,
  onViewSuggestions,
}: FactoryHealthPageProps) {
  return (
    <div className="min-h-screen bg-background">
      <header className={factoryContentHeaderClassName}>
        <h1 className={factoryPageTitleClassName}>Health</h1>
        <p className={factoryPageSubtitleClassName}>
          Verification results for {factoryName}, aggregated over time. Fixing findings raises the score.
        </p>
      </header>

      <main className={factoryContentBodyClassName}>
        <div className="flex flex-col gap-6">
          <div className="grid gap-3 lg:grid-cols-[minmax(0,1.2fr)_minmax(0,2fr)]">
            <HealthScoreCard label="Health score" snapshot={snapshot} />
            <StreakIndicator streaks={streaks} />
          </div>

          <section className="flex flex-col gap-3" aria-label="Recurring patterns">
            <div className="flex flex-col gap-0.5">
              <h3 className="workspace-section-title text-foreground">Recurring patterns</h3>
              <p className="text-[12px] text-muted-foreground">
                Findings that repeat across work orders. Reduce a pattern to zero to complete it.
              </p>
            </div>
            <div className="grid gap-3 lg:grid-cols-2">
              {patterns.map((pattern) => (
                <RecurringPatternCard key={pattern.id} pattern={pattern} onViewSuggestions={onViewSuggestions} />
              ))}
            </div>
          </section>

          <AchievementsGrid achievements={achievements} />

          <section className="flex flex-col gap-3" aria-label="Latest verification summary">
            <div className="flex flex-col gap-0.5">
              <h3 className="workspace-section-title text-foreground">Latest verification summary</h3>
              <p className="text-[12px] text-muted-foreground">
                The same summary posts to Slack when a channel is configured.
              </p>
            </div>
            <RunSummaryReportCard report={latestReport} />
          </section>
        </div>
      </main>
    </div>
  );
}
