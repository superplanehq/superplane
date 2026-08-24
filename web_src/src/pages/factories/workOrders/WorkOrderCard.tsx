import type { FactoriesFactoryLine } from "@/api-client";
import { formatTimeAgo } from "@/lib/date";
import { cn } from "@/lib/utils";
import { Link } from "react-router";
import {
  getWorkOrderAttentionReason,
  WORK_ORDER_ATTENTION_CHIP_CLASSNAME,
  WORK_ORDER_ATTENTION_ICON,
  WORK_ORDER_ATTENTION_LABEL,
} from "../lib/workOrderAttention";
import { factoryHomePath } from "../lib/factoryPagePaths";
import type { WorkOrderListEntry } from "../lib/workOrderListModel";
import { getWorkOrderDisplayStatusMeta } from "../lib/workOrderProgress";
import { ConfidenceMeter } from "./ConfidenceMeter";
import { CardOwnerMark, StartDraftButton, type WorkOrderRowCallbacks } from "./WorkOrderRowActions";

export interface WorkOrderCardContext extends WorkOrderRowCallbacks {
  organizationId: string;
  factoryKey: string;
  factoryLines: FactoriesFactoryLine[];
  /** When set, Start on a draft sends the work order to this line. */
  preferredLineName?: string;
  canDispatch: boolean;
  canAssign: boolean;
  isDispatching: boolean;
  isAssigneesSaving: boolean;
}

export interface WorkOrderCardProps extends WorkOrderCardContext {
  entry: WorkOrderListEntry;
  /**
   * Overlay destination. Defaults to the work order. The Lines board
   * passes onOpen to show the card dialog instead of navigating.
   */
  href?: string;
  /** When set, the card overlay opens this handler instead of navigating. */
  onOpen?: () => void;
  /** First-run analysis score from 0 to 5. Shown to the left of Start. */
  confidenceScore?: number;
}

/**
 * The canonical work order card.
 *
 * Every board uses this complete component. Status is a colored dot next
 * to the title. The footer shows the owner (except on drafts), the age of
 * the work order, and a Start button on drafts. Reviewed drafts also
 * show a score to the left of Start. Waiting cards show an attention
 * label. The owner is
 * display-only on the card.
 */
export function WorkOrderCard({
  entry,
  organizationId,
  factoryKey,
  factoryLines,
  preferredLineName,
  canDispatch,
  isDispatching,
  onDispatch,
  href,
  onOpen,
  confidenceScore,
}: WorkOrderCardProps) {
  const meta = getWorkOrderDisplayStatusMeta(entry.displayStatus);
  const destination = href ?? factoryHomePath(organizationId, factoryKey, factoryLines[0]?.id);
  const startedAt = entry.createdAtMs > 0 ? new Date(entry.createdAtMs) : null;
  const startedLabel = startedAt ? formatTimeAgo(startedAt) : "—";
  const showStart = entry.displayStatus === "draft";
  const attentionReason = getWorkOrderAttentionReason(entry.order);
  const AttentionIcon = attentionReason ? WORK_ORDER_ATTENTION_ICON[attentionReason] : null;

  return (
    <article
      className="group relative w-full rounded-md border border-border bg-card p-2.5 shadow-sm transition hover:border-foreground/20 hover:shadow"
      data-testid={`work-order-card-${entry.id}`}
    >
      {onOpen ? (
        <button
          type="button"
          className="absolute inset-0 z-0 rounded-md"
          aria-label={`Open ${entry.title}`}
          onClick={onOpen}
        />
      ) : (
        <Link to={destination} className="absolute inset-0 z-0 rounded-md" aria-label={`Open ${entry.title}`} />
      )}

      <div className="relative z-10 pointer-events-none">
        <div className="flex min-w-0 items-center gap-2">
          <span className="relative size-2 shrink-0" title={meta.label} aria-label={meta.label}>
            {entry.displayStatus === "running" ? (
              <span
                className={cn("absolute inset-0 rounded-full opacity-60 animate-ping", meta.dotClassName)}
                aria-hidden
              />
            ) : null}
            <span className={cn("relative block size-2 rounded-full", meta.dotClassName)} />
          </span>
          <h3 className="min-w-0 flex-1 truncate text-[13px] font-medium leading-snug text-foreground">
            {entry.title}
          </h3>
        </div>

        <div className="mt-2 flex items-center justify-between gap-2">
          <div className="flex h-5 min-w-0 items-center gap-1.5">
            {showStart ? null : <CardOwnerMark entry={entry} organizationId={organizationId} />}
            <span
              className="truncate text-[11px] leading-none text-muted-foreground"
              title={startedAt?.toLocaleString()}
            >
              {startedLabel}
            </span>
          </div>
          {confidenceScore != null || showStart ? (
            <div className="flex shrink-0 items-center gap-1.5">
              {confidenceScore != null ? (
                <ConfidenceMeter
                  score={confidenceScore}
                  className="shrink-0"
                  testId={`work-order-card-score-${entry.id}`}
                />
              ) : null}
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
          {attentionReason && AttentionIcon ? (
            <span
              className={cn(
                "inline-flex shrink-0 items-center gap-1 rounded-md border px-1.5 py-0.5 text-[10px] font-medium",
                WORK_ORDER_ATTENTION_CHIP_CLASSNAME[attentionReason],
              )}
            >
              <AttentionIcon className="size-3" aria-hidden />
              {WORK_ORDER_ATTENTION_LABEL[attentionReason]}
            </span>
          ) : null}
        </div>
      </div>
    </article>
  );
}
