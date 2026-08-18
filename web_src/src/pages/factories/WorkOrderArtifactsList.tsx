import type { FactoriesWorkOrderArtifact } from "@/api-client";

import { toArtifactDataRecord } from "./lib/workOrderArtifact";
import { WorkOrderArtifactInline } from "./WorkOrderArtifactInline";

interface WorkOrderArtifactsListProps {
  artifacts: FactoriesWorkOrderArtifact[];
  isLoading: boolean;
  error?: Error | null;
}

export function WorkOrderArtifactsList({ artifacts, isLoading, error }: WorkOrderArtifactsListProps) {
  return (
    <section>
      <h3 className="workspace-section-label">Artifacts</h3>

      <div className="mt-2">
        {error ? (
          <p className="text-[13px] text-destructive">Failed to load artifacts.</p>
        ) : isLoading ? (
          <p className="text-[13px] text-muted-foreground">Loading artifacts…</p>
        ) : artifacts.length === 0 ? (
          <p className="text-[13px] text-muted-foreground">
            No artifacts yet. Automation nodes will attach PRs and notes here as they run.
          </p>
        ) : (
          <ul>
            {artifacts.map((artifact) => (
              <li className="flex items-center py-1.5" key={artifact.id ?? `${artifact.type}-${artifact.createdAt}`}>
                <WorkOrderArtifactInline
                  className="w-full justify-start"
                  artifact={{
                    id: artifact.id,
                    type: artifact.type ?? "TYPE_UNSPECIFIED",
                    data: toArtifactDataRecord(artifact.data),
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
