import type { ReactNode } from "react";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunStepper, type FirstRunStepperStep } from "./FirstRunStepper";

const copy = FIRST_RUN_COPY.connect;

export type FirstRunGithubStep = "connect" | "organization" | "repository";

type Provider = "github" | "bitbucket";

function connectLabel(provider: Provider): string {
  return provider === "bitbucket" ? FIRST_RUN_COPY.bitbucket.connectAction : copy.connectGitHub;
}

function organizationStep(provider: Provider, organizationName?: string): FirstRunStepperStep<FirstRunGithubStep> {
  if (provider === "bitbucket") {
    return {
      id: "organization",
      label: FIRST_RUN_COPY.bitbucket.stepWorkspace,
      doneLabel: organizationName ? FIRST_RUN_COPY.bitbucket.stepWorkspaceDone(organizationName) : undefined,
    };
  }
  return {
    id: "organization",
    label: copy.stepOrganization,
    doneLabel: organizationName ? copy.stepOrganizationDone(organizationName) : undefined,
  };
}

/**
 * The Git host steps on one card: connect, organization, repository.
 */
export function FirstRunGithubStepper({
  current,
  provider = "github",
  organizationName,
  action,
  children,
}: {
  current: FirstRunGithubStep;
  provider?: Provider;
  /** Names the finished organization row, e.g. "Organization: acme". */
  organizationName?: string;
  /** Control on the active step header, e.g. the Connect button. */
  action?: ReactNode;
  children?: ReactNode;
}) {
  const steps: FirstRunStepperStep<FirstRunGithubStep>[] = [
    { id: "connect", label: connectLabel(provider) },
    organizationStep(provider, organizationName),
    { id: "repository", label: copy.stepRepository },
  ];
  return (
    <FirstRunStepper steps={steps} current={current} action={action} testId="first-run-github-stepper">
      {children}
    </FirstRunStepper>
  );
}
