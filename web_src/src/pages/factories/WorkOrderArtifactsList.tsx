import type { FactoriesWorkOrderArtifact } from "@/api-client";

import { WorkOrderArtifactInline } from "./WorkOrderArtifactInline";

interface WorkOrderArtifactsListProps {
  artifacts: FactoriesWorkOrderArtifact[];
  isLoading: boolean;
  error?: Error | null;
}

/**
 * Bare artifacts list for the work order sidebar. Renders as an unstyled
 * `<ul>` with `icon + name` rows — no card container, no timestamp caption —
 * to match the mockup's compact metadata treatment.
 */
export function WorkOrderArtifactsList({ artifacts, isLoading, error }: WorkOrderArtifactsListProps) {
  return (
    <section>
      <h3 className="workspace-section-label">Artifacts</h3>

      <div className="mt-2">
        {error ? (
          <p className="text-[13px] text-red-500 dark:text-red-400">Failed to load artifacts.</p>
        ) : isLoading ? (
          <p className="text-[13px] text-muted-foreground">Loading artifacts…</p>
        ) : artifacts.length === 0 ? (
          <p className="text-[13px] text-muted-foreground">
            No artifacts yet. Automation nodes will attach PRs and notes here as they run.
          </p>
        ) : (
          <ul>
            {artifacts.map((artifact) => (
              <li className="py-1.5" key={artifact.id ?? `${artifact.type}-${artifact.createdAt}`}>
                <WorkOrderArtifactInline
                  artifact={{
                    id: artifact.id,
                    type: artifact.type ?? "TYPE_UNSPECIFIED",
                    data: toDataRecord(artifact.data),
                  }}
                />
              </li>
            ))}
          </ul>
        )}
      </div>
    </section>
  );
}

function toDataRecord(data: unknown): Record<string, unknown> | undefined {
  return data && typeof data === "object" ? (data as Record<string, unknown>) : undefined;
}
