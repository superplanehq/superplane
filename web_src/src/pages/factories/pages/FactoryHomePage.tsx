import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { usePermissions } from "@/contexts/usePermissions";
import githubIcon from "@/assets/icons/integrations/github.svg";
import sentryIcon from "@/assets/icons/integrations/sentry.svg";
import { useCanvas, useDescribeCanvasVersion, useInfiniteCanvasRuns } from "@/hooks/useCanvasData";
import { usePageTitle } from "@/hooks/usePageTitle";
import { INGESTION_FACTORY_ID, SENTRY_INGESTION_FACTORY_ID } from "@/pages/home/factories";
import { ChevronRight } from "lucide-react";
import { useEffect, useState } from "react";
import { Link } from "react-router";

import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { factoryAppPath, factoryAppRunPath } from "../lib/factoryPagePaths";
import { findActiveRun, formatNextCycle, nextScheduledCycle } from "./ingestionAutomationStatus";
import { type IngestionAutomationId, useIngestionSetup } from "./onboarding/useIngestionSetup";

const INGESTION_OPTIONS: Array<{
  id: IngestionAutomationId;
  title: string;
  description: string;
  icon: string;
  setupLabel: string;
  comingSoon?: boolean;
}> = [
  {
    id: INGESTION_FACTORY_ID,
    title: "GitHub issue ingestion",
    description: "Keep scanning the issue backlog and prepare a fix for small bugs.",
    icon: githubIcon,
    setupLabel: "Set up GitHub ingestion",
  },
  {
    id: SENTRY_INGESTION_FACTORY_ID,
    title: "Sentry issue ingestion",
    description: "Prepare a fix each time Sentry reports a new issue.",
    icon: sentryIcon,
    setupLabel: "Set up Sentry ingestion",
    comingSoon: true,
  },
];

function integrationLabel(name: string): string {
  if (name === "github") return "GitHub";
  if (name === "sentry") return "Sentry";
  if (name === "claude") return "Claude";
  return name;
}

function IngestionStatusLink({ to, children }: { to: string; children: string }) {
  return (
    <Link
      to={to}
      className="inline-flex items-center gap-0.5 font-medium text-blue-700 underline underline-offset-2 hover:no-underline dark:text-blue-300"
    >
      {children}
      <ChevronRight className="size-3" aria-hidden />
    </Link>
  );
}

function IngestionStatus({ automationId, appId }: { automationId: IngestionAutomationId; appId: string | undefined }) {
  const { organizationId, factoryKey } = useFactoriesLayout();
  const [now, setNow] = useState(() => new Date());
  const canvasQuery = useCanvas(organizationId, appId ?? "", { enabled: Boolean(appId) });
  const liveVersionId = canvasQuery.data?.metadata?.liveVersionId;
  const versionQuery = useDescribeCanvasVersion(appId ?? "", liveVersionId, Boolean(appId && liveVersionId));
  const runsQuery = useInfiniteCanvasRuns(appId ?? "", {}, Boolean(appId));

  useEffect(() => {
    if (!appId) return;
    const timer = window.setInterval(() => setNow(new Date()), 30_000);
    return () => window.clearInterval(timer);
  }, [appId]);

  if (!appId) return null;
  if (canvasQuery.isLoading || versionQuery.isLoading || runsQuery.isLoading) {
    return <p className="mt-3 text-[12px] text-muted-foreground">Checking status…</p>;
  }

  const runs = runsQuery.data?.pages.flatMap((page) => page?.runs ?? []) ?? [];
  const activeRun = findActiveRun(runs);
  if (activeRun) {
    const runHref = activeRun.id
      ? factoryAppRunPath(organizationId, factoryKey, appId, activeRun.id, { from: "overview" })
      : factoryAppPath(organizationId, factoryKey, appId, { from: "overview" });

    return (
      <p className="mt-3 flex items-center gap-2 text-[12px]">
        <span className="size-2 animate-pulse rounded-full bg-blue-500" aria-hidden />
        <span className="font-medium text-blue-700 dark:text-blue-300">Working now</span>
        <span className="text-muted-foreground" aria-hidden>
          ·
        </span>
        <IngestionStatusLink to={runHref}>View run</IngestionStatusLink>
      </p>
    );
  }

  if (automationId === SENTRY_INGESTION_FACTORY_ID) {
    return (
      <p className="mt-3 flex items-center gap-2 text-[12px] text-muted-foreground">
        <span className="size-2 rounded-full bg-emerald-500" aria-hidden />
        Waiting for a new Sentry issue
      </p>
    );
  }

  const nodes = versionQuery.data?.spec?.nodes ?? [];
  const createdAt = canvasQuery.data?.metadata?.createdAt;
  const nextCycle = nextScheduledCycle(nodes, runs, createdAt, now);
  const historyHref = factoryAppPath(organizationId, factoryKey, appId, { from: "overview" });
  return (
    <p className="mt-3 flex items-center gap-2 text-[12px] text-muted-foreground">
      <span className="size-2 rounded-full bg-emerald-500" aria-hidden />
      <span>{nextCycle ? formatNextCycle(nextCycle, now) : "Waiting for the next scan"}</span>
      <span aria-hidden>·</span>
      <IngestionStatusLink to={historyHref}>View history</IngestionStatusLink>
    </p>
  );
}

/** Workspace Home: start manual work, or set up optional issue ingestion. */
export function FactoryHomePage() {
  const { factory, openCreateWorkOrder } = useFactoriesLayout();
  usePageTitle(["Home", factory?.name ?? "Workspace"]);
  const { canAct, isLoading: permissionsLoading } = usePermissions();
  const {
    dialogs,
    installedAutomationApps,
    installedAutomationIds,
    integrationsLoading,
    isInstalling,
    missingIntegration,
    setupAutomation,
  } = useIngestionSetup();
  const canCreateTask = canAct("work_orders", "create");
  const canCreateCanvas = canAct("canvases", "create");
  const [taskTitle, setTaskTitle] = useState("");

  function startTask() {
    if (!canCreateTask) return;
    openCreateWorkOrder(taskTitle.trim());
    setTaskTitle("");
  }

  return (
    <div className="flex min-h-full w-full items-center justify-center px-8 py-8">
      <div className="w-full max-w-[720px]" data-testid="factory-home">
        <h1 className="text-[18px] font-semibold tracking-[-0.02em] text-foreground">Start a task</h1>
        <p className="mt-1 text-[13px] text-muted-foreground">
          Describe a task for your coding agent. You can review the plan before the agent starts.
        </p>

        <form
          className="mt-4 flex items-center gap-2"
          onSubmit={(event) => {
            event.preventDefault();
            startTask();
          }}
        >
          <label htmlFor="factory-home-task" className="sr-only">
            Task
          </label>
          <Input
            id="factory-home-task"
            value={taskTitle}
            onChange={(event) => setTaskTitle(event.target.value)}
            placeholder="Fix the timeout on the billing export"
            disabled={!canCreateTask}
            className="h-10 flex-1 text-[13px]"
          />
          <PermissionTooltip
            allowed={canCreateTask || permissionsLoading}
            message="You do not have permission to create tasks."
          >
            <Button type="submit" className="h-10 shrink-0" disabled={!canCreateTask}>
              Create task
            </Button>
          </PermissionTooltip>
        </form>

        <h2 className="mt-8 text-[16px] font-semibold tracking-[-0.01em] text-foreground">Auto Ingestion</h2>

        <div className="mt-4 grid gap-3 sm:grid-cols-2">
          {INGESTION_OPTIONS.map((option) => {
            const installed = installedAutomationIds.has(option.id);
            const missing = missingIntegration(option.id);
            const buttonLabel = integrationsLoading
              ? "Checking connections…"
              : missing
                ? `Connect ${integrationLabel(missing)}`
                : option.setupLabel;

            return (
              <div
                key={option.id}
                className="relative flex flex-col overflow-hidden rounded-lg border border-border bg-background p-4"
              >
                {option.comingSoon ? (
                  <span className="pointer-events-none absolute -right-9 top-4 w-32 rotate-45 bg-accent py-0.5 text-center text-[10px] font-medium tracking-wide text-foreground shadow-sm">
                    Coming soon
                  </span>
                ) : null}
                <img src={option.icon} alt="" className="size-5" aria-hidden />
                <h3 className="mt-3 text-[13px] font-medium text-foreground">{option.title}</h3>
                <p className="mt-1 flex-1 text-[13px] text-muted-foreground">{option.description}</p>
                {option.comingSoon ? null : (
                  <IngestionStatus automationId={option.id} appId={installedAutomationApps.get(option.id)?.id} />
                )}
                {installed || option.comingSoon ? null : (
                  <PermissionTooltip
                    allowed={canCreateCanvas || permissionsLoading}
                    message="You do not have permission to create automations."
                    className="w-full"
                  >
                    <Button
                      type="button"
                      size="sm"
                      variant="outline"
                      className="mt-4 w-full"
                      disabled={!canCreateCanvas || integrationsLoading || isInstalling}
                      onClick={() => void setupAutomation(option.id)}
                    >
                      {isInstalling ? "Setting up…" : buttonLabel}
                    </Button>
                  </PermissionTooltip>
                )}
              </div>
            );
          })}
        </div>
      </div>
      {dialogs}
    </div>
  );
}
