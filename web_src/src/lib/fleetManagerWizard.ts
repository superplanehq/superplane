export function singleHostInstall(installationType: string | undefined): boolean {
  return installationType === "single-host";
}

export function fleetProviderChoiceCopy(): { title: string; description: string } {
  return {
    title: "Choose where runners run",
    description: "Select the cloud account that will run fleet machines.",
  };
}

export type FleetManagerWizardStep = "provider" | "fleets" | "values" | "install" | "docker";

// Older single-host setup sessions can contain the removed Docker step.
export function omittedFleetWizardStep(
  step: FleetManagerWizardStep,
  _installationType: string | undefined,
): FleetManagerWizardStep | null {
  if (step === "docker") return "provider";
  return null;
}

export const FLEET_MANAGER_WIZARD_STEPS: FleetManagerWizardStep[] = ["provider", "fleets", "values", "install"];
