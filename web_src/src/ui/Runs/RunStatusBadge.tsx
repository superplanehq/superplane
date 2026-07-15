import { RUN_STATUS_META } from "./runPresentation";
import { runStatusBadgeClassName } from "./runStatusBadgeClassNames";

export function RunStatusBadge({ status }: { status: keyof typeof RUN_STATUS_META }) {
  const statusMeta = RUN_STATUS_META[status];
  const StatusIcon = statusMeta.icon;

  return (
    <span aria-label={statusMeta.label} title={statusMeta.label} className={runStatusBadgeClassName(status)}>
      <StatusIcon className="size-3.5" aria-hidden />
      <span>{statusMeta.label}</span>
    </span>
  );
}
