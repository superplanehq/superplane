import type { resolveDraftStartSelection } from "./draftStartModel";

export function DraftStartSelectionLabel({
  resolved,
  thinkingLevel,
}: {
  resolved: ReturnType<typeof resolveDraftStartSelection>;
  thinkingLevel?: string;
}) {
  if (resolved.hasOnlyUnresolvedDefaults) return <span className="min-w-0 truncate">{resolved.label}</span>;
  return (
    <>
      <span className="min-w-0 truncate" title={resolved.label}>
        {resolved.modelLabel}
      </span>
      <span
        className={resolved.isThinkingUnresolved ? "min-w-0 truncate" : "shrink-0"}
        title={thinkingLevel === undefined ? "Implementation default" : "Task override"}
      >
        · {resolved.thinkingLabel}
      </span>
    </>
  );
}
