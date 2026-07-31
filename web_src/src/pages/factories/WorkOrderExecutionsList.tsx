import type { FactoriesWorkOrderExecution } from "@/api-client";
import { Link } from "@/components/Link/link";
import { appRunPath } from "@/lib/appPaths";
import { cn } from "@/lib/utils";
import {
  Check,
  CircleDashed,
  ExternalLink,
  Loader2,
  MinusCircle,
  XCircle,
} from "lucide-react";
import {
  getWorkOrderExecutionDisplayMeta,
  groupWorkOrderExecutionsByLine,
  type WorkOrderExecutionLineGroup,
} from "./workOrderExecutions";

interface WorkOrderExecutionsListProps {
  organizationId: string;
  executions?: FactoriesWorkOrderExecution[];
  variant?: "default" | "compact";
  emptyMessage?: string;
}

export function WorkOrderExecutionsList({
  organizationId,
  executions,
  variant = "default",
  emptyMessage = "No line runs yet. Dispatch this work order to a line to start execution.",
}: WorkOrderExecutionsListProps) {
  const groups = groupWorkOrderExecutionsByLine(executions);

  if (groups.length === 0) {
    return variant === "compact" ? null : <p className="text-sm text-gray-500 dark:text-gray-400">{emptyMessage}</p>;
  }

  if (variant === "compact") {
    return (
      <ul className="mt-4 space-y-4">
        {groups.map((group) => (
          <CompactLineGroup key={group.lineId} organizationId={organizationId} group={group} />
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
                execution={execution}
              />
            ))}
          </ul>
        </li>
      ))}
    </ul>
  );
}

function CompactLineGroup({ organizationId, group }: { organizationId: string; group: WorkOrderExecutionLineGroup }) {
  return (
    <li>
      <p className="text-sm font-medium text-gray-900 dark:text-gray-100">{group.lineName}</p>
      <ul className="mt-2 space-y-1.5">
        {group.executions.map((execution, index) => (
          <CompactExecutionRow
            key={execution.id ?? `${group.lineId}-${execution.step}-${index}`}
            organizationId={organizationId}
            execution={execution}
          />
        ))}
      </ul>
    </li>
  );
}

function CompactExecutionRow({
  organizationId,
  execution,
}: {
  organizationId: string;
  execution: FactoriesWorkOrderExecution;
}) {
  const meta = getWorkOrderExecutionDisplayMeta(execution);
  const stepLabel = execution.step?.trim() || "Unnamed step";
  const runHref =
    execution.run?.appId && execution.run.id ? appRunPath(organizationId, execution.run.appId, execution.run.id) : null;

  return (
    <li className="text-sm">
      <ExecutionStepMeta execution={execution} stepLabel={stepLabel} meta={meta} runHref={runHref} />
    </li>
  );
}

function ExecutionRow({
  organizationId,
  execution,
}: {
  organizationId: string;
  execution: FactoriesWorkOrderExecution;
}) {
  const meta = getWorkOrderExecutionDisplayMeta(execution);
  const stepLabel = execution.step?.trim() || "Unnamed step";
  const appName = execution.run?.appName?.trim();
  const runHref =
    execution.run?.appId && execution.run.id ? appRunPath(organizationId, execution.run.appId, execution.run.id) : null;

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
  size = "sm",
  stepClassName = "text-gray-700 dark:text-gray-300",
}: {
  execution: FactoriesWorkOrderExecution;
  stepLabel: string;
  meta: ReturnType<typeof getWorkOrderExecutionDisplayMeta>;
  runHref: string | null;
  size?: "sm" | "md";
  stepClassName?: string;
}) {
  return (
    <div className="flex min-w-0 items-center gap-2">
      <ExecutionStatusIcon execution={execution} meta={meta} size={size} />
      <span className={cn("min-w-0 truncate font-medium", stepClassName)}>{stepLabel}</span>
      {runHref ? <ViewRunLink href={runHref} /> : null}
    </div>
  );
}

function ViewRunLink({ href }: { href: string }) {
  return (
    <Link
      href={href}
      className="inline-flex shrink-0 items-center gap-1 text-xs leading-none font-medium text-gray-600 no-underline hover:text-gray-900 dark:text-gray-400 dark:hover:text-gray-100"
    >
      View run
      <ExternalLink className="h-3 w-3" aria-hidden />
    </Link>
  );
}

function ExecutionStatusIcon({
  execution,
  meta,
  size = "sm",
}: {
  execution: FactoriesWorkOrderExecution;
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

  if (meta.isActive) {
    return (
      <Loader2
        className={cn(iconClassName, "animate-spin text-violet-500")}
        aria-label={meta.label}
      />
    );
  }

  if (execution.result === "RESULT_CANCELLED") {
    return <MinusCircle className={cn(iconClassName, "text-gray-400")} aria-label="Cancelled" />;
  }

  if (execution.state === "STATE_PENDING") {
    return <CircleDashed className={cn(iconClassName, "text-amber-500")} aria-label="Pending" />;
  }

  return <CircleDashed className={cn(iconClassName, "text-gray-400")} aria-label={meta.label} />;
}
