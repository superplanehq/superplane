import type { FactoriesFactoryPullRequest, FactoriesWorkOrder, FactoriesWorkOrderArtifact } from "@/api-client";
import { Button } from "@/components/ui/button";
import { usePermissions } from "@/contexts/usePermissions";
import { useFactoryBacklogAnalysis } from "@/hooks/useBacklogAnalysisRuns";
import { useFactoryAutomations, useWorkOrder, useWorkOrderArtifacts } from "@/hooks/useFactoryData";
import { useFactoryPRFeedbackHandlers } from "@/hooks/useFactoryPRFeedbackData";
import { useOrgUserLookup } from "@/hooks/useOrgUserLookup";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useWorkOrderCardActions } from "@/hooks/useWorkOrderCardActions";
import { cn } from "@/lib/utils";
import { MarkdownContent } from "@/pages/app/Markdown";
import { ArrowLeft, ChevronDown, ExternalLink } from "lucide-react";
import { useState, type ReactNode } from "react";
import { useLocation, useNavigate, useParams } from "react-router";

import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { factoryHomePath, firstFactoryLineId, workOrderBoardLineIdFromSearch } from "../lib/factoryPagePaths";
import { getWorkOrderDisplayStatus, getWorkOrderDisplayStatusMeta } from "../lib/workOrderProgress";
import { formatWorkOrderIdentifier } from "../lib/workspaceKey";
import { PhaseGlyph } from "../pages/linePhaseGlyph";
import type { PhaseGlyphKind } from "../lib/linePhaseRuns";
import { OutputList, OwnerTimeCostRow } from "../pages/work-order-popup-redesign/popupShared";
import { SplitRunCheckPills, SplitRunReview } from "../pages/work-order-split-run/SplitRunReview";
import { DRAFT_START_MODEL_AUTO } from "../pages/work-order-split-run/draftStartModel";
import {
  columnAppsFromFactoryApps,
  splitRunFixtureForWorkOrder,
  splitRunStatusLabel,
  type SplitRunFixture,
  type SplitRunPhase,
  type SplitRunPhaseStatus,
} from "../pages/work-order-split-run/splitRunMocks";
import { useSplitRunFooterActions } from "../pages/work-order-split-run/useSplitRunFooterActions";
import { useSplitRunFooterCloser } from "../pages/work-order-split-run/useSplitRunFooterCloser";
import { draftStartAction, footerMutationHandlers } from "../pages/work-order-split-run/workOrderPopupActions";
import { useColumnAppCheckRuns } from "../pages/work-order-split-run/useColumnAppCheckRuns";
import { useWorkOrderPRFeedbackLog } from "../pages/useWorkOrderPRFeedbackRunHref";
import { MOBILE_TASK_COPY } from "./mobileCopy";

const PHASE_GLYPH: Record<SplitRunPhaseStatus, PhaseGlyphKind> = {
  passed: "passed",
  running: "running",
  pending: "pending",
  waiting: "waiting",
  failed: "failed",
  cancelled: "cancelled",
};

/**
 * Full-screen task view for the phone shell. Everything stacks in one
 * scrolling column so titles, notes, and the activity log stay readable on a
 * narrow screen. Actions come from the same footer model as the desktop popup.
 */
export function MobileTaskDetailPage() {
  const { organizationId, factoryId, routeSegment, factory } = useFactoriesLayout();
  const { orderNumber = "" } = useParams<{ orderNumber?: string }>();
  const { search } = useLocation();
  const navigate = useNavigate();
  const boardLineId = workOrderBoardLineIdFromSearch(search) ?? firstFactoryLineId(factory);
  const { data: order, isLoading, isError } = useWorkOrder(organizationId, factoryId, orderNumber);
  const backToBoard = () => navigate(factoryHomePath(organizationId, routeSegment, boardLineId));

  usePageTitle([order?.title ?? "Task", factory?.name ?? "Workspace"]);

  if (isLoading && !order) {
    return (
      <MobileTaskFrame onBack={backToBoard}>
        <p className="px-4 py-8 text-[13px] text-muted-foreground">{MOBILE_TASK_COPY.loading}</p>
      </MobileTaskFrame>
    );
  }
  if (!order?.id || isError) {
    return (
      <MobileTaskFrame onBack={backToBoard}>
        <div className="px-4 py-8" data-testid="mobile-task-not-found">
          <p className="text-[15px] font-semibold text-foreground">{MOBILE_TASK_COPY.notFound}</p>
          <p className="mt-1 text-[13px] text-muted-foreground">{MOBILE_TASK_COPY.notFoundHelp}</p>
        </div>
      </MobileTaskFrame>
    );
  }
  return (
    <LoadedMobileTask
      key={order.id}
      order={order}
      orderId={order.id}
      lineId={boardLineId}
      lineName={factory?.lines?.find((line) => line.id === boardLineId)?.name}
      onBack={backToBoard}
    />
  );
}

type MobileTaskModel = {
  fixture: SplitRunFixture;
  artifacts: FactoriesWorkOrderArtifact[];
  pullRequests: FactoriesFactoryPullRequest[];
  canUpdate: boolean;
  lineName?: string;
  onStart: ReturnType<typeof draftStartAction>;
  onArchive: ReturnType<typeof footerMutationHandlers>["onArchive"];
  onReject: ReturnType<typeof footerMutationHandlers>["onReject"];
  onStop: ReturnType<typeof footerMutationHandlers>["onStop"];
  startBusy: boolean;
  actionBusy: boolean;
  columnAppRunQueries: ReturnType<typeof useColumnAppCheckRuns>["queries"];
};

/** Loads everything the task screen shows and wires the footer actions. */
function useMobileTaskModel(
  order: FactoriesWorkOrder,
  orderId: string,
  lineId: string | undefined,
  rawLineName: string | undefined,
  onDone: () => void,
): MobileTaskModel {
  const { organizationId, factoryId } = useFactoriesLayout();
  const { canAct } = usePermissions();
  const canUpdate = canAct("work_orders", "update");
  const lineName = rawLineName?.trim() || undefined;
  const { data: artifacts = [] } = useWorkOrderArtifacts(organizationId, factoryId, orderId);
  const { data: handlers = [] } = useFactoryPRFeedbackHandlers(organizationId, factoryId);
  const { data: apps = [] } = useFactoryAutomations(organizationId, factoryId);
  const pullRequests = order.pullRequests ?? [];
  const prFeedbackRuns = useWorkOrderPRFeedbackLog(pullRequests, handlers);
  const closer = useSplitRunFooterCloser(organizationId, factoryId, order);
  const { resolveUser } = useOrgUserLookup(organizationId);
  const backlogAnalysis = useFactoryBacklogAnalysis(organizationId, factoryId);
  const cardActions = useWorkOrderCardActions(organizationId, factoryId);
  const footerActions = useSplitRunFooterActions(organizationId, factoryId, orderId);
  const columnApps = columnAppsFromFactoryApps(apps);
  const columnAppCheckRuns = useColumnAppCheckRuns(order.checks, columnApps);

  const fixture = splitRunFixtureForWorkOrder(order, {
    checks: order.checks,
    artifacts,
    lineId,
    lineName,
    demoArtifacts: false,
    prFeedbackRuns,
    analysisRuns: backlogAnalysis.runsByWorkOrder.get(orderId) ?? [],
    columnApps,
    isAnalyzing: backlogAnalysis.analyzingOrderIds.has(orderId),
    stoppedBy: closer.actor,
    closer,
    resolveUser,
    columnAppRuns: columnAppCheckRuns.lookup,
  });
  const mutations = footerMutationHandlers(canUpdate, footerActions, fixture, onDone);
  const onStart = draftStartAction(
    fixture.footer.kind,
    lineName
      ? (model, thinkingLevel) => cardActions.onDispatch(orderId, { lineName, model, thinkingLevel })
      : undefined,
    DRAFT_START_MODEL_AUTO,
  );

  return {
    fixture,
    artifacts,
    pullRequests,
    canUpdate,
    lineName,
    onStart,
    onArchive: mutations.onArchive,
    onReject: mutations.onReject,
    onStop: mutations.onStop,
    startBusy: cardActions.dispatchingOrderIds.has(orderId),
    actionBusy: footerActions.busy,
    columnAppRunQueries: columnAppCheckRuns.queries,
  };
}

function LoadedMobileTask({
  order,
  orderId,
  lineId,
  lineName,
  onBack,
}: {
  order: FactoriesWorkOrder;
  orderId: string;
  lineId?: string;
  lineName?: string;
  onBack: () => void;
}) {
  const { organizationId, factoryId, routeSegment } = useFactoriesLayout();
  const model = useMobileTaskModel(order, orderId, lineId, lineName, onBack);
  const { fixture, artifacts, pullRequests } = model;
  const activity = fixture.phases.filter((phase) => !phase.historyRun);

  return (
    <MobileTaskFrame onBack={onBack}>
      {model.columnAppRunQueries}
      <article className="flex flex-col gap-5 px-4 pt-3 pb-8" data-testid="mobile-task-detail">
        <MobileTaskHeader order={order} fixture={fixture} />

        <SplitRunReview
          footer={fixture.footer}
          organizationId={organizationId}
          factoryId={factoryId}
          factoryKey={routeSegment}
          orderId={orderId}
          orderNumber={order.number}
          pullRequests={pullRequests}
          canAct={model.canUpdate}
          onStart={model.onStart}
          onArchive={model.onArchive}
          onReject={model.onReject}
          onStop={model.onStop}
          startBusy={model.startBusy}
          actionBusy={model.actionBusy}
          startDisabled={!model.canUpdate || !model.lineName}
          compact="stacked"
        />

        <Section title={MOBILE_TASK_COPY.description}>
          {fixture.descriptionText?.trim() ? (
            <MarkdownContent content={fixture.descriptionText} variant="workspace" organizationId={organizationId} />
          ) : (
            <p className="text-[13px] text-muted-foreground">{MOBILE_TASK_COPY.noDescription}</p>
          )}
        </Section>

        <Section title={MOBILE_TASK_COPY.activity}>
          {activity.length === 0 ? (
            <p className="text-[13px] text-muted-foreground">{MOBILE_TASK_COPY.noActivity}</p>
          ) : (
            <ol className="flex flex-col divide-y divide-border rounded-lg border border-border bg-card">
              {activity.map((phase) => (
                <li key={phase.id}>
                  <ActivityRow phase={phase} expandedByDefault={phase.id === fixture.currentPhaseId} />
                </li>
              ))}
            </ol>
          )}
        </Section>

        {pullRequests.length > 0 ? (
          <Section title={MOBILE_TASK_COPY.pullRequests}>
            <PullRequestList pullRequests={pullRequests} />
          </Section>
        ) : null}

        {artifacts.length > 0 ? (
          <Section title={MOBILE_TASK_COPY.files}>
            <FileList artifacts={artifacts} />
          </Section>
        ) : null}
      </article>
    </MobileTaskFrame>
  );
}

/** Key, status, full title, and the owner/time/cost row. The title wraps instead of truncating. */
function MobileTaskHeader({ order, fixture }: { order: FactoriesWorkOrder; fixture: SplitRunFixture }) {
  const { organizationId, factoryKey } = useFactoriesLayout();
  const statusMeta = getWorkOrderDisplayStatusMeta(getWorkOrderDisplayStatus(order));
  const identifier = fixture.identifier?.trim() || formatWorkOrderIdentifier(factoryKey, order.number);

  return (
    <header className="flex flex-col gap-2">
      <div className="flex items-center gap-2 text-[12px]">
        {identifier ? (
          <span className="font-mono text-muted-foreground" data-testid="mobile-task-key">
            {identifier}
          </span>
        ) : null}
        <span
          className={cn("inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 font-medium", statusMeta.className)}
          data-testid="mobile-task-status"
        >
          <span className={cn("size-1.5 rounded-full", statusMeta.dotClassName)} aria-hidden />
          {statusMeta.label}
        </span>
      </div>
      <h1 className="text-[20px] leading-snug font-semibold tracking-[-0.02em] text-foreground break-words">
        {fixture.title}
      </h1>
      <OwnerTimeCostRow
        fixture={fixture}
        className="mt-0"
        organizationId={organizationId}
        assigneeIds={fixture.assigneeIds}
        usageByModel={fixture.usageByModel}
        usageByMachineType={fixture.usageByMachineType}
      >
        <span className="text-muted-foreground">
          {fixture.lineName} · {fixture.startedLabel}
        </span>
      </OwnerTimeCostRow>
    </header>
  );
}

function MobileTaskFrame({ onBack, children }: { onBack: () => void; children: ReactNode }) {
  return (
    <div className="flex h-full min-h-0 flex-col" data-testid="mobile-task-page">
      <div className="flex h-12 shrink-0 items-center border-b border-border px-2 pt-[env(safe-area-inset-top)]">
        <Button
          type="button"
          variant="ghost"
          size="sm"
          onClick={onBack}
          className="gap-1.5 text-muted-foreground"
          data-testid="mobile-task-back"
        >
          <ArrowLeft className="size-4" aria-hidden />
          {MOBILE_TASK_COPY.back}
        </Button>
      </div>
      <div className="min-h-0 flex-1 overflow-y-auto">{children}</div>
    </div>
  );
}

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="flex flex-col gap-2">
      <h2 className="workspace-section-title">{title}</h2>
      {children}
    </section>
  );
}

/** One automation on the line. Tap to see its steps. */
function ActivityRow({ phase, expandedByDefault }: { phase: SplitRunPhase; expandedByDefault: boolean }) {
  const [expanded, setExpanded] = useState(expandedByDefault);
  const steps = phase.stream.filter((line) => !line.note);
  const canExpand = steps.length > 0;

  return (
    <div className="flex flex-col" data-testid={`mobile-task-phase-${phase.id}`}>
      <button
        type="button"
        onClick={() => canExpand && setExpanded((current) => !current)}
        aria-expanded={canExpand ? expanded : undefined}
        aria-label={canExpand ? (expanded ? MOBILE_TASK_COPY.hideSteps : MOBILE_TASK_COPY.showSteps) : undefined}
        className="flex w-full items-start gap-3 px-3 py-2.5 text-left"
      >
        <PhaseGlyph kind={PHASE_GLYPH[phase.status]} className="mt-1" />
        <span className="min-w-0 flex-1">
          <span className="block text-[14px] font-medium text-foreground break-words">{phase.name}</span>
          <span className="block text-[12px] text-muted-foreground">
            {splitRunStatusLabel(phase.status)}
            {phase.componentName ? ` · ${phase.componentName}` : ""}
            {phase.duration ? ` · ${phase.duration}` : ""}
          </span>
          {phase.checks && phase.checks.length > 0 ? (
            <span className="mt-1.5 block">
              <SplitRunCheckPills checks={phase.checks} testId={`mobile-task-phase-checks-${phase.id}`} />
            </span>
          ) : null}
        </span>
        {canExpand ? (
          <ChevronDown
            className={cn("mt-1 size-4 shrink-0 text-muted-foreground transition-transform", expanded && "rotate-180")}
            aria-hidden
          />
        ) : null}
      </button>
      {expanded && canExpand ? (
        <ol className="flex flex-col gap-2 border-t border-border bg-muted/30 px-3 py-2.5">
          {steps.map((line) => (
            <li key={line.id} className="flex items-start gap-2 text-[13px]">
              <PhaseGlyph kind={PHASE_GLYPH[line.status]} className="mt-0.5 size-3" />
              <span className="min-w-0 flex-1 break-words">
                <span className="text-foreground">{line.componentName}</span>
                {line.detail ? <span className="block text-[12px] text-muted-foreground">{line.detail}</span> : null}
              </span>
              {line.duration ? (
                <span className="shrink-0 text-[12px] tabular-nums text-muted-foreground">{line.duration}</span>
              ) : null}
            </li>
          ))}
        </ol>
      ) : null}
    </div>
  );
}

function PullRequestList({ pullRequests }: { pullRequests: FactoriesFactoryPullRequest[] }) {
  return (
    <ul className="flex flex-col gap-2">
      {pullRequests.map((pullRequest) => (
        <li key={pullRequest.id ?? pullRequest.url}>
          <a
            href={pullRequest.url}
            target="_blank"
            rel="noreferrer"
            className="flex items-start gap-2 rounded-lg border border-border bg-card px-3 py-2.5 text-[13px] text-foreground"
          >
            <span className="min-w-0 flex-1 break-words">
              <span className="block font-medium">
                {pullRequest.title || `Pull request #${pullRequest.number ?? ""}`}
              </span>
              <span className="block text-[12px] text-muted-foreground">
                {[
                  pullRequest.repository,
                  pullRequest.number ? `#${pullRequest.number}` : null,
                  pullRequestStateLabel(pullRequest),
                ]
                  .filter(Boolean)
                  .join(" · ")}
              </span>
            </span>
            <ExternalLink className="mt-0.5 size-3.5 shrink-0 text-muted-foreground" aria-hidden />
          </a>
        </li>
      ))}
    </ul>
  );
}

function pullRequestStateLabel(pullRequest: FactoriesFactoryPullRequest): string | null {
  if (pullRequest.state === "STATE_MERGED") return "Merged";
  if (pullRequest.state === "STATE_CLOSED") return "Closed";
  if (pullRequest.state === "STATE_DRAFT") return "Draft";
  if (pullRequest.state === "STATE_OPEN") return "Open";
  return null;
}

function FileList({ artifacts }: { artifacts: FactoriesWorkOrderArtifact[] }) {
  return (
    <div className="rounded-lg border border-border bg-card px-3">
      <OutputList artifacts={artifacts} />
    </div>
  );
}
