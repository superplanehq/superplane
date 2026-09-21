import type { FactoriesFactoryLine, FactoriesFactoryPullRequest } from "@/api-client";
import { formatRelative } from "@/lib/datetime";
import { cn } from "@/lib/utils";
import { Bot } from "lucide-react";
import { Link } from "react-router";
import { getWorkOrderAttentionReasons, type WorkOrderAttentionReason } from "../lib/workOrderAttention";
import {
  selectWorkOrderCardPullRequest,
  visibleWorkOrderCardAttentionReasons,
  workOrderCardPullRequestIsMergeable,
} from "../lib/workOrderCardPullRequest";
import { workOrderCardSource } from "../lib/workOrderCardSource";
import { workOrderOpenPath } from "../lib/factoryPagePaths";
import type { WorkOrderListEntry } from "../lib/workOrderListModel";
import { getWorkOrderDisplayStatusMeta } from "../lib/workOrderProgress";
import { ConfidenceAnalyzingIndicator } from "./ConfidenceMeter";
import { CardScoreBadges } from "./ReadinessMark";
import { WorkOrderAttentionChip } from "./WorkOrderAttentionChip";
import { WorkOrderPullRequestChip, WorkOrderMergeableChip } from "./WorkOrderPullRequestChip";
import { CardOwnerMark, type WorkOrderRowCallbacks } from "./WorkOrderRowActions";
import { WorkOrderSourceIcon } from "./WorkOrderSourceIcon";
import { WorkOrderStatusIcon } from "./WorkOrderStatusIcon";
import { WORK_ORDER_CARD_HOVER_SURFACE_CLASS } from "./workOrderCardSurface";

const EMPTY_ADDRESSING_FEEDBACK_IDS: ReadonlySet<string> = new Set();
const EMPTY_ADDRESSING_FEEDBACK_LABELS: ReadonlyMap<string, string> = new Map();
const EMPTY_WAITING_ON_CHECKS_IDS: ReadonlySet<string> = new Set();
const EMPTY_CHECKS_PASSED_IDS: ReadonlySet<string> = new Set();
const EMPTY_CHECKS_PASSED_LABELS: ReadonlyMap<string, string> = new Map();
const EMPTY_FIXES_PAUSED_IDS: ReadonlySet<string> = new Set();
const EMPTY_PULL_REQUESTS: FactoriesFactoryPullRequest[] = [];

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
  /** Completed check-wait title, keyed by task id. */
  checksPassedLabels?: ReadonlyMap<string, string>;
  /** Tasks whose check handler stopped at the attempt limit. */
  fixesPausedOrderIds?: ReadonlySet<string>;
  /** Pull requests attached to tasks on this board. */
  pullRequests?: FactoriesFactoryPullRequest[];
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
  /** Clarity score from ListWorkOrderChecks, 0 to 5. Shown in the footer. */
  clarityScore?: number;
  /** Confidence score from ListWorkOrderChecks, 0 to 5. Shown in the footer. */
  confidenceScore?: number;
  /**
   * True while the agent still works on this draft. The card shows
   * thinking states in the meter slot, even after a score exists.
   */
  isAnalyzing?: boolean;
  /** Extra surface classes. Use for sidebar hover and selected fills. */
  className?: string;
  /** True when this card is the active item in a list. */
  selected?: boolean;
  /** True when the draft analysis session waits for a multiple-choice answer. */
  hasAgentQuestion?: boolean;
}

/**
 * The canonical task card.
 *
 * Every board uses this complete component. Status is an icon next
 * to the title, with the intake source icon on the right. Optional
 * pills sit on a middle row: an attached pull request, then
 * attention such as Waiting on status checks. The
 * footer shows when the task was created on the left, and the owner
 * given name plus avatar on the right (except on drafts). Reviewed
 * drafts show Clarity and Confidence scores. The owner is display-only
 * on the card.
 */
export function WorkOrderCard({
  entry,
  organizationId,
  factoryKey,
  factoryLines,
  addressingFeedbackOrderIds = EMPTY_ADDRESSING_FEEDBACK_IDS,
  addressingFeedbackLabels = EMPTY_ADDRESSING_FEEDBACK_LABELS,
  waitingOnChecksOrderIds = EMPTY_WAITING_ON_CHECKS_IDS,
  checksPassedOrderIds = EMPTY_CHECKS_PASSED_IDS,
  checksPassedLabels = EMPTY_CHECKS_PASSED_LABELS,
  fixesPausedOrderIds = EMPTY_FIXES_PAUSED_IDS,
  pullRequests = EMPTY_PULL_REQUESTS,
  href,
  onOpen,
  clarityScore,
  confidenceScore,
  isAnalyzing = false,
  className,
  selected = false,
  hasAgentQuestion = false,
}: WorkOrderCardProps) {
  const meta = getWorkOrderDisplayStatusMeta(entry.displayStatus);
  const destination = href ?? workOrderOpenPath(organizationId, factoryKey, entry.order.number, factoryLines[0]?.id);
  const createdAt = entry.createdAtMs > 0 ? new Date(entry.createdAtMs) : null;
  const isDraft = entry.displayStatus === "draft";
  const { showAgentQuestion, agentWorking } = draftCardActionFlags(isDraft, isAnalyzing, hasAgentQuestion);
  const cardPullRequest = selectWorkOrderCardPullRequest(pullRequests, entry.id);
  const source = workOrderCardSource(entry.order);
  const attentionReasons = visibleWorkOrderCardAttentionReasons(
    getWorkOrderAttentionReasons(entry.order, {
      addressingFeedback: addressingFeedbackOrderIds.has(entry.id),
      waitingOnChecks: waitingOnChecksOrderIds.has(entry.id),
      checksPassed: checksPassedOrderIds.has(entry.id),
      fixesPaused: fixesPausedOrderIds.has(entry.id),
    }),
    cardPullRequest,
  );

  return (
    <article
      className={cn(
        "group relative w-full rounded-md border border-border bg-card p-2.5 shadow-sm transition hover:border-foreground/20 hover:shadow",
        WORK_ORDER_CARD_HOVER_SURFACE_CLASS,
        className,
      )}
      data-testid={`work-order-card-${entry.id}`}
      data-selected={selected || undefined}
    >
      <WorkOrderCardOpenControl onOpen={onOpen} destination={destination} title={entry.title} />

      <div className="relative z-10 pointer-events-none">
        <WorkOrderCardTitleRow
          entryId={entry.id}
          displayStatus={entry.displayStatus}
          statusLabel={meta.label}
          title={entry.title}
          source={source}
        />

        <WorkOrderCardStatusRow
          entryId={entry.id}
          reasons={attentionReasons}
          feedbackLabel={addressingFeedbackLabels.get(entry.id)}
          checksPassedLabel={checksPassedLabels.get(entry.id)}
          cardPullRequest={cardPullRequest}
          hasAgentQuestion={showAgentQuestion}
        />
        <WorkOrderCardMetaRow
          entry={entry}
          organizationId={organizationId}
          createdAt={createdAt}
          isDraft={isDraft}
          clarityScore={clarityScore}
          confidenceScore={confidenceScore}
          isAnalyzing={agentWorking}
        />
      </div>
    </article>
  );
}

function WorkOrderCardTitleRow({
  entryId,
  displayStatus,
  statusLabel,
  title,
  source,
}: {
  entryId: string;
  displayStatus: WorkOrderListEntry["displayStatus"];
  statusLabel: string;
  title: string;
  source: ReturnType<typeof workOrderCardSource>;
}) {
  return (
    <div className="flex min-w-0 items-center gap-2">
      <WorkOrderStatusIcon status={displayStatus} title={statusLabel} aria-label={statusLabel} />
      <h3 className="min-w-0 flex-1 truncate text-[13px] font-medium leading-snug text-foreground">{title}</h3>
      {source ? <WorkOrderSourceIcon entryId={entryId} source={source} /> : null}
    </div>
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

function WorkOrderCardStatusRow({
  entryId,
  reasons,
  feedbackLabel,
  checksPassedLabel,
  cardPullRequest,
  hasAgentQuestion,
}: {
  entryId: string;
  reasons: WorkOrderAttentionReason[];
  feedbackLabel?: string;
  checksPassedLabel?: string;
  cardPullRequest: ReturnType<typeof selectWorkOrderCardPullRequest>;
  hasAgentQuestion: boolean;
}) {
  if (reasons.length === 0 && !cardPullRequest && !hasAgentQuestion) {
    return null;
  }

  return (
    <div className="mt-1.5 flex min-w-0 flex-wrap items-center gap-1">
      {hasAgentQuestion ? <WorkOrderAgentQuestionChip entryId={entryId} /> : null}
      {cardPullRequest ? (
        <>
          <WorkOrderPullRequestChip pullRequest={cardPullRequest.pullRequest} extraCount={cardPullRequest.extraCount} />
          {workOrderCardPullRequestIsMergeable(cardPullRequest.pullRequest) ? <WorkOrderMergeableChip /> : null}
        </>
      ) : null}
      {reasons.map((reason) => (
        <WorkOrderAttentionChip
          key={reason}
          reason={reason}
          label={attentionChipLabel(reason, feedbackLabel, checksPassedLabel)}
        />
      ))}
    </div>
  );
}

function attentionChipLabel(
  reason: WorkOrderAttentionReason,
  feedbackLabel?: string,
  checksPassedLabel?: string,
): string | undefined {
  if (reason === "feedback") {
    return feedbackLabel;
  }
  if (reason === "checksPassed") {
    return checksPassedLabel;
  }
  return undefined;
}

function WorkOrderAgentQuestionChip({ entryId }: { entryId: string }) {
  return (
    <span
      className="inline-flex max-w-full shrink-0 items-center gap-1 rounded-full border border-blue-500/30 bg-blue-500/10 px-2 py-0.5 text-[10px] font-medium text-blue-700 dark:text-blue-400"
      data-testid={`work-order-card-agent-question-${entryId}`}
      title="Agent question"
    >
      <Bot className="size-3 shrink-0" aria-hidden />
      <span className="truncate">Agent question</span>
    </span>
  );
}

function WorkOrderCardMetaRow({
  entry,
  organizationId,
  createdAt,
  isDraft,
  clarityScore,
  confidenceScore,
  isAnalyzing,
}: {
  entry: WorkOrderListEntry;
  organizationId: string;
  createdAt: Date | null;
  isDraft: boolean;
  clarityScore?: number;
  confidenceScore?: number;
  isAnalyzing: boolean;
}) {
  const createdLabel = createdAt ? formatRelative(createdAt) : "—";
  const hasScore = clarityScore != null || confidenceScore != null;
  const showActions = hasScore || isAnalyzing;
  const ownerMark = isDraft ? null : <CardOwnerMark entry={entry} organizationId={organizationId} />;

  return (
    <div className="mt-2 flex items-center justify-between gap-2">
      <span
        className="truncate text-[11px] leading-4 text-muted-foreground"
        title={createdAt ? `Created ${createdAt.toLocaleString()}` : undefined}
      >
        {createdLabel}
      </span>
      {ownerMark || showActions ? (
        <div className="ml-auto flex h-5 min-w-0 items-center gap-1.5">
          {ownerMark}
          {showActions ? (
            <CardScores
              entryId={entry.id}
              clarity={clarityScore}
              confidence={confidenceScore}
              isAnalyzing={isAnalyzing}
            />
          ) : null}
        </div>
      ) : null}
    </div>
  );
}

function draftCardActionFlags(isDraft: boolean, isAnalyzing: boolean, hasAgentQuestion: boolean) {
  const showAgentQuestion = hasAgentQuestion && isDraft;
  const agentWorking = isAnalyzing && !showAgentQuestion;
  return { showAgentQuestion, agentWorking };
}

/**
 * Thinking states while the agent still works, even after a score exists.
 * When the agent waits for the user the card shows one verdict word with a
 * tone dot. The two scores stay in the tooltip. Intake-only drafts have
 * Confidence alone; the verdict uses the scores that exist.
 */
function CardScores({
  entryId,
  clarity,
  confidence,
  isAnalyzing,
}: {
  entryId: string;
  clarity?: number;
  confidence?: number;
  isAnalyzing: boolean;
}) {
  if (isAnalyzing) {
    return (
      <ConfidenceAnalyzingIndicator
        className="shrink-0"
        testId={`work-order-card-analyzing-${entryId}`}
        showThinkingStates
        showTooltip={false}
      />
    );
  }
  if (clarity == null && confidence == null) {
    return null;
  }
  return <CardScoreBadges clarity={clarity} confidence={confidence} testId={`work-order-card-score-${entryId}`} />;
}
