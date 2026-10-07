import type { FactoriesFactoryPullRequest, FactoriesWorkOrder, FactoriesWorkOrderArtifact } from "@/api-client";
import { ExternalLink } from "lucide-react";
import type { ReactNode } from "react";

import { useFactoriesLayout } from "../layout/factoriesLayoutContext";
import { OutputList } from "../pages/work-order-popup-redesign/popupShared";
import type { SplitRunFixture } from "../pages/work-order-split-run/splitRunMocks";
import { MOBILE_TASK_COPY } from "./mobileCopy";
import { MobileTaskActivity } from "./MobileTaskActivity";

export function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="flex flex-col gap-2">
      <h2 className="workspace-section-title">{title}</h2>
      {children}
    </section>
  );
}

/** Activity, pull requests, and files. The phone page shows them with and without the refine chat. */
export function MobileTaskSections({
  order,
  orderId,
  fixture,
  artifacts,
  pullRequests,
  expandCurrentPhase,
}: {
  order: FactoriesWorkOrder;
  orderId: string;
  fixture: SplitRunFixture;
  artifacts: FactoriesWorkOrderArtifact[];
  pullRequests: FactoriesFactoryPullRequest[];
  expandCurrentPhase: boolean;
}) {
  const { organizationId, factoryId } = useFactoriesLayout();
  return (
    <>
      <Section title={MOBILE_TASK_COPY.activity}>
        <MobileTaskActivity
          organizationId={organizationId}
          factoryId={factoryId}
          orderId={orderId}
          phases={fixture.phases.filter((phase) => !phase.historyRun)}
          expandedPhaseId={expandCurrentPhase ? fixture.currentPhaseId : undefined}
          files={order.files}
        />
      </Section>

      {pullRequests.length > 0 ? (
        <Section title={MOBILE_TASK_COPY.pullRequests}>
          <PullRequestList pullRequests={pullRequests} />
        </Section>
      ) : null}

      {artifacts.length > 0 ? (
        <Section title={MOBILE_TASK_COPY.files}>
          <FileList artifacts={artifacts} />
        </Section>
      ) : null}
    </>
  );
}

function PullRequestList({ pullRequests }: { pullRequests: FactoriesFactoryPullRequest[] }) {
  return (
    <ul className="flex flex-col gap-2">
      {pullRequests.map((pullRequest) => (
        <li key={pullRequest.id ?? pullRequest.url}>
          <a
            href={pullRequest.url}
            target="_blank"
            rel="noreferrer"
            className="flex items-start gap-2 rounded-lg border border-border bg-card px-3 py-2.5 text-[13px] text-foreground"
          >
            <span className="min-w-0 flex-1 break-words">
              <span className="block font-medium">
                {pullRequest.title || `Pull request #${pullRequest.number ?? ""}`}
              </span>
              <span className="block text-[12px] text-muted-foreground">
                {[
                  pullRequest.repository,
                  pullRequest.number ? `#${pullRequest.number}` : null,
                  pullRequestStateLabel(pullRequest),
                ]
                  .filter(Boolean)
                  .join(" · ")}
              </span>
            </span>
            <ExternalLink className="mt-0.5 size-3.5 shrink-0 text-muted-foreground" aria-hidden />
          </a>
        </li>
      ))}
    </ul>
  );
}

function pullRequestStateLabel(pullRequest: FactoriesFactoryPullRequest): string | null {
  if (pullRequest.state === "STATE_MERGED") return "Merged";
  if (pullRequest.state === "STATE_CLOSED") return "Closed";
  if (pullRequest.state === "STATE_DRAFT") return "Draft";
  if (pullRequest.state === "STATE_OPEN") return "Open";
  return null;
}

function FileList({ artifacts }: { artifacts: FactoriesWorkOrderArtifact[] }) {
  return (
    <div className="rounded-lg border border-border bg-card px-3">
      <OutputList artifacts={artifacts} />
    </div>
  );
}
