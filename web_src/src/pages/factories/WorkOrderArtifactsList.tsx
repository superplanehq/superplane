import type { FactoriesWorkOrderArtifact } from "@/api-client";
import { useRevealAfterPending } from "@/hooks/useRevealAfterPending";
import { cn } from "@/lib/utils";

import { LOADING_REVEAL_CLASSNAME } from "./lib/loadingReveal";
import { toArtifactDataRecord } from "./lib/workOrderArtifact";
import { withoutPullRequestArtifacts } from "./lib/workOrderPullRequest";
import { WorkOrderArtifactInline } from "./WorkOrderArtifactInline";
import { WorkOrderListSkeleton } from "./WorkOrderListSkeleton";

interface WorkOrderArtifactsListProps {
  artifacts: FactoriesWorkOrderArtifact[];
  isLoading: boolean;
  error?: Error | null;
}

export function WorkOrderArtifactsList({ artifacts, isLoading, error }: WorkOrderArtifactsListProps) {
  const visibleArtifacts = withoutPullRequestArtifacts(artifacts);
  const reveal = useRevealAfterPending(isLoading);
  return (
    <section>
      <h3 className="workspace-section-label">Artifacts</h3>

      <div className="mt-2">
        {error ? (
          <p className="text-[13px] text-destructive">Failed to load artifacts.</p>
        ) : isLoading ? (
          <WorkOrderListSkeleton label="Loading artifacts" />
        ) : visibleArtifacts.length === 0 ? (
          <p className={cn("text-[13px] text-muted-foreground", reveal && LOADING_REVEAL_CLASSNAME)}>
            No artifacts yet. Automation nodes will attach notes, files, and links here as they run.
          </p>
        ) : (
          <ul className={cn(reveal && LOADING_REVEAL_CLASSNAME)} data-reveal={reveal ? "" : undefined}>
            {visibleArtifacts.map((artifact) => (
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
