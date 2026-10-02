import type { Finding } from "./types";
import { EnforcementBadge, SeverityBadge } from "./SeverityBadge";
import { formatFindingLocation } from "./findingLocation";

/** One finding: severity, rule, location, what is wrong, and how to fix it. */
export function FindingItem({ finding }: { finding: Finding }) {
  return (
    <div className="flex flex-col gap-1.5 py-3">
      <div className="flex flex-wrap items-center gap-2">
        <SeverityBadge severity={finding.severity} />
        <span className="text-[13px] font-medium text-foreground">{finding.ruleName}</span>
        <EnforcementBadge enforcement={finding.enforcement} />
      </div>
      {finding.location ? (
        <code className="w-fit rounded bg-slate-100 px-1.5 py-0.5 font-mono text-[12px] text-slate-700 dark:bg-gray-900 dark:text-gray-300">
          {formatFindingLocation(finding.location)}
        </code>
      ) : null}
      <p className="workspace-body-text text-foreground">{finding.description}</p>
      <p className="text-[13px] text-muted-foreground">
        <span className="font-medium text-foreground">Fix: </span>
        {finding.remediation}
      </p>
    </div>
  );
}
