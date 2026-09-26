import type { FactoriesFactoryPullRequest, FactoriesWorkOrderArtifact } from "@/api-client";
import { cn } from "@/lib/utils";

import { toArtifactDataRecord } from "../../../lib/workOrderArtifact";
import type { WorkOrderCheckPresentation } from "../../../lib/workOrderChecks";
import { WorkOrderArtifactInline } from "../../../WorkOrderArtifactInline";
import { WorkOrderPullRequestInline } from "../../../WorkOrderPullRequestInline";
import { SplitRunCheckPills } from "../SplitRunReview";

const CHIP_CLASSNAME =
  "inline-flex h-7 max-w-full items-center rounded-full border border-border/80 bg-background px-2.5 [&>*]:text-[12.5px]";

/**
 * What a stage produced: pull requests, artifacts, and checks. Same inline
 * components as the current tab, so outputs look the same in every variant.
 */
export function StageOutputs({
  pullRequests = [],
  artifacts = [],
  checks = [],
  className,
  testId,
}: {
  pullRequests?: FactoriesFactoryPullRequest[];
  artifacts?: FactoriesWorkOrderArtifact[];
  checks?: WorkOrderCheckPresentation[];
  className?: string;
  testId?: string;
}) {
  if (pullRequests.length === 0 && artifacts.length === 0 && checks.length === 0) {
    return null;
  }
  return (
    <div className={cn("flex min-w-0 flex-wrap items-center gap-1.5", className)} data-testid={testId}>
      {pullRequests.map((pullRequest) => (
        <span key={pullRequest.id ?? pullRequest.url} className={CHIP_CLASSNAME}>
          <WorkOrderPullRequestInline pullRequest={pullRequest} />
        </span>
      ))}
      {artifacts.map((artifact) => (
        <span key={artifact.id ?? artifact.type} className={CHIP_CLASSNAME}>
          <WorkOrderArtifactInline
            artifact={{ id: artifact.id, type: artifact.type ?? "", data: toArtifactDataRecord(artifact.data) }}
          />
        </span>
      ))}
      {checks.length > 0 ? (
        <SplitRunCheckPills checks={checks} testId={testId ? `${testId}-checks` : undefined} />
      ) : null}
    </div>
  );
}
