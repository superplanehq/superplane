import { Badge } from "@/components/ui/badge";
import { cn } from "@/lib/utils";
import { CircleAlert, CircleCheck, Minus, TriangleAlert } from "lucide-react";
import { useState } from "react";

import {
  formatCheckScore,
  LEVEL_LABEL,
  type WorkOrderCheckLevel,
  type WorkOrderCheckPresentation,
} from "../../lib/workOrderChecks";
import { getWorkOrderRunHref } from "../../lib/workOrderExecutions";
import { WorkOrderCheckDialog } from "../../WorkOrderCheckDialog";
import { SplitRunAttentionNote } from "./SplitRunAttentionNote";
import {
  splitRunDecisionTone,
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

/**
 * Decision note under Description and Automations. Header stays Close only.
 */
export function SplitRunReview({
  footer,
  className,
  organizationId,
  factoryKey,
  orderNumber,
  canAct = true,
  onStart,
  onReject,
  onBackToDraft,
  onStop,
  startBusy = false,
  actionBusy = false,
  startDisabled = false,
}: {
  footer: SplitRunFooter;
  className?: string;
  organizationId?: string;
  factoryKey?: string;
  orderNumber?: string;
  canAct?: boolean;
  onStart?: () => void | Promise<void>;
  onReject?: () => void | Promise<void>;
  onBackToDraft?: () => void | Promise<void>;
  onStop?: (choice: SplitRunStopChoice) => void | Promise<void>;
  startBusy?: boolean;
  actionBusy?: boolean;
  startDisabled?: boolean;
}) {
  if (!footer.attentionCard || !footer.note) {
    return null;
  }
  const runHref = reviewRunHref(organizationId, factoryKey, footer.run, orderNumber);
  const actions = canAct ? footer.actions : [];
  const onAction = (action: SplitRunFooterAction) => {
    if (action.kind === "start") {
      void onStart?.();
      return;
    }
    if (action.kind === "reject") {
      void onReject?.();
      return;
    }
    if (action.kind === "approve") {
      void onStop?.("completed");
      return;
    }
    if (action.kind === "rerun") {
      void onStop?.("rerun-step");
      return;
    }
    if (action.kind === "reopen") {
      void onStop?.("reopen");
      return;
    }
    if (action.kind === "back-to-draft") {
      void onBackToDraft?.();
    }
  };

  return (
    <div className={cn("shrink-0", className)} data-testid="split-run-review">
      <SplitRunAttentionNote
        note={footer.note}
        tone={splitRunDecisionTone(footer)}
        actions={actions}
        runHref={runHref}
        actionBusy={actionBusy}
        startBusy={startBusy}
        startDisabled={startDisabled}
        onAction={onAction}
      />
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
