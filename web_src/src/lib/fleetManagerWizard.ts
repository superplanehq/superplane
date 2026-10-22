export function singleHostInstall(installationType: string | undefined): boolean {
  return installationType === "single-host";
}

export function fleetProviderChoiceCopy(singleHost: boolean): { title: string; description: string } {
  if (singleHost) {
    return {
      title: "Runners on this VM",
      description: "Fleet Manager starts runner containers on this machine.",
    };
  }
  return {
    title: "Choose where runners run",
    description: "Select the cloud account that will run fleet machines.",
  };
}

export type FleetManagerWizardStep = "provider" | "fleets" | "values" | "install" | "docker";

// Cloud installs skip the This VM command. Single-host installs skip the cloud chart steps.
export function omittedFleetWizardStep(
  step: FleetManagerWizardStep,
  installationType: string | undefined,
): FleetManagerWizardStep | null {
  if (singleHostInstall(installationType)) {
    if (step === "provider" || step === "docker") return null;
    return "provider";
  }
  if (step === "docker") return "provider";
  return null;
}

export const FLEET_MANAGER_WIZARD_STEPS: FleetManagerWizardStep[] = ["provider", "fleets", "values", "install"];
