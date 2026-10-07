import type { WizardStepId } from "./onboardingFixtures";

export function onboardingStepPath(basePath: string, step: WizardStepId): string {
  const [pathname, search = ""] = basePath.split("?");
  const searchParams = new URLSearchParams(search);
  searchParams.set("step", step);

  searchParams.delete("pick");
  searchParams.delete("githubSetup");
  searchParams.delete("githubOrg");
  searchParams.delete("githubIntegrationId");

  return `${pathname}?${searchParams.toString()}`;
}
