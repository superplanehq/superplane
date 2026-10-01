import { Switch } from "@/components/ui/switch";

import type { MergeConfidenceCheck } from "./mergeConfidenceChecks";
import { MERGE_CONFIDENCE_CHECK_COPY } from "./mergeConfidenceCopy";

export function MergeConfidenceCheckList({
  checks,
  onToggle,
  idPrefix,
}: {
  checks: readonly MergeConfidenceCheck[];
  onToggle: (check: MergeConfidenceCheck, enabled: boolean) => void;
  idPrefix: string;
}) {
  return (
    <div className="flex flex-col divide-y divide-border rounded-lg border border-border bg-card px-3">
      {MERGE_CONFIDENCE_CHECK_COPY.map((check) => (
        <div
          key={check.id}
          className="flex items-start justify-between gap-6 py-3"
          data-testid={`${idPrefix}-${check.id}`}
        >
          <div className="min-w-0 space-y-0.5">
            <p className="text-[13px] font-medium text-foreground">{check.label}</p>
            <p className="text-[12px] leading-5 text-muted-foreground">{check.helper}</p>
          </div>
          <Switch
            checked={checks.includes(check.id)}
            onCheckedChange={(enabled) => onToggle(check.id, enabled)}
            aria-label={check.label}
            className="mt-0.5"
          />
        </div>
      ))}
    </div>
  );
}
