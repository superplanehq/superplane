import { Badge } from "@/components/reui/badge";
import { Button } from "@/components/ui/button";
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { cn } from "@/lib/utils";
import { MarkdownContent } from "@/pages/app/Markdown";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/ui/collapsible";
import { ChevronRight, CircleStop, RotateCw } from "lucide-react";
import { useState } from "react";

import type { FilesFile } from "@/api-client";

import { formatUsdCents, parseWorkOrderMetric } from "../../../lib/workOrderUsage";
import { formatCheckScore, workOrderCheckStatus, type WorkOrderCheckPresentation } from "../../../lib/workOrderChecks";
import { WorkOrderCheckAnalysis } from "../../../WorkOrderCheckDialog";
import { formatCompactTokenValue } from "@/lib/formatTokenCount";
import { displayRunnerModel } from "../draftStartModel";
import { useLivePhaseSpend } from "../liveHeaderSpendContext";
import { PhaseAgentUsageProvider } from "../phaseAgentUsageContext";
import { PhaseUsageSpendButton } from "../PhaseUsageChartButton";
import { useSpecificModelIds } from "../specificModelIds";
import type { SplitRunPhase } from "../splitRunMocks";
import type { SplitRunSource } from "../splitRunSource";
import { WorkOrderSplitRunDescription } from "../WorkOrderSplitRunDescription";
import { WorkOrderSplitRunSource } from "../WorkOrderSplitRunSource";
import type { AutomationStage, ConsoleAutomation } from "./automationsViewModel";
import { AgentRunsPage } from "./consoleAgentRuns";
import { ArtifactsPage } from "./consoleArtifactRows";
import { runFooterLine, runFooterSpendLabel } from "./consoleCardText";
import { artifactsPageCount, consolePages, type ConsolePageId } from "./consolePages";
import { META_TEXT_CLASSNAME } from "./redesignFormat";

// Line tabs on the shadcn Tabs primitives, after ReUI c-tabs-2.
const PANE_TABS_LIST =
  "h-auto w-full justify-start gap-1 overflow-x-auto rounded-none border-b border-border bg-transparent p-0 dark:bg-transparent";
const PANE_TAB =
  "h-7 flex-none gap-1.5 rounded-none border-0 border-b-2 border-transparent px-2 text-[12px] font-medium text-muted-foreground shadow-none " +
  "data-[state=active]:border-primary data-[state=active]:bg-transparent data-[state=active]:text-foreground data-[state=active]:shadow-none " +
  "dark:data-[state=active]:border-primary dark:data-[state=active]:bg-transparent";

const PAGE_LABEL: Record<ConsolePageId, string> = {
  agent: "Agent runs",
  artifacts: "Artifacts",
  checks: "Checks",
};

export function AutomationCardBody({
  automation,
  phase,
  organizationId,
  factoryId,
  orderId,
  taskDescription,
  canEditDescription = false,
  descriptionBusy = false,
  onDescriptionSave,
  source,
  files,
  onStop,
  onRetry,
  actionBusy,
  phases,
}: {
  automation: ConsoleAutomation;
  phase?: SplitRunPhase;
  phases?: SplitRunPhase[];
  organizationId?: string;
  factoryId?: string;
  orderId?: string;
  taskDescription?: string;
  canEditDescription?: boolean;
  descriptionBusy?: boolean;
  onDescriptionSave?: (next: string) => void | Promise<void>;
  /** Where the task came from. The Backlog creation card shows it under the task text. */
  source?: SplitRunSource;
  files?: FilesFile[];
  onStop?: () => void;
  onRetry?: () => void;
  actionBusy: boolean;
}) {
  const { latest, runs } = automation;
  const pages = consolePages(latest, phase, runs);
  const [chosen, setChosen] = useState<ConsolePageId>();
  const active = chosen && pages.includes(chosen) ? chosen : pages[0];
  return (
    <PhaseAgentUsageProvider>
      <div className="space-y-3">
        {pages.includes("agent") ? null : <StageDescription stage={latest} />}
        <CardPageTabs pages={pages} active={active} stage={latest} runCount={runs.length} onChange={setChosen} />
        {/* The log stays mounted so live steps and spend keep streaming. */}
        {pages.includes("agent") ? (
          <div className={cn(active !== "agent" && "hidden")}>
            <AgentRunsPage
              runs={runs}
              phases={phases ?? (phase ? [phase] : [])}
              automationName={automation.name}
              organizationId={organizationId}
              usagePhaseId={latest.id}
            />
          </div>
        ) : null}
        {active === "artifacts" ? (
          <ArtifactsPage
            stage={latest}
            taskDocument={
              latest.id === "backlog" ? (
                <CreationTaskDocument
                  description={taskDescription}
                  canEdit={canEditDescription}
                  busy={descriptionBusy}
                  onSave={onDescriptionSave}
                  files={files}
                  organizationId={organizationId}
                  factoryId={factoryId}
                  orderId={orderId}
                />
              ) : undefined
            }
          />
        ) : null}
        {active === "checks" ? <ChecksPage checks={latest.checks} /> : null}
        <CreationSource stageId={latest.id} source={source} />
        <CardRunFooter stage={latest} phase={phase} actionBusy={actionBusy} onRetry={onRetry} onStop={onStop} />
      </div>
    </PhaseAgentUsageProvider>
  );
}

function CardPageTabs({
  pages,
  active,
  stage,
  runCount,
  onChange,
}: {
  pages: ConsolePageId[];
  active?: ConsolePageId;
  stage: AutomationStage;
  runCount: number;
  onChange: (page: ConsolePageId) => void;
}) {
  // One page needs no chooser: the content renders bare and the header
  // badges already name it.
  if (pages.length < 2 || !active) {
    return null;
  }
  return (
    <Tabs value={active} onValueChange={(next) => onChange(next as ConsolePageId)}>
      <TabsList aria-label="Run details" className={PANE_TABS_LIST}>
        {pages.map((page) => (
          <TabsTrigger key={page} value={page} className={PANE_TAB}>
            {PAGE_LABEL[page]}
            <PageTabDetail page={page} stage={stage} runCount={runCount} />
          </TabsTrigger>
        ))}
      </TabsList>
    </Tabs>
  );
}

function CreationTaskDocument({
  description,
  canEdit,
  busy,
  onSave,
  files,
  organizationId,
  factoryId,
  orderId,
}: {
  description?: string;
  canEdit: boolean;
  busy: boolean;
  onSave?: (next: string) => void | Promise<void>;
  files?: FilesFile[];
  organizationId?: string;
  factoryId?: string;
  orderId?: string;
}) {
  return (
    <div data-testid="redesign-console-task-description">
      <WorkOrderSplitRunDescription
        description={description ?? ""}
        canEdit={canEdit}
        busy={busy}
        collapsible={false}
        onSave={onSave}
        files={files}
        organizationId={organizationId}
        factoryId={factoryId}
        orderId={orderId}
      />
    </div>
  );
}

function CreationSource({ stageId, source }: { stageId: string; source?: SplitRunSource }) {
  if (stageId !== "backlog" || !source) {
    return null;
  }
  return (
    <div data-testid="redesign-console-card-source">
      <span className="text-[12px] font-medium text-muted-foreground">Source</span>
      <WorkOrderSplitRunSource source={source} />
    </div>
  );
}

function CardRunFooter({
  stage,
  phase,
  actionBusy,
  onRetry,
  onStop,
}: {
  stage: AutomationStage;
  phase?: SplitRunPhase;
  actionBusy: boolean;
  onRetry?: () => void;
  onStop?: () => void;
}) {
  const { lead, spendLabel, model } = useCardFooterMeta(stage, phase);
  if (!lead && !spendLabel && !model && !onRetry && !onStop) {
    return null;
  }
  return (
    <div
      className="flex flex-wrap items-center gap-x-3 gap-y-2 border-t pt-3"
      data-testid={`redesign-console-run-footer-${stage.id}`}
    >
      <CardFooterMeta
        stageId={stage.id}
        live={stage.status === "running"}
        lead={lead}
        spendLabel={spendLabel}
        model={model}
      />
      <CardFooterActions actionBusy={actionBusy} onRetry={onRetry} onStop={onStop} />
    </div>
  );
}

function useCardFooterMeta(stage: AutomationStage, phase?: SplitRunPhase) {
  const live = useLivePhaseSpend(stage.id);
  const tokens = Math.max(parseWorkOrderMetric(phase?.totalTokens), live.tokens);
  const cents = Math.max(parseWorkOrderMetric(phase?.costCents), live.cents);
  const modelIds = useSpecificModelIds();
  return {
    lead: runFooterLine(stage),
    spendLabel: runFooterSpendLabel(
      cents > 0 ? formatUsdCents(cents) : undefined,
      tokens > 0 ? formatCompactTokenValue(tokens) : undefined,
    ),
    model: displayRunnerModel(phase?.model ?? stage.model ?? "", modelIds),
  };
}

function CardFooterMeta({
  stageId,
  live,
  lead,
  spendLabel,
  model,
}: {
  stageId: string;
  live: boolean;
  lead?: string;
  spendLabel?: string;
  model?: string;
}) {
  return (
    <span className={cn(META_TEXT_CLASSNAME, "inline-flex min-w-0 flex-wrap items-center gap-x-1")}>
      {lead ? <span>{lead}</span> : null}
      {lead && spendLabel ? <span aria-hidden>·</span> : null}
      {spendLabel ? (
        <PhaseUsageSpendButton
          phaseId={stageId}
          spendLabel={spendLabel}
          live={live}
          className={cn(META_TEXT_CLASSNAME, "font-medium text-current underline underline-offset-2")}
        />
      ) : null}
      {(lead || spendLabel) && model ? <span aria-hidden>·</span> : null}
      {model ? <span>{model}</span> : null}
    </span>
  );
}

function CardFooterActions({
  actionBusy,
  onRetry,
  onStop,
}: {
  actionBusy: boolean;
  onRetry?: () => void;
  onStop?: () => void;
}) {
  return (
    <div className="ms-auto flex shrink-0 items-center gap-1.5">
      {onRetry ? (
        <Button size="sm" variant="outline" className="gap-1.5" disabled={actionBusy} onClick={onRetry}>
          <RotateCw className="size-3.5" aria-hidden />
          Retry
        </Button>
      ) : null}
      {onStop ? (
        <Button size="sm" variant="outline" className="gap-1.5" disabled={actionBusy} onClick={onStop}>
          <CircleStop className="size-3.5" aria-hidden />
          Stop
        </Button>
      ) : null}
    </div>
  );
}

/** Type-specific tab detail: a count on Agent runs, Artifacts, and Checks. */
function PageTabDetail({ page, stage, runCount }: { page: ConsolePageId; stage: AutomationStage; runCount: number }) {
  const count = page === "agent" ? runCount : page === "artifacts" ? artifactsPageCount(stage) : stage.checks.length;
  return <span className="tabular-nums opacity-60">{count}</span>;
}

/** The stage description always shows above the tabs. It is context, not a page. */
function StageDescription({ stage }: { stage: AutomationStage }) {
  if (stage.id === "backlog" || !stage.description?.trim()) {
    return null;
  }
  return (
    <div className="text-[12.5px] leading-5 text-muted-foreground">
      <MarkdownContent content={stage.description} variant="workspace" />
    </div>
  );
}

/** Every score on one page: value and verdict up front, the analysis on demand. */
function ChecksPage({ checks }: { checks: WorkOrderCheckPresentation[] }) {
  return (
    <div className="flex flex-col divide-y divide-border">
      {checks.map((check) => (
        <CheckSection key={check.id} check={check} defaultOpen={checks.length === 1} />
      ))}
    </div>
  );
}

function CheckSection({ check, defaultOpen }: { check: WorkOrderCheckPresentation; defaultOpen: boolean }) {
  const { value, scale } = formatCheckScore(check);
  const status = workOrderCheckStatus(check);
  const expandable = Boolean(check.analysis?.trim());
  const header = (
    <>
      <span className="text-[12px] font-medium text-foreground">{check.name}</span>
      <span className={cn("text-[13px] font-semibold tabular-nums", status.className)}>
        {value}
        {scale}
      </span>
      <Badge variant="outline" className={cn("border", status.badgeClassName)}>
        {status.label}
      </Badge>
    </>
  );
  return (
    <Collapsible defaultOpen={defaultOpen && expandable} className="py-2.5 first:pt-0 last:pb-0">
      {expandable ? (
        <CollapsibleTrigger className="group flex w-full cursor-pointer items-center gap-2 text-left">
          {header}
          <ChevronRight
            className="ms-auto size-3.5 shrink-0 text-muted-foreground transition-transform group-data-[state=open]:rotate-90"
            aria-hidden
          />
        </CollapsibleTrigger>
      ) : (
        <div className="flex items-center gap-2">{header}</div>
      )}
      {check.summary ? <p className="mt-1.5 text-[12px] leading-5 text-muted-foreground">{check.summary}</p> : null}
      {expandable ? (
        <CollapsibleContent>
          <WorkOrderCheckAnalysis check={check} />
        </CollapsibleContent>
      ) : null}
    </Collapsible>
  );
}
