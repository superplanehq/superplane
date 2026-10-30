import { RunStatusLabel } from "./RunIdentity";
import { IntakeOutcomeLabel } from "./IntakeOutcomeLabel";
import { INTAKE_OUTCOMES } from "./intakeRunOutcome";
import { RunHistoryTable } from "./RunHistoryTable";
import type { AutomationRun } from "./types";

interface AutomationRunHistoryProps {
  runs: AutomationRun[];
  onSelectRun: (run: AutomationRun) => void;
  loading?: boolean;
  error?: string;
  onRetry?: () => void;
  onLoadMore?: () => void;
  loadingMore?: boolean;
  tasksLoading?: boolean;
}

export function AutomationRunHistory({ runs, onSelectRun, tasksLoading, ...props }: AutomationRunHistoryProps) {
  return (
    <RunHistoryTable
      {...props}
      subjectsLoading={tasksLoading}
      runs={runs.map((run) => ({
        id: run.id,
        subject: run.source
          ? { key: run.source.reference, title: run.source.title }
          : run.task
            ? { key: run.task.id, title: run.task.title }
            : undefined,
        status: run.intake ? INTAKE_OUTCOMES[run.intake.outcome].label : run.status,
        statusContent: run.intake ? (
          <IntakeOutcomeLabel outcome={run.intake.outcome} />
        ) : (
          <RunStatusLabel status={run.status} />
        ),
        startedAt: run.startedAt,
        duration: run.duration,
        searchText: [run.task?.id, run.task?.title, run.intake?.event].join(" "),
      }))}
      onSelectRun={(id) => {
        const run = runs.find((run) => run.id === id);
        if (run) onSelectRun(run);
      }}
    />
  );
}
