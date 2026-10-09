import type { FactoriesFactoryLine, FactoriesFactoryPullRequest } from "@/api-client";
import { formatRelative } from "@/lib/datetime";
import { cn } from "@/lib/utils";
import { Bot } from "lucide-react";
import { Link } from "react-router";
import {
  getWorkOrderAttentionReasons,
  getWorkOrderFailedAttentionLabel,
  type WorkOrderAttentionReason,
} from "../lib/workOrderAttention";
import {
  selectWorkOrderCardPullRequest,
  visibleWorkOrderCardAttentionReasons,
  workOrderCardPullRequestIsMergeable,
} from "../lib/workOrderCardPullRequest";
import { workOrderCardSource } from "../lib/workOrderCardSource";
import { workOrderOpenPath } from "../lib/factoryPagePaths";
import type { WorkOrderListEntry } from "../lib/workOrderListModel";
import { getWorkOrderDisplayStatusMeta } from "../lib/workOrderProgress";
import type { MergeConfidenceCardCheck } from "../lib/mergeConfidenceScore";
import { ConfidenceAnalyzingIndicator } from "./ConfidenceMeter";
import { CardScoreBadges, MergeConfidenceChip } from "./ReadinessMark";
import { WorkOrderAttentionChip } from "./WorkOrderAttentionChip";
import { WorkOrderPullRequestChip, WorkOrderMergeableChip } from "./WorkOrderPullRequestChip";
import { CardOwnerMark, type WorkOrderRowCallbacks } from "./WorkOrderRowActions";
import { WorkOrderSourceIcon } from "./WorkOrderSourceIcon";
import { WorkOrderStatusIcon } from "./WorkOrderStatusIcon";
import { WORK_ORDER_CARD_HOVER_SURFACE_CLASS } from "./workOrderCardSurface";
import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { FEATURE_FACTORY_PULL_REQUEST_MERGE } from "@/lib/experimentalFeatures";

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
  /** Clarity score from the task, 0 to 5. Shown in the footer. */
  clarityScore?: number;
  /** Confidence score from the task, 0 to 5. Shown in the footer. */
  confidenceScore?: number;
  /** Hide Clarity when Planning has that score off. */
  showClarity?: boolean;
  /** Hide Confidence when Planning has that score off. */
  showConfidenceScore?: boolean;
  /**
   * Merge confidence headline from Verify. Independent of Clarity and
   * Confidence. Absent until at least one merge confidence check exists.
   */
  mergeConfidence?: { score: number; maxScore: number; checks?: MergeConfidenceCardCheck[] };
  /** Review sub-parameters for the Confidence tooltip when the headline is derived. */
  reviewMetrics?: { key: string; name: string; score: number }[];
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
  /**
   * Short credit-failure label for a draft whose analysis never started.
   * Line steps use the failed attention chip instead.
   */
  creditLabel?: string;
  /**
   * False on the public board. The card is static text: no link, no dialog.
   */
  interactive?: boolean;
  /** @deprecated Owner is always shown on cards. Kept for call-site compatibility. */
  showOwner?: boolean;
}

/**
 * The canonical task card.
 *
 * Every board uses this complete component. Status is an icon next
 * to the title; the owner avatar sits on the right of that row. Optional
 * pills sit on a middle row: an attached pull request, then
 * attention such as Waiting on status checks. The footer shows the
 * intake source on the left, then when the task was last updated. A task
 * with no owner shows a dashed person icon on the title row. Reviewed
 * drafts show Clarity and Confidence scores on the right of the footer.
 * After Verify writes a merge confidence check, three bars sit with
 * those scores.
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
  showClarity,
  showConfidenceScore,
  mergeConfidence,
  reviewMetrics,
  isAnalyzing = false,
  className,
  selected = false,
  hasAgentQuestion = false,
  creditLabel,
  interactive = true,
  canAssign,
  isAssigneesSaving,
  onAssigneesSave,
}: WorkOrderCardProps) {
  const meta = getWorkOrderDisplayStatusMeta(entry.displayStatus);
  const destination = interactive
    ? (href ?? workOrderOpenPath(organizationId, factoryKey, entry.order.number, factoryLines[0]?.id))
    : "";
  const updatedAt = entry.updatedAtMs > 0 ? new Date(entry.updatedAtMs) : null;
  const isDraft = entry.displayStatus === "draft";
  const { showAgentQuestion, agentWorking } = draftCardActionFlags(isDraft, isAnalyzing, hasAgentQuestion);
  const cardPullRequest = selectWorkOrderCardPullRequest(pullRequests, entry.id);
  const source = workOrderCardSource(entry.order);
  const pullRequestMerge = useExperimentalFeature(organizationId ? organizationId : null);
  const showPullRequestMerge = pullRequestMerge.has(FEATURE_FACTORY_PULL_REQUEST_MERGE);
  const attentionReasons = visibleWorkOrderCardAttentionReasons(
    getWorkOrderAttentionReasons(entry.order, {
      addressingFeedback: addressingFeedbackOrderIds.has(entry.id),
      waitingOnChecks: waitingOnChecksOrderIds.has(entry.id),
      checksPassed: checksPassedOrderIds.has(entry.id),
      fixesPaused: fixesPausedOrderIds.has(entry.id),
    }),
    cardPullRequest,
    showPullRequestMerge,
  );
  return (
    <article
      className={cn(
        "group relative w-full rounded-md border border-border bg-card p-2.5 shadow-sm",
        interactive && "transition hover:border-foreground/20 hover:shadow",
        interactive && WORK_ORDER_CARD_HOVER_SURFACE_CLASS,
        className,
      )}
      data-testid={`work-order-card-${entry.id}`}
      data-selected={selected || undefined}
    >
      {interactive ? <WorkOrderCardOpenControl onOpen={onOpen} destination={destination} title={entry.title} /> : null}

      <div className="relative z-10 pointer-events-none">
        <WorkOrderCardTitleRow
          entryId={entry.id}
          displayStatus={entry.displayStatus}
          statusLabel={meta.label}
          title={entry.title}
          showOwnerCorner
          organizationId={organizationId}
          entry={entry}
          canAssign={canAssign}
          isAssigneesSaving={isAssigneesSaving}
          onAssigneesSave={onAssigneesSave}
        />

        <WorkOrderCardStatusRow
          entryId={entry.id}
          reasons={attentionReasons}
          feedbackLabel={addressingFeedbackLabels.get(entry.id)}
          checksPassedLabel={checksPassedLabels.get(entry.id)}
          failedLabel={getWorkOrderFailedAttentionLabel(entry.order)}
          creditLabel={isDraft ? creditLabel : undefined}
          cardPullRequest={cardPullRequest}
          hasAgentQuestion={showAgentQuestion}
          showPullRequestMerge={showPullRequestMerge}
        />
        <WorkOrderCardMetaRow
          entryId={entry.id}
          source={source}
          updatedAt={updatedAt}
          clarityScore={clarityScore}
          confidenceScore={confidenceScore}
          showClarity={showClarity}
          showConfidenceScore={showConfidenceScore}
          mergeConfidence={mergeConfidence}
          reviewMetrics={reviewMetrics}
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
  showOwnerCorner,
  organizationId,
  entry,
  canAssign,
  isAssigneesSaving,
  onAssigneesSave,
}: {
  entryId: string;
  displayStatus: WorkOrderListEntry["displayStatus"];
  statusLabel: string;
  title: string;
  showOwnerCorner: boolean;
  organizationId: string;
  entry: WorkOrderListEntry;
  canAssign: boolean;
  isAssigneesSaving: boolean;
  onAssigneesSave: (orderId: string, assigneeIds: string[]) => Promise<void>;
}) {
  return (
    <div className="flex min-w-0 items-center gap-2">
      <WorkOrderStatusIcon status={displayStatus} title={statusLabel} aria-label={statusLabel} />
      <h3 className="min-w-0 flex-1 truncate text-[13px] font-medium leading-5 text-foreground">{title}</h3>
      {showOwnerCorner ? (
        <div className="flex shrink-0 items-center gap-0.5" data-testid={`work-order-card-title-trailing-${entryId}`}>
          <CardOwnerMark
            entry={entry}
            organizationId={organizationId}
            canAssign={canAssign}
            isAssigneesSaving={isAssigneesSaving}
            onAssigneesSave={onAssigneesSave}
          />
        </div>
      ) : null}
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
  failedLabel,
  creditLabel,
  cardPullRequest,
  hasAgentQuestion,
  showPullRequestMerge,
}: {
  entryId: string;
  reasons: WorkOrderAttentionReason[];
  feedbackLabel?: string;
  checksPassedLabel?: string;
  failedLabel?: string;
  creditLabel?: string;
  cardPullRequest: ReturnType<typeof selectWorkOrderCardPullRequest>;
  hasAgentQuestion: boolean;
  showPullRequestMerge?: boolean;
}) {
  const showCredit = Boolean(creditLabel) && !reasons.includes("failed");
  if (reasons.length === 0 && !cardPullRequest && !hasAgentQuestion && !showCredit) {
    return null;
  }

  return (
    <div className="mt-1.5 flex min-w-0 flex-wrap items-center gap-1">
      {hasAgentQuestion ? <WorkOrderAgentQuestionChip entryId={entryId} /> : null}
      {cardPullRequest ? (
        <>
          <WorkOrderPullRequestChip pullRequest={cardPullRequest.pullRequest} extraCount={cardPullRequest.extraCount} />
          {showPullRequestMerge && workOrderCardPullRequestIsMergeable(cardPullRequest.pullRequest) ? (
            <WorkOrderMergeableChip />
          ) : null}
        </>
      ) : null}
      {showCredit ? <WorkOrderAttentionChip reason="failed" label={creditLabel} /> : null}
      {reasons.map((reason) => (
        <WorkOrderAttentionChip
          key={reason}
          reason={reason}
          label={attentionChipLabel(reason, { feedbackLabel, checksPassedLabel, failedLabel })}
        />
      ))}
    </div>
  );
}

function attentionChipLabel(
  reason: WorkOrderAttentionReason,
  labels: { feedbackLabel?: string; checksPassedLabel?: string; failedLabel?: string },
): string | undefined {
  if (reason === "feedback") {
    return labels.feedbackLabel;
  }
  if (reason === "checksPassed") {
    return labels.checksPassedLabel;
  }
  if (reason === "failed") {
    return labels.failedLabel;
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
  entryId,
  source,
  updatedAt,
  clarityScore,
  confidenceScore,
  showClarity = true,
  showConfidenceScore = true,
  mergeConfidence,
  reviewMetrics,
  isAnalyzing,
}: {
  entryId: string;
  source: ReturnType<typeof workOrderCardSource>;
  updatedAt: Date | null;
  clarityScore?: number;
  confidenceScore?: number;
  showClarity?: boolean;
  showConfidenceScore?: boolean;
  mergeConfidence?: { score: number; maxScore: number; checks?: MergeConfidenceCardCheck[] };
  reviewMetrics?: { key: string; name: string; score: number }[];
  isAnalyzing: boolean;
}) {
  const updatedLabel = updatedAt ? formatRelative(updatedAt) : "—";
  const hasScore = (showClarity && clarityScore != null) || (showConfidenceScore && confidenceScore != null);
  const showActions = hasScore || isAnalyzing;
  const showTrailing = showActions || mergeConfidence != null;

  return (
    <div className="mt-2 flex items-center justify-between gap-2">
      <div className="flex min-w-0 flex-1 items-center gap-2" data-testid={`work-order-card-footer-leading-${entryId}`}>
        {source ? <WorkOrderSourceIcon entryId={entryId} source={source} /> : null}
        <span
          className="truncate text-[11px] font-medium leading-[0.875rem] text-muted-foreground"
          title={updatedAt ? `Updated ${updatedAt.toLocaleString()}` : undefined}
        >
          {updatedLabel}
        </span>
      </div>
      {showTrailing ? (
        <div className="flex shrink-0 items-center gap-1.5">
          {mergeConfidence ? (
            <MergeConfidenceChip
              score={mergeConfidence.score}
              maxScore={mergeConfidence.maxScore}
              checks={mergeConfidence.checks}
              testId={`work-order-card-merge-${entryId}`}
            />
          ) : null}
          {showActions ? (
            <CardScores
              entryId={entryId}
              clarity={clarityScore}
              confidence={confidenceScore}
              showClarity={showClarity}
              showConfidence={showConfidenceScore}
              reviewMetrics={reviewMetrics}
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
  showClarity = true,
  showConfidence = true,
  reviewMetrics,
  isAnalyzing,
}: {
  entryId: string;
  clarity?: number;
  confidence?: number;
  showClarity?: boolean;
  showConfidence?: boolean;
  reviewMetrics?: { key: string; name: string; score: number }[];
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
  return (
    <CardScoreBadges
      clarity={clarity}
      confidence={confidence}
      showClarity={showClarity}
      showConfidence={showConfidence}
      reviewMetrics={reviewMetrics}
      testId={`work-order-card-score-${entryId}`}
    />
  );
}
