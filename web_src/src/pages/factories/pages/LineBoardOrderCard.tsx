import type { FactoriesWorkOrderCheckScore, FactoriesWorkOrderSummary } from "@/api-client";
import { useRevealAfterPending } from "@/hooks/useRevealAfterPending";
import { cn } from "@/lib/utils";
import { Skeleton } from "@/ui/skeleton";
import { useMemo, type ComponentProps, type ReactNode } from "react";

import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import {
  boardCardLoadsConfidenceChecks,
  clarityScoreFromChecks,
  confidenceScoreFromChecks,
} from "../lib/confidenceScore";
import { LOADING_REVEAL_CLASSNAME } from "../lib/loadingReveal";
import { buildWorkOrderListEntry } from "../lib/workOrderListModel";
import { WorkOrderCard, type WorkOrderCardContext } from "../workOrders/WorkOrderCard";
import { draftCardAgentIsWorking, planningSessionHasPendingSurvey } from "./planningSessionView";
import { factoryShowsClarity, factoryShowsConfidence } from "./planningSettingsModel";

const FILTER_CARD_SKELETON_COUNT = 3;
const FILTER_LOADING_LABEL = "Loading tasks";

export function LineBoardOrderCardSkeleton() {
  return (
    <div className="w-full rounded-md border border-border bg-card p-2.5 shadow-sm">
      <Skeleton className="h-4 w-3/4" />
      <div className="mt-3 flex items-center justify-between gap-2">
        <Skeleton className="h-3 w-16" />
        <Skeleton className="size-5 rounded-full" />
      </div>
    </div>
  );
}

export function LineBoardColumnCardList({
  pending,
  className,
  testId,
  onScroll,
  children,
}: {
  pending: boolean;
  className: string;
  testId?: string;
  onScroll?: (element: HTMLElement) => void;
  children: ReactNode;
}) {
  const reveal = useRevealAfterPending(pending);

  return (
    <ul
      className={cn(className, reveal && LOADING_REVEAL_CLASSNAME)}
      data-testid={testId}
      data-reveal={reveal ? "" : undefined}
      role={pending ? "status" : undefined}
      aria-label={pending ? FILTER_LOADING_LABEL : undefined}
      aria-busy={pending || undefined}
      onScroll={onScroll ? (event) => onScroll(event.currentTarget) : undefined}
    >
      {pending
        ? Array.from({ length: FILTER_CARD_SKELETON_COUNT }, (_, index) => (
            <li key={index}>
              <LineBoardOrderCardSkeleton />
            </li>
          ))
        : children}
    </ul>
  );
}

export function LineBoardOrderCard({
  order,
  workOrderCardContext,
  onOpenWorkOrder,
  isAnalyzing = false,
}: {
  order: FactoriesWorkOrderSummary;
  workOrderCardContext: WorkOrderCardContext;
  onOpenWorkOrder: (orderId: string, order?: FactoriesWorkOrderSummary) => void;
  isAnalyzing?: boolean;
}) {
  return (
    <LineBoardWorkOrderCard
      order={order}
      workOrderCardContext={workOrderCardContext}
      isAnalyzing={isAnalyzing}
      onOpen={() => {
        if (order.id) {
          onOpenWorkOrder(order.id, order);
        }
      }}
    />
  );
}

export function LineBoardWorkOrderCard({
  order,
  workOrderCardContext,
  onOpen,
  isAnalyzing = false,
}: {
  order: FactoriesWorkOrderSummary;
  workOrderCardContext: WorkOrderCardContext;
  onOpen: () => void;
  isAnalyzing?: boolean;
}) {
  const { factory } = useFactoriesLayout();
  const entry = useMemo(() => buildWorkOrderListEntry(order, factory), [factory, order]);
  const showConfidence = boardCardLoadsConfidenceChecks(entry.displayStatus);
  const session = order.planningSession;
  const scores = cardScores(showConfidence, order.checkScores, session, isAnalyzing, {
    showClarity: factoryShowsClarity(factory),
    showConfidence: factoryShowsConfidence(factory),
  });

  return (
    <WorkOrderCard
      {...workOrderCardContext}
      entry={entry}
      {...scores}
      hasAgentQuestion={showConfidence && planningSessionHasPendingSurvey(session)}
      onOpen={onOpen}
    />
  );
}

/**
 * Scores come from the list. The thinking state follows the planning session
 * summary on that list. A local backlog run still counts until the first score.
 * The opened card loads the full session.
 */
function cardScores(
  showConfidence: boolean,
  checks: FactoriesWorkOrderCheckScore[] | undefined,
  session: FactoriesWorkOrderSummary["planningSession"],
  backlogAnalyzing: boolean,
  visibility: { showClarity: boolean; showConfidence: boolean },
): Pick<
  ComponentProps<typeof WorkOrderCard>,
  "clarityScore" | "confidenceScore" | "isAnalyzing" | "showClarity" | "showConfidenceScore"
> {
  if (!showConfidence) {
    return { isAnalyzing: false, showClarity: false, showConfidenceScore: false };
  }
  const clarityScore = visibility.showClarity ? clarityScoreFromChecks(checks) : undefined;
  const confidenceScore = visibility.showConfidence ? confidenceScoreFromChecks(checks) : undefined;
  return {
    clarityScore,
    confidenceScore,
    showClarity: visibility.showClarity,
    showConfidenceScore: visibility.showConfidence,
    isAnalyzing: draftCardAgentIsWorking(session, backlogAnalyzing, clarityScore ?? confidenceScore),
  };
}
