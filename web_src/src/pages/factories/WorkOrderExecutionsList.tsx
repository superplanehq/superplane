import type { FactoriesWorkOrderQueueItem } from "@/api-client";
import { Link } from "@/components/Link/link";
import { formatTimeAgo } from "@/lib/date";
import { cn } from "@/lib/utils";
import { Check, CircleDashed, Clock, Loader2, MinusCircle, XCircle } from "lucide-react";
import {
  getExecutionStepTimestamp,
  getWorkOrderExecutionDisplayMeta,
  getWorkOrderExecutionRunHref,
  groupWorkOrderExecutionsByLine,
  isQueuedStepRow,
  workOrderStepRows,
  type WorkOrderExecutionLineGroup,
  type WorkOrderStepRow,
} from "./lib/workOrderExecutions";

interface WorkOrderExecutionsListProps {
  organizationId: string;
  factoryKey: string;
  orderNumber?: string;
  executions?: WorkOrderStepRow[];
  queueItems?: FactoriesWorkOrderQueueItem[];
  variant?: "default" | "compact" | "inline";
  emptyMessage?: string;
}

export function WorkOrderExecutionsList({
  organizationId,
  factoryKey,
  orderNumber,
  executions,
  queueItems,
  variant = "default",
  emptyMessage = "No line runs yet. Dispatch this work order to a line to start execution.",
}: WorkOrderExecutionsListProps) {
  const groups = groupWorkOrderExecutionsByLine(workOrderStepRows(executions, queueItems));

  if (groups.length === 0) {
    return variant === "compact" || variant === "inline" ? null : (
      <p className="text-sm text-gray-500 dark:text-gray-400">{emptyMessage}</p>
    );
  }

  if (variant === "inline") {
    return (
      <ul className="space-y-1.5">
        {groups.flatMap((group) =>
          group.executions.map((execution, index) => (
            <CompactExecutionRow
              key={execution.id ?? `${group.lineId}-${execution.step}-${index}`}
              organizationId={organizationId}
              factoryKey={factoryKey}
              orderNumber={orderNumber}
              execution={execution}
            />
          )),
        )}
      </ul>
    );
  }

  if (variant === "compact") {
    return (
      <ul className="mt-4 space-y-4">
        {groups.map((group) => (
          <CompactLineGroup
            key={group.lineId}
            organizationId={organizationId}
            factoryKey={factoryKey}
            orderNumber={orderNumber}
            group={group}
          />
        ))}
      </ul>
    );
  }

  return (
    <ul className="space-y-4">
      {groups.map((group) => (
        <li key={group.lineId} className="overflow-hidden rounded-lg border border-gray-200 dark:border-gray-700/70">
          <div className="border-b border-gray-200 bg-gray-50 px-4 py-2 text-sm font-medium text-gray-900 dark:border-gray-700/70 dark:bg-gray-800/60 dark:text-gray-100">
            {group.lineName}
          </div>
          <ul className="divide-y divide-gray-200 dark:divide-gray-700/70">
            {group.executions.map((execution, index) => (
              <ExecutionRow
                key={execution.id ?? `${group.lineId}-${execution.step}-${index}`}
                organizationId={organizationId}
                factoryKey={factoryKey}
                orderNumber={orderNumber}
                execution={execution}
              />
            ))}
          </ul>
        </li>
      ))}
    </ul>
  );
}

function CompactLineGroup({
  organizationId,
  factoryKey,
  orderNumber,
  group,
}: {
  organizationId: string;
  factoryKey: string;
  orderNumber?: string;
  group: WorkOrderExecutionLineGroup;
}) {
  return (
    <li>
      <p className="text-sm font-medium text-gray-900 dark:text-gray-100">{group.lineName}</p>
      <ul className="mt-2 space-y-1.5">
        {group.executions.map((execution, index) => (
          <CompactExecutionRow
            key={execution.id ?? `${group.lineId}-${execution.step}-${index}`}
            organizationId={organizationId}
            factoryKey={factoryKey}
            orderNumber={orderNumber}
            execution={execution}
          />
        ))}
      </ul>
    </li>
  );
}

function CompactExecutionRow({
  organizationId,
  factoryKey,
  orderNumber,
  execution,
}: {
  organizationId: string;
  factoryKey: string;
  orderNumber?: string;
  execution: WorkOrderStepRow;
}) {
  const meta = getWorkOrderExecutionDisplayMeta(execution);
  const stepLabel = execution.step?.trim() || "Unnamed step";
  const runHref = getWorkOrderExecutionRunHref(organizationId, factoryKey, execution, { orderNumber });

  return (
    <li className="flex min-w-0 items-center gap-2 text-sm text-gray-700 dark:text-gray-300">
      <ExecutionStepMeta execution={execution} stepLabel={stepLabel} meta={meta} runHref={runHref} showTimestamp />
    </li>
  );
}

function ExecutionRow({
  organizationId,
  factoryKey,
  orderNumber,
  execution,
}: {
  organizationId: string;
  factoryKey: string;
  orderNumber?: string;
  execution: WorkOrderStepRow;
}) {
  const meta = getWorkOrderExecutionDisplayMeta(execution);
  const stepLabel = execution.step?.trim() || "Unnamed step";
  const appName = execution.run?.appName?.trim();
  const runHref = getWorkOrderExecutionRunHref(organizationId, factoryKey, execution, { orderNumber });

  return (
    <li className="px-4 py-3">
      <ExecutionStepMeta
        execution={execution}
        stepLabel={stepLabel}
        meta={meta}
        runHref={runHref}
        size="md"
        stepClassName="text-gray-900 dark:text-gray-100"
      />
      {appName ? <p className="mt-1 pl-6 text-xs text-gray-500 dark:text-gray-400">{appName}</p> : null}
    </li>
  );
}

function ExecutionStepMeta({
  execution,
  stepLabel,
  meta,
  runHref,
  showTimestamp = false,
  size = "sm",
  stepClassName = "text-gray-700 dark:text-gray-300",
}: {
  execution: WorkOrderStepRow;
  stepLabel: string;
  meta: ReturnType<typeof getWorkOrderExecutionDisplayMeta>;
  runHref?: string | null;
  showTimestamp?: boolean;
  size?: "sm" | "md";
  stepClassName?: string;
}) {
  const stepAt = showTimestamp ? getExecutionStepTimestamp(execution) : "";
  const queuePosition = execution.queuePosition ?? 0;
  const stepLabelContent = (
    <span className={cn("min-w-0 font-medium", !showTimestamp && "truncate")}>
      {stepLabel}
      {queuePosition > 0 ? (
        <>
          {" "}
          <span className="text-xs font-normal text-sky-600 dark:text-sky-300">#{queuePosition} in queue</span>
        </>
      ) : null}
      {stepAt ? (
        <>
          {" "}
          <time className="text-xs font-normal text-gray-500 dark:text-gray-400">
            {formatTimeAgo(new Date(stepAt))}
          </time>
        </>
      ) : null}
    </span>
  );
  const linkContent = (
    <>
      <ExecutionStatusIcon execution={execution} meta={meta} size={size} />
      {stepLabelContent}
    </>
  );
  const linkClassName = cn(
    "pointer-events-auto inline-flex w-fit max-w-full items-center gap-2 rounded-md px-1 py-0.5 text-sm no-underline transition-colors",
    "text-gray-700 hover:bg-gray-50 hover:text-gray-900 dark:text-gray-300 dark:hover:bg-gray-800/40 dark:hover:text-gray-100",
    stepClassName,
  );
  const staticClassName = cn(
    "inline-flex w-fit max-w-full items-center gap-2 text-sm",
    stepClassName || "text-gray-700 dark:text-gray-300",
  );

  if (runHref) {
    return (
      <Link href={runHref} className={linkClassName}>
        {linkContent}
      </Link>
    );
  }

  return <div className={staticClassName}>{linkContent}</div>;
}

function ExecutionStatusIcon({
  execution,
  meta,
  size = "sm",
}: {
  execution: WorkOrderStepRow;
  meta: ReturnType<typeof getWorkOrderExecutionDisplayMeta>;
  size?: "sm" | "md";
}) {
  const iconClassName = cn("shrink-0", size === "md" ? "h-4 w-4" : "h-3.5 w-3.5");

  if (execution.result === "RESULT_PASSED") {
    return <Check className={cn(iconClassName, "text-emerald-500")} aria-label="Passed" />;
  }

  if (execution.result === "RESULT_FAILED") {
    return <XCircle className={cn(iconClassName, "text-red-500")} aria-label="Failed" />;
  }

  if (isQueuedStepRow(execution)) {
    return <Clock className={cn(iconClassName, "text-sky-500")} aria-label="Queued" />;
  }

  if (meta.isActive) {
    return <Loader2 className={cn(iconClassName, "animate-spin text-violet-500")} aria-label={meta.label} />;
  }

  if (execution.result === "RESULT_CANCELLED") {
    return <MinusCircle className={cn(iconClassName, "text-gray-400")} aria-label="Cancelled" />;
  }

  if (execution.state === "STATE_PENDING") {
    return <CircleDashed className={cn(iconClassName, "text-amber-500")} aria-label="Pending" />;
  }

  return <CircleDashed className={cn(iconClassName, "text-gray-400")} aria-label={meta.label} />;
}
