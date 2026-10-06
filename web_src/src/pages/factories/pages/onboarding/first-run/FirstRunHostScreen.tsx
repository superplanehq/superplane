import { VCS_OPTIONS, type VcsHostId } from "../onboardingFixtures";
import { ConnectOptionRow, IntegrationChoiceIcon } from "../onboardingSteps";
import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunHeading, FirstRunShell } from "./FirstRunShell";
import type { FirstRunSphereProps } from "./FirstRunSpherePane";
import type { FirstRunChrome } from "./firstRunTypes";

const copy = FIRST_RUN_COPY.host;

export function FirstRunHostScreen({
  selectedHost,
  bitbucketAvailable = false,
  bitbucketFeatureLoading = false,
  chrome,
  sphere,
  onChooseHost,
}: {
  selectedHost: VcsHostId | null;
  bitbucketAvailable?: boolean;
  bitbucketFeatureLoading?: boolean;
  chrome?: FirstRunChrome;
  sphere?: FirstRunSphereProps;
  onChooseHost: (host: VcsHostId) => void;
}) {
  return (
    <FirstRunShell testId="first-run-host" chrome={chrome} sphere={sphere}>
      <FirstRunHeading headline={copy.headline}>
        <p className="text-[15px] leading-6 text-muted-foreground">{copy.body}</p>
      </FirstRunHeading>

      <div className="mt-8 space-y-3">
        {VCS_OPTIONS.map((option) => {
          const bitbucket = option.id === "bitbucket";
          const waiting = bitbucket && bitbucketFeatureLoading;
          const unavailable = bitbucket && !bitbucketFeatureLoading && !bitbucketAvailable;
          return (
            <div key={option.id} data-testid={`first-run-host-${option.id}`}>
              <ConnectOptionRow
                icon={<IntegrationChoiceIcon name={option.id} />}
                title={option.label}
                detail={hostOptionDetail(option.detail, waiting, unavailable)}
                selected={selectedHost === option.id}
                soon={option.soon || unavailable}
                disabled={waiting}
                onSelect={() => onChooseHost(option.id)}
              />
            </div>
          );
        })}
      </div>
    </FirstRunShell>
  );
}

function hostOptionDetail(detail: string, waiting: boolean, unavailable: boolean): string {
  if (waiting) return copy.bitbucketLookupLoading;
  if (unavailable) return copy.bitbucketSoon;
  return detail;
}
