import {
  AlertTriangle,
  ArrowRight,
  CheckCircle2,
  ExternalLink,
  GitBranch,
  GitPullRequest,
  PauseCircle,
  Plug,
  ShieldCheck,
} from "lucide-react";

import { Button } from "@/components/ui/button";
import { appDarkModeClasses } from "@/lib/appDarkModeClasses";
import { formatTimeAgo } from "@/lib/date";
import { cn } from "@/lib/utils";
import { Sparkline } from "@/pages/app/console/widget/Sparkline";
import { EmptyState } from "@/ui/emptyState";
import { RunStatusBadge } from "@/ui/Runs/RunStatusBadge";

import type {
  FactoryActivityEntry,
  FactoryHomeData,
  FactoryOutcomeMetric,
  FactoryProject,
  FactoryPullRequest,
  FactoryReviewItem,
  FactoryRun,
  FactoryStartingTask,
} from "./factoryHomeTypes";
import {
  homeCardTitleClassName,
  homeListCardClassName,
  homePageTitleClassName,
  homePanelTitleClassName,
  homeTagClassName,
} from "./homePageStyles";

/**
 * Steps up from the gray-500/gray-400 pairing used elsewhere on Home: both fail
 * 4.5:1 — 500 on white (4.2:1) and 400 on the raised dark surface (4.38:1).
 */
const mutedTextClassName = "text-gray-600 dark:text-gray-300";
const panelClassName = cn(
  "rounded-lg bg-white p-5 outline outline-slate-950/10",
  appDarkModeClasses.surfaceRaised,
  "dark:outline-gray-700/70",
);
/** Sparklines support the value in a stat tile; they never compete with it. */
const sparklineClassName = "text-slate-300 dark:text-gray-600";

interface FactoryHomePageProps {
  data: FactoryHomeData;
  onStartTask: (task: FactoryStartingTask) => void;
  onOpenRun: (run: FactoryRun) => void;
  onOpenReview: (item: FactoryReviewItem) => void;
  /** Invoked from the health banner when the factory needs attention. */
  onResolveHealth: () => void;
}

/**
 * Homepage for a Software Factory scoped to one project.
 *
 * Section order follows what an operator needs in descending urgency: work that
 * is blocked on a person, work the agents are doing right now, then the outcome
 * telemetry and audit trail that say whether the factory is worth trusting.
 */
export function FactoryHomePage({ data, onStartTask, onOpenRun, onOpenReview, onResolveHealth }: FactoryHomePageProps) {
  const { project, startingTasks, needsReview, inFlight, outcomes, activity } = data;

  return (
    <div className="mx-auto w-full max-w-6xl px-8 py-10">
      <FactoryHeader project={project} onResolveHealth={onResolveHealth} />
      <OutcomesRow metrics={outcomes} />
      <div className="mt-6 grid grid-cols-1 gap-6 lg:grid-cols-3">
        <div className="flex flex-col gap-6 lg:col-span-2">
          <NeedsReviewPanel items={needsReview} onOpenReview={onOpenReview} />
          <InFlightPanel runs={inFlight} onOpenRun={onOpenRun} />
          <ActivityPanel entries={activity} />
        </div>
        <div className="flex flex-col gap-6">
          <StartTaskPanel tasks={startingTasks} onStartTask={onStartTask} />
          <FactorySetupPanel project={project} />
        </div>
      </div>
    </div>
  );
}

const HEALTH_META = {
  healthy: {
    label: "Factory healthy",
    icon: ShieldCheck,
    badgeClassName: "bg-emerald-100 text-emerald-700 dark:bg-emerald-950/70 dark:text-emerald-300",
  },
  degraded: {
    label: "Needs attention",
    icon: AlertTriangle,
    badgeClassName: "bg-amber-100 text-amber-800 dark:bg-amber-950/70 dark:text-amber-300",
  },
  paused: {
    label: "Paused",
    icon: PauseCircle,
    badgeClassName: "bg-slate-200 text-gray-700 dark:bg-slate-900 dark:text-gray-300",
  },
} as const;

function FactoryHeader({ project, onResolveHealth }: { project: FactoryProject; onResolveHealth: () => void }) {
  const health = HEALTH_META[project.health];
  const HealthIcon = health.icon;

  return (
    <header>
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2.5">
            <h1 className={cn(homePageTitleClassName, "text-2xl")}>{project.name}</h1>
            <span
              className={cn(
                "inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium",
                health.badgeClassName,
              )}
            >
              <HealthIcon className="size-3.5" aria-hidden />
              {health.label}
            </span>
          </div>
          <p className={cn("mt-2 max-w-2xl text-sm leading-normal", mutedTextClassName)}>{project.description}</p>
          <div className={cn("mt-3 flex flex-wrap items-center gap-x-4 gap-y-1.5 text-xs", mutedTextClassName)}>
            <a
              href={project.repositoryUrl}
              target="_blank"
              rel="noreferrer"
              className="inline-flex items-center gap-1.5 underline decoration-slate-300 underline-offset-4 hover:text-slate-900 dark:decoration-gray-600 dark:hover:text-gray-100"
            >
              {project.repository}
              <ExternalLink className="size-3" aria-hidden />
            </a>
            <span className="inline-flex items-center gap-1.5">
              <GitBranch className="size-3.5" aria-hidden />
              {project.defaultBranch}
            </span>
            <span>Owned by {project.owner}</span>
          </div>
        </div>
      </div>

      {project.health !== "healthy" && project.healthDetail && (
        <div
          className={cn(
            "mt-5 flex flex-wrap items-center justify-between gap-3 rounded-lg px-4 py-3",
            "bg-amber-50 outline outline-amber-200 dark:bg-amber-950/40 dark:outline-amber-900/60",
          )}
        >
          <p className="flex items-center gap-2 text-sm text-amber-900 dark:text-amber-200">
            <AlertTriangle className="size-4 shrink-0" aria-hidden />
            {project.healthDetail}
          </p>
          <Button type="button" size="sm" variant="outline" onClick={onResolveHealth}>
            Fix now
          </Button>
        </div>
      )}
    </header>
  );
}

function OutcomesRow({ metrics }: { metrics: FactoryOutcomeMetric[] }) {
  if (metrics.length === 0) return null;

  return (
    <div className="mt-6 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
      {metrics.map((metric) => (
        <OutcomeTile key={metric.id} metric={metric} />
      ))}
    </div>
  );
}

/**
 * Stat tile: label, value, signed delta against a named period, supporting
 * sparkline. The delta's sign carries the direction so color is never the only
 * channel saying whether the movement is good.
 */
function OutcomeTile({ metric }: { metric: FactoryOutcomeMetric }) {
  const improving = metric.delta?.startsWith(metric.betterDirection === "up" ? "+" : "-");
  const deltaToneClassName = improving ? "text-emerald-700 dark:text-emerald-400" : "text-red-700 dark:text-red-400";

  return (
    <div className={cn("px-4 py-3.5", homeListCardClassName)}>
      <p className={cn("text-xs font-medium", mutedTextClassName)}>{metric.label}</p>
      <div className="mt-1.5 flex items-end justify-between gap-3">
        <div>
          <p className="text-2xl font-semibold leading-none text-slate-900 dark:text-gray-100">{metric.value}</p>
          {metric.delta && (
            <p className="mt-1.5 text-xs">
              <span className={cn("font-medium", deltaToneClassName)}>{metric.delta}</span>
              {metric.deltaPeriod && <span className={mutedTextClassName}> vs {metric.deltaPeriod}</span>}
            </p>
          )}
        </div>
        {metric.trend && metric.trend.length > 0 && (
          <Sparkline values={metric.trend} width={92} height={30} className={sparklineClassName} />
        )}
      </div>
    </div>
  );
}

function PanelHeading({ title, count }: { title: string; count?: number }) {
  return (
    <div className="mb-3 flex items-center gap-2">
      <h2 className={homePanelTitleClassName}>{title}</h2>
      {count !== undefined && count > 0 && <span className={homeTagClassName}>{count}</span>}
    </div>
  );
}

function PullRequestLink({ pullRequest }: { pullRequest: FactoryPullRequest }) {
  return (
    <a
      href={pullRequest.url}
      target="_blank"
      rel="noreferrer"
      className={cn(
        "inline-flex items-center gap-1 underline decoration-slate-300 underline-offset-4",
        "hover:text-slate-900 dark:decoration-gray-600 dark:hover:text-gray-100",
      )}
    >
      <GitPullRequest className="size-3.5" aria-hidden />#{pullRequest.number}
    </a>
  );
}

function NeedsReviewPanel({
  items,
  onOpenReview,
}: {
  items: FactoryReviewItem[];
  onOpenReview: (item: FactoryReviewItem) => void;
}) {
  return (
    <section className={panelClassName}>
      <PanelHeading title="Waiting on you" count={items.length} />
      {items.length === 0 ? (
        <EmptyState
          compact
          tone="neutral"
          icon={CheckCircle2}
          title="Nothing is blocked"
          description="Agents will park work here when they need a decision."
        />
      ) : (
        <ul className="flex flex-col gap-2.5">
          {items.map((item) => (
            <li key={item.id} className={cn("px-3.5 py-3", homeListCardClassName)}>
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div className="min-w-0">
                  <p className={homeCardTitleClassName}>{item.title}</p>
                  <p className={cn("mt-1 text-sm", mutedTextClassName)}>{item.reason}</p>
                  <div className={cn("mt-2 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs", mutedTextClassName)}>
                    <span>Waiting {formatTimeAgo(new Date(item.waitingSince), false)}</span>
                    {item.pullRequest && <PullRequestLink pullRequest={item.pullRequest} />}
                  </div>
                </div>
                <Button type="button" size="sm" onClick={() => onOpenReview(item)}>
                  Review
                  <ArrowRight />
                </Button>
              </div>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

function InFlightPanel({ runs, onOpenRun }: { runs: FactoryRun[]; onOpenRun: (run: FactoryRun) => void }) {
  return (
    <section className={panelClassName}>
      <PanelHeading title="In flight" count={runs.length} />
      {runs.length === 0 ? (
        <EmptyState
          compact
          icon={GitBranch}
          title="No agents working"
          description="Start a task to put the factory to work."
        />
      ) : (
        <ul className="flex flex-col gap-2.5">
          {runs.map((run) => (
            <li key={run.id} className={cn("px-3.5 py-3", homeListCardClassName)}>
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div className="min-w-0">
                  {/*
                   * Only the title opens the run. Making the whole row a button
                   * would nest the pull-request link inside it, which is invalid.
                   */}
                  <button
                    type="button"
                    onClick={() => onOpenRun(run)}
                    className={cn(homeCardTitleClassName, "text-left hover:underline underline-offset-4")}
                  >
                    {run.title}
                  </button>
                  <div className={cn("mt-1.5 flex flex-wrap items-center gap-x-3 gap-y-1 text-xs", mutedTextClassName)}>
                    <span>{run.stage}</span>
                    <span className="inline-flex items-center gap-1.5">
                      <GitBranch className="size-3.5" aria-hidden />
                      {run.branch}
                    </span>
                    {run.pullRequest && <PullRequestLink pullRequest={run.pullRequest} />}
                    <span>Started {formatTimeAgo(new Date(run.startedAt))}</span>
                  </div>
                </div>
                <RunStatusBadge status={run.status} />
              </div>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}

function StartTaskPanel({
  tasks,
  onStartTask,
}: {
  tasks: FactoryStartingTask[];
  onStartTask: (task: FactoryStartingTask) => void;
}) {
  return (
    <section className={panelClassName}>
      <PanelHeading title="Start a task" />
      <p className={cn("-mt-1 mb-3 text-xs leading-normal", mutedTextClassName)}>
        Each task runs as one small, reviewable change ending in a pull request.
      </p>
      <ul className="flex flex-col gap-2">
        {tasks.map((task) => (
          <li key={task.id}>
            <button
              type="button"
              onClick={() => onStartTask(task)}
              className={cn("group w-full px-3.5 py-2.5 text-left", homeListCardClassName)}
            >
              <div className="flex items-center justify-between gap-2">
                <p className="text-sm font-medium text-slate-900 dark:text-gray-100">{task.label}</p>
                <ArrowRight
                  className="size-4 shrink-0 text-gray-400 transition-transform group-hover:translate-x-0.5"
                  aria-hidden
                />
              </div>
              <p className={cn("mt-0.5 text-xs leading-normal", mutedTextClassName)}>{task.description}</p>
            </button>
          </li>
        ))}
      </ul>
    </section>
  );
}

function FactorySetupPanel({ project }: { project: FactoryProject }) {
  return (
    <section className={panelClassName}>
      <PanelHeading title="Connections" />
      <ul className="flex flex-col gap-2">
        {project.integrations.map((integration) => (
          <li key={integration.id} className="flex items-center justify-between gap-3 text-sm">
            <span className="inline-flex items-center gap-2 text-slate-900 dark:text-gray-100">
              <Plug className="size-3.5 text-gray-400" aria-hidden />
              {integration.label}
            </span>
            {integration.connected ? (
              <span className="inline-flex items-center gap-1 text-xs font-medium text-emerald-700 dark:text-emerald-400">
                <CheckCircle2 className="size-3.5" aria-hidden />
                Connected
              </span>
            ) : (
              <span className="inline-flex items-center gap-1 text-xs font-medium text-amber-700 dark:text-amber-400">
                <AlertTriangle className="size-3.5" aria-hidden />
                Not connected
              </span>
            )}
          </li>
        ))}
      </ul>
    </section>
  );
}

function ActivityPanel({ entries }: { entries: FactoryActivityEntry[] }) {
  return (
    <section className={panelClassName}>
      <PanelHeading title="Recent activity" />
      {entries.length === 0 ? (
        <EmptyState
          compact
          icon={GitBranch}
          title="No activity yet"
          description="Runs will show up here as they finish."
        />
      ) : (
        <ul className={cn("flex flex-col divide-y divide-slate-100 text-sm", "dark:divide-gray-700/70")}>
          {entries.map((entry) => (
            <li key={entry.id} className="flex flex-wrap items-baseline justify-between gap-x-4 gap-y-1 py-2.5">
              <p className="text-slate-900 dark:text-gray-100">
                {entry.summary} <span className={mutedTextClassName}>by {entry.actor}</span>
              </p>
              <span className={cn("text-xs", mutedTextClassName)}>{formatTimeAgo(new Date(entry.at))}</span>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
