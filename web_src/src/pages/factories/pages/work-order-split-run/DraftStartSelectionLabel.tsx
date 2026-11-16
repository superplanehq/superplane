import type { DraftStartDefaults } from "@/hooks/useDraftStartDefaults";
import { draftStartDefaultsStatus } from "@/hooks/useDraftStartDefaults";
import type { resolveDraftStartSelection } from "./draftStartModel";

export function DraftStartSelectionLabel({
  defaults,
  resolved,
  thinkingLevel,
}: {
  defaults: DraftStartDefaults;
  resolved: ReturnType<typeof resolveDraftStartSelection>;
  thinkingLevel?: string;
}) {
  if (!defaults.values) return <span className="min-w-0 truncate">{draftStartDefaultsStatus(defaults)}</span>;
  return (
    <>
      <span className="min-w-0 truncate" title={resolved.label}>
        {resolved.modelLabel}
      </span>
      <span className="shrink-0" title={thinkingLevel === undefined ? "Implementation default" : "Task override"}>
        · {resolved.thinkingLabel}
      </span>
    </>
  );
}
