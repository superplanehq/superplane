import type { ReactNode } from "react";
import type { FactoriesFactoryPullRequest } from "@/api-client";
import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import { CircleAlert, CircleCheck, Minus, TriangleAlert } from "lucide-react";
import { useState } from "react";

import { draftReadiness, startEmphasisForTone, type DraftReadinessTone } from "../../lib/draftReadiness";
import {
  formatCheckScore,
  LEVEL_LABEL,
  type WorkOrderCheckLevel,
  type WorkOrderCheckPresentation,
} from "../../lib/workOrderChecks";
import { getWorkOrderRunHref } from "../../lib/workOrderExecutions";
import { WorkOrderCheckDialog } from "../../WorkOrderCheckDialog";
import { SplitRunAttentionNote } from "./SplitRunAttentionNote";
import { StartConfirmDialog } from "./StartConfirmDialog";
import { needsStartConfirm, persistSkipStartConfirm } from "./startConfirm";
import {
  splitRunDecisionTone,
  splitRunFooterScores,
  type SplitRunFooter,
  type SplitRunFooterAction,
  type SplitRunStopChoice,
} from "./splitRunFooter";

const PILL_TONE: Record<WorkOrderCheckLevel, string> = {
  positive: "border-emerald-500/30 bg-emerald-500/10 text-emerald-800 dark:text-emerald-300",
  neutral: "border-slate-400/40 bg-slate-500/10 text-slate-700 dark:text-slate-300",
  caution: "border-amber-500/30 bg-amber-500/10 text-amber-800 dark:text-amber-300",
  critical: "border-red-900/40 bg-red-700 text-white hover:bg-red-700/90",
};

const PILL_ICON = {
  positive: CircleCheck,
  neutral: Minus,
  caution: TriangleAlert,
  critical: CircleAlert,
} as const;

function reviewRunHref(
  organizationId: string | undefined,
  factoryKey: string | undefined,
  run: SplitRunFooter["run"],
  orderNumber?: string,
) {
  if (!organizationId || !factoryKey || !run) {
    return null;
  }
  return getWorkOrderRunHref(organizationId, factoryKey, run.appId, run.runId, { orderNumber });
}

type FooterCallback = (() => void | Promise<void>) | undefined;

const STOP_CHOICE: Partial<Record<SplitRunFooterAction["kind"], SplitRunStopChoice>> = {
  approve: "completed",
  rerun: "rerun-step",
  reopen: "reopen",
};

/** Maps a footer action to its callback. Start goes through the confirm gate. */
function footerActionHandler({
  requestStart,
  onArchive,
  onReject,
  onStop,
}: {
  requestStart: () => void;
  onArchive: FooterCallback;
  onReject: FooterCallback;
  onStop?: (choice: SplitRunStopChoice) => void | Promise<void>;
}) {
  const directActions: Partial<Record<SplitRunFooterAction["kind"], FooterCallback>> = {
    archive: onArchive,
    reject: onReject,
  };
  return (action: SplitRunFooterAction) => {
    if (action.kind === "start") {
      requestStart();
      return;
    }
    const directAction = directActions[action.kind];
    if (directAction) {
      void directAction();
      return;
    }
    const stopChoice = STOP_CHOICE[action.kind];
    if (stopChoice) {
      void onStop?.(stopChoice);
    }
  };
}

/** Start weight follows the given verdict, or the verdict the footer scores produce. */
function reviewStartEmphasis(footer: SplitRunFooter, startTone?: DraftReadinessTone) {
  return startEmphasisForTone(startTone ?? draftReadiness(splitRunFooterScores(footer)).tone);
}

function shouldConfirmUnclearStart(confirmUnclearStart: boolean, footer: SplitRunFooter, agentWorking: boolean) {
  return confirmUnclearStart && needsStartConfirm({ ...splitRunFooterScores(footer), agentWorking });
}

function requestUnclearStart({
  confirmUnclearStart,
  footer,
  agentWorking,
  onStart,
  openConfirm,
}: {
  confirmUnclearStart: boolean;
  footer: SplitRunFooter;
  agentWorking: boolean;
  onStart?: () => void | Promise<void>;
  openConfirm: () => void;
}) {
  if (shouldConfirmUnclearStart(confirmUnclearStart, footer, agentWorking)) {
    openConfirm();
    return;
  }
  void onStart?.();
}

/**
 * Decision note under the plan on Description, and under Automations.
 */
export function SplitRunReview({
  footer,
  className,
  organizationId,
  factoryId,
  factoryKey,
  orderId,
  orderNumber,
  pullRequests,
  canAct = true,
  onStart,
  onArchive,
  onReject,
  onStop,
  startBusy = false,
  actionBusy = false,
  startDisabled = false,
  modelSelect,
  compact = false,
  actionsOnly = false,
  confirmUnclearStart = false,
  startTone,
  agentWorking,
}: {
  footer: SplitRunFooter;
  className?: string;
  organizationId?: string;
  factoryId?: string;
  factoryKey?: string;
  orderId?: string;
  orderNumber?: string;
  pullRequests?: FactoriesFactoryPullRequest[];
  canAct?: boolean;
  onStart?: () => void | Promise<void>;
  onArchive?: () => void | Promise<void>;
  onReject?: () => void | Promise<void>;
  onStop?: (choice: SplitRunStopChoice) => void | Promise<void>;
  startBusy?: boolean;
  actionBusy?: boolean;
  startDisabled?: boolean;
  modelSelect?: ReactNode;
  compact?: boolean;
  actionsOnly?: boolean;
  confirmUnclearStart?: boolean;
  /** Verdict that sets the Start weight on the refine strip. Defaults to the footer scores. */
  startTone?: DraftReadinessTone;
  /** True while the refine session starts or runs. Start still warns even when scores exist. */
  agentWorking?: boolean;
}) {
  const [confirmOpen, setConfirmOpen] = useState(false);
  if (!footer.attentionCard || !footer.note) {
    return null;
  }
  const startEmphasis = reviewStartEmphasis(footer, startTone);
  const runHref = reviewRunHref(organizationId, factoryKey, footer.run, orderNumber);
  const actions = canAct
    ? footer.actions.filter((action) => action.kind !== "refine" && action.kind !== "archive")
    : [];
  const requestStart = () =>
    requestUnclearStart({
      confirmUnclearStart,
      footer,
      agentWorking: Boolean(agentWorking),
      onStart,
      openConfirm: () => setConfirmOpen(true),
    });
  const confirmStart = (skipNext: boolean) => {
    if (skipNext) {
      persistSkipStartConfirm();
    }
    setConfirmOpen(false);
    void onStart?.();
  };
  const onAction = footerActionHandler({ requestStart, onArchive, onReject, onStop });

  return (
    <div
      className={cn(compact && !actionsOnly ? "min-w-0 flex-1" : "shrink-0", className)}
      data-testid={actionsOnly ? undefined : "split-run-review"}
    >
      <SplitRunAttentionNote
        note={footer.note}
        tone={splitRunDecisionTone(footer)}
        actions={actions}
        runHref={runHref}
        actionBusy={actionBusy}
        startBusy={startBusy}
        startDisabled={startDisabled}
        modelSelect={modelSelect}
        compact={compact}
        actionsOnly={actionsOnly}
        startEmphasis={startEmphasis}
        organizationId={organizationId}
        factoryId={factoryId}
        orderId={orderId}
        pullRequests={pullRequests}
        canAct={canAct}
        onAction={onAction}
      />
      {confirmUnclearStart ? (
        <StartConfirmDialog
          open={confirmOpen}
          scores={splitRunFooterScores(footer)}
          agentWorking={agentWorking}
          onOpenChange={setConfirmOpen}
          onConfirm={confirmStart}
        />
      ) : null}
    </div>
  );
}

export function SplitRunCheckPills({
  checks,
  label = "Checks",
  testId = "split-run-checks",
}: {
  checks: WorkOrderCheckPresentation[];
  label?: string;
  testId?: string;
}) {
  if (checks.length === 0) {
    return null;
  }

  return (
    <section className="inline-flex max-w-full" aria-label={label} data-testid={testId}>
      <ul className="flex flex-wrap gap-1.5">
        {checks.map((check) => (
          <li key={check.id}>
            <SplitRunCheckPill check={check} />
          </li>
        ))}
      </ul>
    </section>
  );
}

function SplitRunCheckPill({ check }: { check: WorkOrderCheckPresentation }) {
  const [dialogOpen, setDialogOpen] = useState(false);
  const { value, scale } = formatCheckScore(check);
  const level = LEVEL_LABEL[check.level];
  const Icon = PILL_ICON[check.level];
  const score = `${value}${scale}`;

  return (
    <>
      <Badge asChild variant="outline" className={cn("rounded-full px-2 py-0.5", PILL_TONE[check.level])}>
        <button
          type="button"
          onClick={() => setDialogOpen(true)}
          aria-label={`${check.name} ${score}. ${level.label}`}
          data-testid={`split-run-check-${check.id}`}
        >
          <Icon aria-hidden />
          <span>{check.name}</span>
          <span className="tabular-nums">{score}</span>
        </button>
      </Badge>
      <WorkOrderCheckDialog open={dialogOpen} onClose={() => setDialogOpen(false)} check={check} />
    </>
  );
}
