import { VCS_OPTIONS, type VcsHostId } from "../onboardingFixtures";
import { ConnectOptionRow, IntegrationChoiceIcon } from "../onboardingSteps";
import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunHeading, FirstRunShell } from "./FirstRunShell";
import type { FirstRunSphereProps } from "./FirstRunSpherePane";
import type { FirstRunChrome } from "./firstRunTypes";

const copy = FIRST_RUN_COPY.host;

export function FirstRunHostScreen({
  selectedHost,
  chrome,
  sphere,
  onChooseHost,
}: {
  selectedHost: VcsHostId | null;
  chrome?: FirstRunChrome;
  sphere?: FirstRunSphereProps;
  onChooseHost: (host: VcsHostId) => void;
}) {
  return (
    <FirstRunShell testId="first-run-host" chrome={chrome} sphere={sphere} visual="preview">
      <FirstRunHeading headline={copy.headline}>
        <p className="text-[15px] leading-6 text-muted-foreground">{copy.body}</p>
      </FirstRunHeading>

      <div className="mt-8 space-y-3">
        {VCS_OPTIONS.map((option) => (
          <div key={option.id} data-testid={`first-run-host-${option.id}`}>
            <ConnectOptionRow
              icon={<IntegrationChoiceIcon name={option.id} />}
              title={option.label}
              detail={option.detail}
              selected={selectedHost === option.id}
              soon={option.soon}
              onSelect={() => onChooseHost(option.id)}
            />
          </div>
        ))}
      </div>
    </FirstRunShell>
  );
}
