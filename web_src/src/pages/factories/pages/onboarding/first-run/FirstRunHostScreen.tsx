import {
  GITHUB_ENTERPRISE_VCS_OPTION,
  VCS_OPTIONS,
  type FirstRunHostChoice,
  type FirstRunHostOption,
  type VcsHostId,
} from "../onboardingFixtures";
import { ConnectOptionRow, IntegrationChoiceIcon } from "../onboardingSteps";
import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunHeading, FirstRunShell } from "./FirstRunShell";
import type { FirstRunSphereProps } from "./FirstRunSpherePane";
import type { FirstRunChrome } from "./firstRunTypes";

const copy = FIRST_RUN_COPY.host;

export function FirstRunHostScreen({
  selectedHost,
  githubEnterpriseSelected = false,
  githubEnterpriseLicensed = false,
  chrome,
  sphere,
  onChooseHost,
}: {
  selectedHost: VcsHostId | null;
  githubEnterpriseSelected?: boolean;
  githubEnterpriseLicensed?: boolean;
  chrome?: FirstRunChrome;
  sphere?: FirstRunSphereProps;
  onChooseHost: (host: FirstRunHostChoice) => void;
}) {
  const options: FirstRunHostOption[] = [...VCS_OPTIONS];
  options.splice(1, 0, GITHUB_ENTERPRISE_VCS_OPTION);

  return (
    <FirstRunShell testId="first-run-host" chrome={chrome} sphere={sphere} visual="preview">
      <FirstRunHeading headline={copy.headline}>
        <p className="text-[15px] leading-6 text-muted-foreground">{copy.body}</p>
      </FirstRunHeading>

      <div className="mt-8 space-y-3">
        {options.map((option) => (
          <div key={option.id} data-testid={`first-run-host-${option.id}`}>
            <ConnectOptionRow
              icon={<IntegrationChoiceIcon name={option.id === "github-enterprise" ? "github" : option.id} />}
              title={option.label}
              detail={
                option.id === "github-enterprise" && !githubEnterpriseLicensed
                  ? copy.githubEnterpriseRequiresLicense
                  : option.detail
              }
              selected={
                option.id === "github-enterprise"
                  ? githubEnterpriseSelected
                  : selectedHost === option.id && !githubEnterpriseSelected
              }
              soon={option.soon}
              disabled={option.id === "github-enterprise" && !githubEnterpriseLicensed}
              badge={
                option.id === "github-enterprise" && !githubEnterpriseLicensed ? copy.githubEnterpriseBadge : undefined
              }
              onSelect={() => onChooseHost(option.id)}
            />
          </div>
        ))}
      </div>
    </FirstRunShell>
  );
}
