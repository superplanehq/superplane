import type { WizardStepId } from "./onboardingFixtures";

export function onboardingStepPath(basePath: string, step: WizardStepId): string {
  const [pathname, search = ""] = basePath.split("?");
  const searchParams = new URLSearchParams(search);
  searchParams.set("step", step);

  if (step === "vcs") {
    searchParams.set("pick", "newest");
  } else {
    searchParams.delete("pick");
  }

  return `${pathname}?${searchParams.toString()}`;
}
