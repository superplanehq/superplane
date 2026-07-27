import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn } from "@/lib/utils";
import { HomePageShell } from "@/pages/home/HomePageShell";
import {
  Activity,
  ArrowLeft,
  Bot,
  CheckCircle2,
  CirclePause,
  CirclePlay,
  Clock3,
  ExternalLink,
  FileCode2,
  GitBranch,
  GitPullRequest,
  Octagon,
  ShieldCheck,
} from "lucide-react";
import { useState } from "react";
import { Link } from "react-router-dom";

import { ActiveWorkTimeline } from "./ActiveWorkTimeline";
import type { ActiveWorkItemData, TimelineEventKind, WorkRunState } from "./activeWorkItemTypes";
import { SteeringComposer } from "./SteeringComposer";

interface ActiveWorkItemPageProps {
  data: ActiveWorkItemData;
}

const panelClassName =
  "overflow-hidden rounded-lg border border-slate-950/15 bg-white dark:border-gray-700/70 dark:bg-gray-900";

export function ActiveWorkItemPage({ data }: ActiveWorkItemPageProps) {
  const [events, setEvents] = useState(data.timeline);
  const [runState, setRunState] = useState<WorkRunState>(data.state);
  const [planApproved, setPlanApproved] = useState(false);
  const [steeringContext, setSteeringContext] = useState<string | null>(null);

  const appendEvent = (title: string, description: string, kind: TimelineEventKind, details?: string[]) => {
    setEvents((current) => [
      ...current,
      {
        id: `event-${current.length + 1}-${title}`,
        time: "Now",
        actor: kind === "user" || kind === "approval" ? "You" : "Factory",
        title,
        description,
        kind,
        details,
      },
    ]);
  };

  const approvePlan = () => {
    setPlanApproved(true);
    setRunState("running");
    appendEvent(
      `Plan v${data.plan.version} approved by you`,
      "The Builder picked up the approved plan and started with route and form-state preservation.",
      "progress",
      ["Builder active", "Plan locked for this run"],
    );
  };

  const sendDirection = (message: string) => {
    appendEvent(
      "Direction added",
      message,
      "user",
      steeringContext ? [`In response to ${steeringContext}`] : ["Applies to current work"],
    );
    setSteeringContext(null);
  };

  const togglePause = () => {
    if (runState === "paused") {
      setRunState(planApproved ? "running" : "awaiting-approval");
      appendEvent("Work resumed", "The factory may continue from the current checkpoint.", "progress");
      return;
    }

    setRunState("paused");
    appendEvent("Paused by you", "Agents will finish their current tool call, then hold this work item.", "user");
  };

  const stopWork = () => {
    setRunState("stopped");
    appendEvent("Work stopped", "This run was stopped by you. Its chronology remains available for audit.", "user");
  };

  return (
    <HomePageShell>
      <div className="mx-auto w-full max-w-[1400px] px-4 py-5 sm:px-6 sm:py-6 lg:px-8">
        <WorkItemHeader data={data} state={runState} onTogglePause={togglePause} onStop={stopWork} />

        <div className="mt-5 grid items-start gap-4 lg:grid-cols-[minmax(0,1fr)_18rem]">
          <section className={panelClassName}>
            <div className="flex min-h-16 flex-col justify-center gap-1 border-b border-slate-200 px-4 py-3 sm:px-6 dark:border-gray-700">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <h2 className="text-sm font-semibold text-slate-900 dark:text-gray-100">Chronology</h2>
                <span className="flex items-center gap-1.5 text-xs text-slate-500 dark:text-gray-400">
                  <Activity className="size-3.5" />
                  Live work record
                </span>
              </div>
              <p className="text-xs text-slate-500 dark:text-gray-400">
                Agent activity, decisions, feedback, and checkpoints in execution order.
              </p>
            </div>

            <ActiveWorkTimeline
              events={events}
              plan={data.plan}
              planApproved={planApproved}
              onApprovePlan={approvePlan}
              onSteer={setSteeringContext}
            />

            <SteeringComposer
              context={steeringContext}
              disabled={runState === "stopped"}
              onClearContext={() => setSteeringContext(null)}
              onSend={sendDirection}
            />
          </section>

          <WorkContextRail data={data} state={runState} planApproved={planApproved} />
        </div>
      </div>
    </HomePageShell>
  );
}

function WorkItemHeader({
  data,
  state,
  onTogglePause,
  onStop,
}: {
  data: ActiveWorkItemData;
  state: WorkRunState;
  onTogglePause: () => void;
  onStop: () => void;
}) {
  const paused = state === "paused";
  const stopped = state === "stopped";

  return (
    <header>
      <div className="flex flex-wrap items-center gap-1.5 text-xs text-slate-500 dark:text-gray-400">
        <Button asChild type="button" variant="ghost" size="xs" className="-ml-2">
          <Link to="../.." relative="path">
            <ArrowLeft />
            {data.projectName}
          </Link>
        </Button>
        <span aria-hidden="true">/</span>
        <span>Active work</span>
        <span aria-hidden="true">/</span>
        <span>{data.id}</span>
      </div>

      <div className="mt-3 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div className="min-w-0">
          <div className="flex flex-wrap items-center gap-2">
            <h1 className="text-2xl font-semibold text-slate-900 dark:text-gray-100">{data.title}</h1>
            <WorkStateBadge state={state} />
          </div>
          <p className="mt-2 max-w-3xl text-sm leading-6 text-slate-600 dark:text-gray-400">{data.goal}</p>
          <div className="mt-2 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-slate-500 dark:text-gray-400">
            <span className="inline-flex items-center gap-1.5">
              <GitBranch className="size-3.5" />
              {data.branch}
            </span>
            <span className="inline-flex items-center gap-1.5">
              <Clock3 className="size-3.5" />
              {data.elapsed} elapsed
            </span>
          </div>
        </div>

        <div className="flex shrink-0 items-center gap-2">
          <Button
            type="button"
            variant="outline"
            aria-label={paused ? "Resume work" : "Pause work"}
            onClick={onTogglePause}
            disabled={stopped}
          >
            {paused ? <CirclePlay /> : <CirclePause />}
            {paused ? "Resume" : "Pause"}
          </Button>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                type="button"
                variant="outline"
                size="icon-sm"
                aria-label="Stop work"
                onClick={onStop}
                disabled={stopped}
                className="text-red-700 hover:text-red-800 dark:text-red-400 dark:hover:text-red-300"
              >
                <Octagon />
              </Button>
            </TooltipTrigger>
            <TooltipContent>Stop work</TooltipContent>
          </Tooltip>
        </div>
      </div>
    </header>
  );
}

function WorkContextRail({
  data,
  state,
  planApproved,
}: {
  data: ActiveWorkItemData;
  state: WorkRunState;
  planApproved: boolean;
}) {
  return (
    <aside aria-label="Work item context" className="space-y-4 lg:sticky lg:top-4">
      <section className={panelClassName}>
        <RailHeader title="Live context" />
        <dl className="divide-y divide-slate-200 dark:divide-gray-700">
          <ContextRow label="State">
            <WorkStateBadge state={state} />
          </ContextRow>
          <ContextRow label="Checkpoint">
            <span className="text-right text-xs font-medium text-slate-700 dark:text-gray-300">
              Plan v{data.plan.version} {planApproved ? "approved" : "review"}
            </span>
          </ContextRow>
          <ContextRow label="Branch">
            <span className="max-w-40 truncate font-mono text-[11px] text-slate-700 dark:text-gray-300">
              {data.branch}
            </span>
          </ContextRow>
          <ContextRow label="Base">
            <span className="font-mono text-[11px] text-slate-700 dark:text-gray-300">{data.baseBranch}</span>
          </ContextRow>
          <ContextRow label="Checks">
            <span className="inline-flex items-center gap-1.5 text-xs font-medium text-emerald-700 dark:text-emerald-400">
              <ShieldCheck className="size-3.5" />
              {data.checksPassed}/{data.checksTotal}
            </span>
          </ContextRow>
          <ContextRow label="Files">
            <span className="inline-flex items-center gap-1.5 text-xs text-slate-700 dark:text-gray-300">
              <FileCode2 className="size-3.5" />
              {data.filesChanged} changed
            </span>
          </ContextRow>
        </dl>
      </section>

      <section className={panelClassName}>
        <RailHeader title="Agents" />
        <div className="divide-y divide-slate-200 dark:divide-gray-700">
          {data.agents.map((agent) => (
            <div key={agent.name} className="flex items-start gap-3 px-4 py-3">
              <span
                className={cn(
                  "mt-0.5 flex size-7 shrink-0 items-center justify-center rounded-full border",
                  agent.active
                    ? "border-sky-300 bg-sky-50 text-sky-700 dark:border-sky-700 dark:bg-sky-950 dark:text-sky-300"
                    : "border-slate-200 bg-slate-50 text-slate-400 dark:border-gray-700 dark:bg-gray-800 dark:text-gray-500",
                )}
              >
                <Bot className="size-3.5" />
              </span>
              <div className="min-w-0 flex-1">
                <div className="flex items-center justify-between gap-2">
                  <p className="text-xs font-medium text-slate-800 dark:text-gray-200">{agent.name}</p>
                  <span
                    className={cn(
                      "size-1.5 shrink-0 rounded-full",
                      agent.active ? "bg-sky-500" : "bg-slate-300 dark:bg-gray-600",
                    )}
                  />
                </div>
                <p className="mt-0.5 text-[11px] text-slate-500 dark:text-gray-400">{agent.role}</p>
                <p className="mt-1 text-[11px] font-medium text-slate-600 dark:text-gray-300">{agent.status}</p>
              </div>
            </div>
          ))}
        </div>
      </section>

      <section className={panelClassName}>
        <RailHeader title="Delivery" />
        <div className="space-y-3 px-4 py-3 text-xs">
          <div className="flex items-center justify-between gap-3">
            <span className="text-slate-500 dark:text-gray-400">Repository</span>
            <a
              href="#repository"
              className="inline-flex min-w-0 items-center gap-1 font-medium text-sky-700 hover:underline dark:text-sky-400"
            >
              <span className="max-w-40 truncate">{data.repository}</span>
              <ExternalLink className="size-3" />
            </a>
          </div>
          <div className="flex items-center justify-between gap-3">
            <span className="text-slate-500 dark:text-gray-400">Pull request</span>
            <span className="inline-flex items-center gap-1.5 font-medium text-slate-700 dark:text-gray-300">
              <GitPullRequest className="size-3.5" />
              {data.pullRequest ?? "Not opened"}
            </span>
          </div>
        </div>
      </section>
    </aside>
  );
}

function RailHeader({ title }: { title: string }) {
  return (
    <div className="flex min-h-12 items-center border-b border-slate-200 px-4 py-2 dark:border-gray-700">
      <h2 className="text-xs font-semibold uppercase text-slate-500 dark:text-gray-400">{title}</h2>
    </div>
  );
}

function ContextRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex min-h-10 items-center justify-between gap-3 px-4 py-2">
      <dt className="text-xs text-slate-500 dark:text-gray-400">{label}</dt>
      <dd className="min-w-0">{children}</dd>
    </div>
  );
}

function WorkStateBadge({ state }: { state: WorkRunState }) {
  const label = {
    "awaiting-approval": "Awaiting plan approval",
    running: "Building",
    paused: "Paused",
    stopped: "Stopped",
  }[state];

  return (
    <Badge
      variant="outline"
      className={cn(
        state === "awaiting-approval" &&
          "border-amber-200 bg-amber-50 text-amber-800 dark:border-amber-800 dark:bg-amber-950/50 dark:text-amber-300",
        state === "running" &&
          "border-sky-200 bg-sky-50 text-sky-700 dark:border-sky-800 dark:bg-sky-950/50 dark:text-sky-300",
        state === "paused" &&
          "border-slate-300 bg-slate-100 text-slate-700 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-300",
        state === "stopped" &&
          "border-red-200 bg-red-50 text-red-700 dark:border-red-900 dark:bg-red-950/50 dark:text-red-300",
      )}
    >
      {state === "running" ? <Activity /> : <CheckCircle2 />}
      {label}
    </Badge>
  );
}
