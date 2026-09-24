import { GITHUB_SETUP_ORG_PARAM, GITHUB_SETUP_REQUEST_PARAM } from "@/lib/integrationSetupReturn";

import type { WizardStepId } from "./onboardingFixtures";

export function onboardingStepPath(basePath: string, step: WizardStepId): string {
  const [pathname, search = ""] = basePath.split("?");
  const searchParams = new URLSearchParams(search);
  searchParams.set("step", step);

  if (step === "vcs") {
    searchParams.set("pick", "newest");
  } else {
    searchParams.delete("pick");
    // One-shot GitHub return flags belong on the connect screen only.
    // Keeping them on later steps remounts Connect and rechecks a request
    // that already finished.
    searchParams.delete(GITHUB_SETUP_REQUEST_PARAM);
    searchParams.delete(GITHUB_SETUP_ORG_PARAM);
  }

  return `${pathname}?${searchParams.toString()}`;
}
