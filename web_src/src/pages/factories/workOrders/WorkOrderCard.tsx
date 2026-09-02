import type { FactoriesFactoryLine } from "@/api-client";
import { formatTimeAgo } from "@/lib/date";
import { Link } from "react-router";
import { getWorkOrderAttentionReasons, type WorkOrderAttentionReason } from "../lib/workOrderAttention";
import { workOrderOpenPath } from "../lib/factoryPagePaths";
import type { WorkOrderListEntry } from "../lib/workOrderListModel";
import { getWorkOrderDisplayStatusMeta } from "../lib/workOrderProgress";
import { ConfidenceAnalyzingIndicator, ConfidenceMeter } from "./ConfidenceMeter";
import { WorkOrderAttentionChip, WorkOrderChecksPassedMark } from "./WorkOrderAttentionChip";
import { CardOwnerMark, StartDraftButton, type WorkOrderRowCallbacks } from "./WorkOrderRowActions";
import { WorkOrderStatusDot } from "./WorkOrderStatusDot";

const EMPTY_ADDRESSING_FEEDBACK_IDS: ReadonlySet<string> = new Set();
const EMPTY_ADDRESSING_FEEDBACK_LABELS: ReadonlyMap<string, string> = new Map();
const EMPTY_WAITING_ON_CHECKS_IDS: ReadonlySet<string> = new Set();
const EMPTY_CHECKS_PASSED_IDS: ReadonlySet<string> = new Set();
const EMPTY_FIXES_PAUSED_IDS: ReadonlySet<string> = new Set();

export interface WorkOrderCardContext extends WorkOrderRowCallbacks {
  organizationId: string;
  factoryId?: string;
  factoryKey: string;
  factoryLines: FactoriesFactoryLine[];
  /** When set, Start on a draft sends the task to this line. */
  preferredLineName?: string;
  canDispatch: boolean;
  canAssign: boolean;
  /** Tasks with a dispatch in flight. Only their controls show a busy state. */
  dispatchingOrderIds: ReadonlySet<string>;
  isAssigneesSaving: boolean;
  /** Tasks with a queued or running discussion or exclusive-repair run. */
  addressingFeedbackOrderIds?: ReadonlySet<string>;
  /** Activity text for an addressing run, keyed by task id. */
  addressingFeedbackLabels?: ReadonlyMap<string, string>;
  /** Tasks that wait for pull request checks and do not address comments yet. */
  waitingOnChecksOrderIds?: ReadonlySet<string>;
  /** Tasks whose latest check wait finished with passing checks. */
  checksPassedOrderIds?: ReadonlySet<string>;
  /** Tasks whose check handler stopped at the attempt limit. */
  fixesPausedOrderIds?: ReadonlySet<string>;
}

export interface WorkOrderCardProps extends WorkOrderCardContext {
  entry: WorkOrderListEntry;
  /**
   * Overlay destination. Defaults to the task. The Lines board
   * passes onOpen to show the card dialog instead of navigating.
   */
  href?: string;
  /** When set, the card overlay opens this handler instead of navigating. */
  onOpen?: () => void;
  /** Confidence score from ListWorkOrderChecks, 0 to 5. Shown left of Start. */
  confidenceScore?: number;
  /**
   * True while the Backlog automation analyzes this task. The card
   * shows a spinner in the meter slot until the score arrives.
   */
  isAnalyzing?: boolean;
}

/**
 * The canonical task card.
 *
 * Every board uses this complete component. Status is a colored dot next
 * to the title. The footer shows the owner (except on drafts), the age of
 * the task, and a Start button on drafts. Reviewed drafts also
 * show a score to the left of Start. Waiting cards show an attention
 * label such as Waiting for user review. The owner is display-only on
 * the card.
 */
export function WorkOrderCard({
  entry,
  organizationId,
  factoryKey,
  factoryLines,
  preferredLineName,
  canDispatch,
  dispatchingOrderIds,
  addressingFeedbackOrderIds = EMPTY_ADDRESSING_FEEDBACK_IDS,
  addressingFeedbackLabels = EMPTY_ADDRESSING_FEEDBACK_LABELS,
  waitingOnChecksOrderIds = EMPTY_WAITING_ON_CHECKS_IDS,
  checksPassedOrderIds = EMPTY_CHECKS_PASSED_IDS,
  fixesPausedOrderIds = EMPTY_FIXES_PAUSED_IDS,
  onDispatch,
  href,
  onOpen,
  confidenceScore,
  isAnalyzing = false,
}: WorkOrderCardProps) {
  const meta = getWorkOrderDisplayStatusMeta(entry.displayStatus);
  const destination = href ?? workOrderOpenPath(organizationId, factoryKey, entry.order.number, factoryLines[0]?.id);
  const startedAt = entry.createdAtMs > 0 ? new Date(entry.createdAtMs) : null;
  const showStart = entry.displayStatus === "draft";
  const attentionReasons = getWorkOrderAttentionReasons(entry.order, {
    addressingFeedback: addressingFeedbackOrderIds.has(entry.id),
    waitingOnChecks: waitingOnChecksOrderIds.has(entry.id),
    checksPassed: checksPassedOrderIds.has(entry.id),
    fixesPaused: fixesPausedOrderIds.has(entry.id),
  });

  return (
    <article
      className="group relative w-full rounded-md border border-border bg-card p-2.5 shadow-sm transition hover:border-foreground/20 hover:shadow"
      data-testid={`work-order-card-${entry.id}`}
    >
      <WorkOrderCardOpenControl onOpen={onOpen} destination={destination} title={entry.title} />

      <div className="relative z-10 pointer-events-none">
        <div className="flex min-w-0 items-center gap-2">
          <WorkOrderStatusDot
            colorClassName={meta.dotClassName}
            pulsing={entry.displayStatus === "running"}
            title={meta.label}
            aria-label={meta.label}
          />
          <h3 className="min-w-0 flex-1 truncate text-[13px] font-medium leading-snug text-foreground">
            {entry.title}
          </h3>
        </div>

        <WorkOrderCardMetaRow
          entry={entry}
          organizationId={organizationId}
          factoryLines={factoryLines}
          preferredLineName={preferredLineName}
          canDispatch={canDispatch}
          isDispatching={dispatchingOrderIds.has(entry.id)}
          onDispatch={onDispatch}
          startedAt={startedAt}
          showStart={showStart}
          confidenceScore={confidenceScore}
          isAnalyzing={isAnalyzing}
          attentionReasons={attentionReasons}
          feedbackLabel={addressingFeedbackLabels.get(entry.id)}
        />
      </div>
    </article>
  );
}

function WorkOrderCardOpenControl({
  onOpen,
  destination,
  title,
}: {
  onOpen?: () => void;
  destination: string;
  title: string;
}) {
  if (onOpen) {
    return (
      <button type="button" className="absolute inset-0 z-0 rounded-md" aria-label={`Open ${title}`} onClick={onOpen} />
    );
  }
  return <Link to={destination} className="absolute inset-0 z-0 rounded-md" aria-label={`Open ${title}`} />;
}

function WorkOrderCardMetaRow({
  entry,
  organizationId,
  factoryLines,
  preferredLineName,
  canDispatch,
  isDispatching,
  onDispatch,
  startedAt,
  showStart,
  confidenceScore,
  isAnalyzing,
  attentionReasons,
  feedbackLabel,
}: {
  entry: WorkOrderListEntry;
  organizationId: string;
  factoryLines: FactoriesFactoryLine[];
  preferredLineName?: string;
  canDispatch: boolean;
  isDispatching: boolean;
  onDispatch: WorkOrderCardContext["onDispatch"];
  startedAt: Date | null;
  showStart: boolean;
  confidenceScore?: number;
  isAnalyzing: boolean;
  attentionReasons: WorkOrderAttentionReason[];
  feedbackLabel?: string;
}) {
  const startedLabel = startedAt ? formatTimeAgo(startedAt) : "—";
  const showActions = confidenceScore != null || isAnalyzing || showStart;

  return (
    <div className="mt-2 flex items-center justify-between gap-2">
      <div className="flex h-5 min-w-0 items-center gap-1.5">
        {showStart ? null : <CardOwnerMark entry={entry} organizationId={organizationId} />}
        <span className="truncate text-[11px] leading-4 text-muted-foreground" title={startedAt?.toLocaleString()}>
          {startedLabel}
        </span>
      </div>
      {showActions ? (
        <div className="flex shrink-0 items-center gap-1.5">
          <CardConfidence entryId={entry.id} score={confidenceScore} isAnalyzing={isAnalyzing} />
          {showStart ? (
            <StartDraftButton
              entry={entry}
              lines={factoryLines}
              preferredLineName={preferredLineName}
              canDispatch={canDispatch}
              isDispatching={isDispatching}
              onDispatch={onDispatch}
            />
          ) : null}
        </div>
      ) : null}
      {attentionReasons.length > 0 ? (
        <div className="flex shrink-0 justify-end gap-1">
          {attentionReasons.map((reason) =>
            reason === "checksPassed" ? (
              <WorkOrderChecksPassedMark key={reason} />
            ) : (
              <WorkOrderAttentionChip
                key={reason}
                reason={reason}
                label={reason === "feedback" ? feedbackLabel : undefined}
              />
            ),
          )}
        </div>
      ) : null}
    </div>
  );
}

/**
 * Score meter, or a spinner while the Backlog automation still analyzes the
 * task. Both take the same slot, so the card does not move when the
 * score arrives.
 */
function CardConfidence({ entryId, score, isAnalyzing }: { entryId: string; score?: number; isAnalyzing: boolean }) {
  if (score != null) {
    return <ConfidenceMeter score={score} className="shrink-0" testId={`work-order-card-score-${entryId}`} />;
  }
  if (isAnalyzing) {
    return <ConfidenceAnalyzingIndicator className="shrink-0" testId={`work-order-card-analyzing-${entryId}`} />;
  }
  return null;
}